// Everything you need to develop the Dagger Engine
// https://dagger.io
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/platforms"
	"github.com/dagger/dagger/.dagger/internal/dagger"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
)

// A dev environment for the DaggerDev Engine
type DaggerDev struct {
	Source  *dagger.Directory // +private
	Tag     string
	Version string

	// When set, module codegen is automatically applied when retrieving the Dagger source code
	ModCodegen        bool
	ModCodegenTargets []string

	// Can be used by nested clients to forward docker credentials to avoid
	// rate limits
	DockerCfg *dagger.Secret // +private

	// +private
	GitRef string
	GitDir *dagger.Directory
}

func New(
	ctx context.Context,
	// +optional
	// +defaultPath="/"
	// +ignore=["bin", ".git", "**/node_modules", "**/.venv", "**/__pycache__"]
	source *dagger.Directory,

	// Git directory, for metadata introspection
	// +optional
	// +defaultPath="/"
	// +ignore=["!.git"]
	gitDir *dagger.Directory,

	// +optional
	version string,
	// +optional
	tag string,

	// +optional
	dockerCfg *dagger.Secret,

	// Git ref (used for test-publish checks)
	// +optional
	ref string,

	// Re-generate all dagger modules
	// +optional
	modCodegen bool,
) (*DaggerDev, error) {
	v, err := dag.VersionInfo(source, version).String(ctx)
	if err != nil {
		return nil, err
	}
	if modCodegen {
		source = dag.Supermod(source).
			DevelopAll(dagger.SupermodDevelopAllOpts{Exclude: []string{
				"docs/.*",
				"core/integration/.*",
			}}).Source()
	}
	if ref == "" {
		// FIXME: this doesn't always work in github actions
		ref, err := dag.
			Wolfi().
			Container(dagger.WolfiContainerOpts{Packages: []string{"git"}}).
			WithMountedDirectory("/src", gitDir).
			WithWorkdir("/src").
			WithMountedFile("/bin/get-ref.sh", dag.CurrentModule().Source().File("get-ref.sh")).
			WithExec([]string{"sh", "get-ref.sh"}).
			Stdout(ctx)
		if err != nil {
			return nil, err
		}
		ref = strings.TrimRight(ref, "\n")
	}
	return &DaggerDev{
		Source:    source,
		Version:   v,
		Tag:       tag,
		DockerCfg: dockerCfg,
		GitRef:    ref,
		GitDir:    gitDir,
	}, nil
}

// Wrap 3 SDK-specific checks into a single check
type SDKChecks interface {
	Lint(ctx context.Context) error
	Test(ctx context.Context) error
	TestPublish(ctx context.Context, tag string) error
}

func (dev *DaggerDev) sdkCheck(sdk string) Check {
	var checks SDKChecks
	switch sdk {
	case "python":
		checks = &PythonSDK{Dagger: dev}
	case "go":
		checks = NewGoSDK(dev.Source(), dev.Engine())
	case "typescript":
		checks = &TypescriptSDK{Dagger: dev}
	case "php":
		checks = &PHPSDK{Dagger: dev}
	case "java":
		checks = &JavaSDK{Dagger: dev}
	case "rust":
		checks = &RustSDK{Dagger: dev}
	case "elixir":
		checks = &ElixirSDK{Dagger: dev}
	}
	return func(ctx context.Context) (rerr error) {
		lint := func() (rerr error) {
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("lint sdk/%s", sdk))
			defer func() {
				if rerr != nil {
					span.SetStatus(codes.Error, rerr.Error())
				}
				span.End()
			}()
			return checks.Lint(ctx)
		}
		test := func() (rerr error) {
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("test sdk/%s", sdk))
			defer func() {
				if rerr != nil {
					span.SetStatus(codes.Error, rerr.Error())
				}
				span.End()
			}()
			return checks.Test(ctx)
		}
		testPublish := func() (rerr error) {
			ctx, span := Tracer().Start(ctx, fmt.Sprintf("test-publish sdk/%s", sdk))
			defer func() {
				if rerr != nil {
					span.SetStatus(codes.Error, rerr.Error())
				}
				span.End()
			}()
			// Inspect .git to avoid dependencing on $GITHUB_REF
			ref, err := dev.Ref(ctx)
			if err != nil {
				return fmt.Errorf("failed to introspect git ref: %s", err.Error())
			}
			fmt.Printf("===> ref = \"%s\"\n", ref)
			return checks.TestPublish(ctx, ref)
		}
		if err := lint(); err != nil {
			return err
		}
		if err := test(); err != nil {
			return err
		}
		if err := testPublish(); err != nil {
			return err
		}
		return nil
	}
}

const (
	CheckDocs          = "docs"
	CheckPythonSDK     = "sdk/python"
	CheckGoSDK         = "sdk/go"
	CheckTypescriptSDK = "sdk/typescript"
	CheckPHPSDK        = "sdk/php"
	CheckJavaSDK       = "sdk/java"
	CheckRustSDK       = "sdk/rust"
	CheckElixirSDK     = "sdk/elixir"
)

// Check that everything works. Use this as CI entrypoint.
func (dev *DaggerDev) Check(ctx context.Context,
	// Directories to check
	// +optional
	targets []string,
) error {
	var routes = map[string]Check{
		CheckDocs:          (&Docs{Dagger: dev}).Lint,
		CheckPythonSDK:     dev.sdkCheck("python"),
		CheckGoSDK:         dev.sdkCheck("go"),
		CheckTypescriptSDK: dev.sdkCheck("typescript"),
		CheckPHPSDK:        dev.sdkCheck("php"),
		CheckJavaSDK:       dev.sdkCheck("java"),
		CheckRustSDK:       dev.sdkCheck("rust"),
		CheckElixirSDK:     dev.sdkCheck("elixir"),
	}
	if len(targets) == 0 {
		targets = make([]string, 0, len(routes))
		for key := range routes {
			targets = append(targets, key)
		}
	}
	for _, target := range targets {
		if _, exists := routes[target]; !exists {
			return fmt.Errorf("no such target: %s", target)
		}
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, target := range targets {
		check := routes[target]
		eg.Go(func() error { return check(ctx) })
	}
	return eg.Wait()
}

// Develop Dagger SDKs
func (dev *DaggerDev) SDK() *SDK {
	return &SDK{
		Go:         NewGoSDK(dev.Src, dev.Engine()),
		Python:     &PythonSDK{Dagger: dev},
		Typescript: &TypescriptSDK{Dagger: dev},
		Elixir:     &ElixirSDK{Dagger: dev},
		Rust:       &RustSDK{Dagger: dev},
		PHP:        &PHPSDK{Dagger: dev},
		Java:       &JavaSDK{Dagger: dev},
	}
}

// Creates a dev container that has a running CLI connected to a dagger engine
func (dev *DaggerDev) Dev(
	ctx context.Context,
	// Mount a directory into the container's workdir, for convenience
	// +optional
	target *dagger.Directory,
	// Set target distro
	// +optional
	image *Distro,
	// Enable experimental GPU support
	// +optional
	gpuSupport bool,
) (*dagger.Container, error) {
	if target == nil {
		target = dag.Directory()
	}

	svc, err := dev.
		Engine().
		WithImage(image).
		WithGpuSupport(gpuSupport).
		Service(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, err := svc.Endpoint(ctx, dagger.ServiceEndpointOpts{Scheme: "tcp"})
	if err != nil {
		return nil, err
	}

	client, err := dev.CLI().Binary(ctx, "")
	if err != nil {
		return nil, err
	}

	return dev.Go().Env().
		WithMountedDirectory("/mnt", target).
		WithMountedFile("/usr/bin/dagger", client).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_CLI_BIN", "/usr/bin/dagger").
		WithServiceBinding("dagger-engine", svc).
		WithEnvVariable("_EXPERIMENTAL_DAGGER_RUNNER_HOST", endpoint).
		WithWorkdir("/mnt"), nil
}

// Creates an static dev build
func (dev *DaggerDev) DevExport(
	ctx context.Context,
	// +optional
	platform dagger.Platform,

	// +optional
	race bool,
	// +optional
	trace bool,

	// Set target distro
	// +optional
	image *Distro,
	// Enable experimental GPU support
	// +optional
	gpuSupport bool,
) (*dagger.Directory, error) {
	var platformSpec platforms.Platform
	if platform == "" {
		platformSpec = platforms.DefaultSpec()
	} else {
		var err error
		platformSpec, err = platforms.Parse(string(platform))
		if err != nil {
			return nil, err
		}
	}

	engine := dev.Engine()
	if race {
		engine = engine.WithRace()
	}
	if trace {
		engine = engine.WithTrace()
	}
	enginePlatformSpec := platformSpec
	enginePlatformSpec.OS = "linux"
	engineCtr, err := engine.
		WithPlatform(dagger.Platform(platforms.Format(enginePlatformSpec))).
		WithImage(image).
		WithGpuSupport(gpuSupport).
		Container(ctx)
	if err != nil {
		return nil, err
	}
	engineTar := engineCtr.AsTarball(dagger.ContainerAsTarballOpts{
		// use gzip to avoid incompatibility w/ older docker versions
		ForcedCompression: dagger.Gzip,
	})

	cli := dev.CLI()
	cliBin, err := cli.Binary(ctx, platform)
	if err != nil {
		return nil, err
	}
	cliPath := "dagger"
	if platformSpec.OS == "windows" {
		cliPath += ".exe"
	}

	dir := dag.Directory().
		WithFile("engine.tar", engineTar).
		WithFile(cliPath, cliBin)
	return dir, nil
}
