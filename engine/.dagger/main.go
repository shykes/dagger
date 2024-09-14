package main

import (
	"context"
	"fmt"

	"dagger/engine/internal/dagger"

	"github.com/dagger/dagger/engine/distconsts"

	"github.com/moby/buildkit/identity"
	"golang.org/x/sync/errgroup"
)

type Distro string

const (
	DistroAlpine = "alpine"
	DistroWolfi  = "wolfi"
	DistroUbuntu = "ubuntu"
)

func New(
	ctx context.Context,
	// +optional
	// +defaultPath="/"
	// +ignore=["*", "!**.go", "!**/go.mod", "!**/go.sum", "!**.graphqls", "!**.proto", "!**.json", "!**.yaml", "!**/testdata", "!**.sql"]
	source *dagger.Directory,
	// Git commit to include in engine version
	// +optional
	commit string,
	// Git tag to include in engine version, in short format
	// +optional
	tag string,
	// Custom engine config values
	// +optional
	config []string,
	// +optional
	args []string,
	// Build the engine with race checking mode
	// +optional
	race bool,
	// Build the engine with tracing enabled
	// +optional
	trace bool,
	// +optional
	// Set an instance name, to spawn different instances of the service, each
	// with their own lifecycle and state volume
	instanceName string,
	// +optional
	dockerConfig *dagger.Secret,
	// Build the engine with GPU support
	// +optional
	gpu bool,
	// +optional
	platform dagger.Platform,
	// Choose a flavor of base image
	// +optional
	image *Distro,
) *Engine {
	return &Engine{
		Source:       source,
		Tag:          tag,
		Commit:       commit,
		Config:       config,
		Args:         args,
		Race:         race,
		Trace:        trace,
		InstanceName: instanceName,
		DockerConfig: dockerConfig,
		GPU:          gpu,
		Platform:     platform,
		Image:        image,
	}
}

type Engine struct {
	Args   []string // +private
	Config []string // +private

	Trace bool // +private

	Source       *dagger.Directory // +private
	Commit       string            // +private
	Tag          string            // +private
	Race         bool              // +private
	InstanceName string            // +private
	DockerConfig *dagger.Secret    // +private
	GPU          bool              // +private
	Platform     dagger.Platform   // +private
	Image        *Distro           // +private
}

// Run engine tests
func (engine *Engine) Test(
	ctx context.Context,
	// Packages to test (default all)
	// +optional
	// +default=["./..."]
	pkgs []string,
	// Only run these tests
	// +optional
	run string,
	// Skip these tests
	// +optional
	skip string,
	// Abort test run on first failure
	// +optional
	failfast bool,
	// How many tests to run in parallel - defaults to the number of CPUs
	// +optional
	parallel int,
	// How long before timing out the test run
	// +optional
	timeout string,
	// +optional
	race bool,
	// +default=1
	// +optional
	count int,
) error {
	return dag.Go(engine.Source).Test(ctx, dagger.GoTestOpts{
		Pkgs:     pkgs,
		Run:      run,
		Skip:     skip,
		Failfast: failfast,
		Parallel: parallel,
		Timeout:  timeout,
		Race:     race,
		Count:    count,
	})
}

// List all engine tests, using 'go test -list=.'
func (e *Engine) Tests(
	ctx context.Context,
	// Packages to include in the test list
	// +optional
	pkgs []string,
) (string, error) {
	return dag.Go(e.Source).Tests(ctx, dagger.GoTestsOpts{
		Pkgs: pkgs,
	})
}

// Build the engine container
func (e *Engine) Container(
	ctx context.Context,
	// Build a dev container, with additional configuration for e2e testing
	// +optional
	dev bool,
	// Scan the container for vulnerabilities after building it
	// +optional
	scan bool,
	// Config files used by the vulnerability scanner
	// +optional
	// +defaultPath="."
	// +ignore=["*", "!.trivyignore", "!trivyignore.yml", "!trivyignore.yaml"]
	scanConfig *dagger.Directory,
) (*dagger.Container, error) {
	if dev {
		e.Config = append(e.Config, `grpc=address=["unix:///var/run/buildkit/buildkitd.sock", "tcp://0.0.0.0:1234"]`)
		e.Args = append(e.Args, `network-name=dagger-dev`, `network-cidr=10.88.0.0/16`)
	}
	cfg, err := generateConfig(e.Trace, e.Config)
	if err != nil {
		return nil, err
	}
	entrypoint, err := generateEntrypoint(e.Args)
	if err != nil {
		return nil, err
	}
	builder, err := newBuilder(ctx, e.Source)
	if err != nil {
		return nil, err
	}
	builder = builder.
		WithVersion(e.Version).
		WithTag(e.Tag).
		WithRace(e.Race)
	if e.Platform != "" {
		builder = builder.WithPlatform(e.Platform)
	}
	if e.Image != nil {
		switch *e.Image {
		case DistroAlpine:
			builder = builder.WithAlpineBase()
		case DistroWolfi:
			builder = builder.WithWolfiBase()
		case DistroUbuntu:
			builder = builder.WithUbuntuBase()
		default:
			return nil, fmt.Errorf("unknown base image type %s", *e.Image)
		}
	}
	if e.GPU {
		builder = builder.WithGPUSupport()
	}
	ctr, err := builder.Engine(ctx)
	if err != nil {
		return nil, err
	}
	ctr = ctr.
		WithFile("/etc/dagger/engine.toml", cfg).
		WithFile("/usr/local/bin/dagger-entrypoint.sh", entrypoint).
		WithEntrypoint([]string{"dagger-entrypoint.sh"}).
		WithFile(
			"/usr/local/bin/dagger",
			dag.DaggerCli().Binary(dagger.DaggerCliBinaryOpts{
				Platform: e.Platform,
				Version:  e.Version,
				Tag:      e.Tag,
			})).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_RUNNER_HOST", "unix:///var/run/buildkit/buildkitd.sock")
	if dev {
		ctr = ctr.
			WithExposedPort(1234, dagger.ContainerWithExposedPortOpts{Protocol: dagger.Tcp}).
			WithMountedCache(
				distconsts.EngineDefaultStateDir,
				e.cacheVolume(),
				dagger.ContainerWithMountedCacheOpts{
					// only one engine can run off it's local state dir at a time; Private means that we will attempt to re-use
					// these cache volumes if they are not already locked to another running engine but otherwise will create a new
					// one, which gets us best-effort cache re-use for these nested engine services
					Sharing: dagger.Private,
				}).
			WithExec(nil, dagger.ContainerWithExecOpts{
				UseEntrypoint:            true,
				InsecureRootCapabilities: true,
			})
	}
	if scan {
		if _, err := e.Scan(ctx, scanConfig, ctr); err != nil {
			return ctr, err
		}
	}
	return ctr, nil
}

// Instantiate the engine as a service, and bind it to the given client
func (e *Engine) Bind(ctx context.Context, client *dagger.Container) *dagger.Container {
	return client.
		With(func(c *dagger.Container) *dagger.Container {
			ectr, err := e.Container(ctx, true, false, nil)
			if err != nil {
				return c.
					WithEnvVariable("ERR", err.Error()).
					WithExec([]string{"sh", "-c", "echo $ERR >/dev/stderr; exit 1"})
			}
			return c.WithServiceBinding("dagger-engine", ectr.AsService())
		}).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_RUNNER_HOST", "tcp://dagger-engine:1234").
		WithMountedFile("/.dagger-cli", dag.DaggerCli().Binary(dagger.DaggerCliBinaryOpts{
			Platform: e.Platform,
			Version:  e.Version,
			Tag:      e.Tag,
		})).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/.dagger-cli").
		WithExec([]string{"ln", "-s", "/.dagger-cli", "/usr/local/bin/dagger"}).
		With(func(c *dagger.Container) *dagger.Container {
			if e.DockerConfig != nil {
				// this avoids rate limiting in our ci tests
				return c.WithMountedSecret("/root/.docker/config.json", e.DockerConfig)
			}
			return c
		})
}

func (e *Engine) cacheVolume() *dagger.CacheVolume {
	var name string
	if e.Version != "" {
		name = "dagger-dev-engine-state-" + e.Version
	} else {
		name = "dagger-dev-engine-state-" + identity.NewID()
	}
	if e.InstanceName != "" {
		name += "-" + e.InstanceName
	}
	return dagger.Connect().CacheVolume(name)
}

// Lint the engine source code
func (e *Engine) Lint(
	ctx context.Context,
) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		exclude := []string{"docs/.*", "core/integration/.*"}
		// Run dagger module codegen recursively before linting
		src := dag.Supermod(e.Source).
			DevelopAll(dagger.SupermodDevelopAllOpts{Exclude: exclude}).
			Source()
		// Lint each go module
		pkgs, err := dag.Dirdiff().
			Find(ctx, src, "go.mod", dagger.DirdiffFindOpts{Exclude: exclude})
		if err != nil {
			return err
		}
		return dag.Go(src).Lint(ctx, dagger.GoLintOpts{Packages: pkgs})
	})
	eg.Go(func() error {
		return e.LintGenerate(ctx)
	})

	return eg.Wait()
}

func (e *Engine) Env() *dagger.Container {
	return dag.Go(e.Source).Env()
}

// Generate any engine-related files
// Note: this is codegen of the 'go generate' variety, not 'dagger develop'
func (e *Engine) Generate() *dagger.Directory {
	return e.Env().
		WithoutDirectory("sdk"). // sdk generation happens separately
		// protobuf dependencies
		WithExec([]string{"apk", "add", "protoc=~3.21.12"}). // FIXME: use common apko module
		WithExec([]string{"go", "install", "google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2"}).
		WithExec([]string{"go", "install", "github.com/gogo/protobuf/protoc-gen-gogoslick@v1.3.2"}).
		WithExec([]string{"go", "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0"}).
		WithExec([]string{"go", "generate", "-v", "./..."}).
		Directory(".")
}

// Lint any generated engine-related files
func (e *Engine) LintGenerate(ctx context.Context) error {
	return dag.Dirdiff().AssertEqual(
		ctx,
		e.Env().WithoutDirectory("sdk").Directory("."),
		e.Generate(),
		[]string{"."},
	)
}

func (e *Engine) Scan(
	ctx context.Context,
	// Trivy config files
	// +optional
	// +defaultPath="."
	// +ignore=["*", "!.trivyignore", "!trivyignore.yml", "!trivyignore.yaml"]
	ignoreFiles *dagger.Directory,
	// The container to scan
	target *dagger.Container,
) (string, error) {
	ignoreFileNames, err := ignoreFiles.Entries(ctx)
	if err != nil {
		return "", err
	}
	// FIXME: trivy module
	ctr := dag.Container().
		From("aquasec/trivy:0.50.4").
		WithMountedFile("/mnt/engine.tar", target.AsTarball()).
		WithMountedDirectory("/mnt/ignores", ignoreFiles).
		WithMountedCache("/root/.cache/", dag.CacheVolume("trivy-cache"))
	args := []string{
		"trivy",
		"image",
		"--format=json",
		"--no-progress",
		"--exit-code=1",
		"--vuln-type=os,library",
		"--severity=CRITICAL,HIGH",
		"--show-suppressed",
	}
	if len(ignoreFileNames) > 0 {
		args = append(args, "--ignorefile=/mnt/ignores/"+ignoreFileNames[0])
	}
	args = append(args, "--input", "/mnt/engine.tar")
	return ctr.WithExec(args).Stdout(ctx)
}
