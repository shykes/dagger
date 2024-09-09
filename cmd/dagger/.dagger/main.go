package main

import (
	"context"
	"dagger/dagger/internal/dagger"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"
)

const (
	// https://github.com/goreleaser/goreleaser/releases
	goReleaserVersion = "v2.2.0"
)

type DaggerCli struct{}

func (cli DaggerCli) Build(
	// +optional
	// +defaultPath="/"
	// +ignore_0.13=["!/cmd/dagger/*", "!**/go.sum", "!**/go.mod", "!**/*.go", "!.git", ".git/objects/*", "!.changes"]
	// stopgap:
	// +ignore=["bin", "**/node_modules", "**/.venv", "**/__pycache__"]
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	tag string,
	// +optional
	platform dagger.Platform,
) *dagger.Directory {
	return dag.Go(source).
		Build(dagger.GoBuildOpts{
			Platform: platform,
			Pkgs:     []string{"./cmd/dagger"},
			Values: []string{
				"github.com/dagger/dagger/engine.Version=" + version,
				"github.com/dagger/dagger/engine.Tag=" + tag,
			},
			NoSymbols: true,
			NoDwarf:   true,
		})
}

func (cli DaggerCli) Binary(
	ctx context.Context,

	// +optional
	// +defaultPath="/"
	// +ignore_0.13=["!/cmd/dagger/*", "!**/go.sum", "!**/go.mod", "!**/*.go", "!.git", ".git/objects/*", "!.changes"]
	// stopgap:
	// +ignore=["bin", "**/node_modules", "**/.venv", "**/__pycache__"]
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	tag string,
	// +optional
	// +default="current"
	platform dagger.Platform,
) (*dagger.File, error) {
	bin := cli.
		Build(source, version, tag, platform).
		Directory("bin")
	// The binary might be called dagger or dagger.exe
	files, err := bin.
		Glob(ctx, "dagger*")
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("No matching binary in %q", files)
	}
	return bin.File(files[0]), nil
}

// Publish the CLI using GoReleaser
func (cli DaggerCli) Publish(
	ctx context.Context,
	// +optional
	// +defaultPath="/"
	// +ignore_0.13=["!/cmd/dagger/*", "!**/go.sum", "!**/go.mod", "!**/*.go", "!.git", ".git/objects/*", "!.changes"]
	// stopgap:
	// +ignore=["bin", "**/node_modules", "**/.venv", "**/__pycache__"]
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	tag string,
	githubOrgName string,
	githubToken *dagger.Secret,
	goreleaserKey *dagger.Secret,
	awsAccessKeyID *dagger.Secret,
	awsSecretAccessKey *dagger.Secret,
	awsRegion *dagger.Secret,
	awsBucket *dagger.Secret,
	artefactsFQDN string,
) error {
	ctr := cli.Goreleaser().WithMountedDirectory("", source)
	// Verify tag
	_, err := ctr.WithExec([]string{"git", "show-ref", "--verify", "refs/tags/" + tag}).Sync(ctx)
	if err != nil {
		err, ok := err.(*ExecError)
		if !ok || !strings.Contains(err.Stderr, "not a valid ref") {
			return err
		}
		// clear the set tag
		tag = ""
		// goreleaser refuses to run if there isn't a tag, so set it to a dummy but valid semver
		ctr = ctr.WithExec([]string{"git", "tag", "0.0.0"})
	}
	args := []string{"release", "--clean", "--skip=validate", "--verbose"}
	if tag != "" {
		args = append(args, "--release-notes", fmt.Sprintf(".changes/%s.md", tag))
	} else {
		// if this isn't an official semver version, do a dev release
		args = append(args,
			"--nightly",
			"--config", ".goreleaser.nightly.yml",
		)
	}
	_, err = ctr.
		WithEnvVariable("GH_ORG_NAME", githubOrgName).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithSecretVariable("GORELEASER_KEY", goreleaserKey).
		WithSecretVariable("AWS_ACCESS_KEY_ID", awsAccessKeyID).
		WithSecretVariable("AWS_SECRET_ACCESS_KEY", awsSecretAccessKey).
		WithSecretVariable("AWS_REGION", awsRegion).
		WithSecretVariable("AWS_BUCKET", awsBucket).
		WithEnvVariable("ARTEFACTS_FQDN", artefactsFQDN).
		WithEnvVariable("ENGINE_VERSION", version).
		WithEnvVariable("ENGINE_TAG", tag).
		WithEntrypoint([]string{"/sbin/tini", "--", "/entrypoint.sh"}).
		WithExec(args, dagger.ContainerWithExecOpts{
			UseEntrypoint: true,
		}).
		Sync(ctx)
	return err
}

// Verify that the CLI builds without actually publishing anything
func (cli DaggerCli) TestPublish(
	ctx context.Context,
	// +optional
	// +defaultPath="/"
	// +ignore_0.13=["!/cmd/dagger/*", "!**/go.sum", "!**/go.mod", "!**/*.go", "!.git", ".git/objects/*", "!.changes"]
	// stopgap:
	// +ignore=["bin", "**/node_modules", "**/.venv", "**/__pycache__"]
	source *dagger.Directory,
	// +optional
	version string,
	// +optional
	tag string,
) error {
	// TODO: ideally this would also use go releaser, but we want to run this
	// step in PRs and locally and we use goreleaser pro features that require
	// a key which is private. For now, this just builds the CLI for the same
	// targets so there's at least some coverage
	oses := []string{"linux", "windows", "darwin"}
	arches := []string{"amd64", "arm64", "arm"}

	eg, _ := errgroup.WithContext(context.Background())
	// Check that the build is not broken on any target platform
	for _, os := range oses {
		for _, arch := range arches {
			if arch == "arm" && os == "darwin" {
				continue
			}
			platform := os + "/" + arch
			if arch == "arm" {
				platform += "/v7" // not always correct but not sure of better way
			}
			eg.Go(func() error {
				_, err := cli.Binary(ctx, source, version, tag, dagger.Platform(platform))
				return err
			})
		}
	}
	// Test that the goreleaser environment is not broken
	eg.Go(func() error {
		_, err := cli.Goreleaser().Sync(ctx)
		return err
	})
	return eg.Wait()
}

func (cli DaggerCli) Goreleaser() *dagger.Container {
	return dag.Container().
		From(fmt.Sprintf("ghcr.io/goreleaser/goreleaser-pro:%s-pro", goReleaserVersion)).
		WithEntrypoint([]string{}).
		WithExec([]string{"apk", "add", "aws-cli"}).
		// install nix
		WithExec([]string{"apk", "add", "xz"}).
		WithDirectory("/nix", dag.Directory()).
		WithNewFile("/etc/nix/nix.conf", `build-users-group =`).
		WithExec([]string{"sh", "-c", "curl -fsSL https://nixos.org/nix/install | sh -s -- --no-daemon"}).
		WithEnvVariable("PATH", "$PATH:/nix/var/nix/profiles/default/bin",
			dagger.ContainerWithEnvVariableOpts{Expand: true},
		).
		// goreleaser requires nix-prefetch-url, so check we can run it
		WithExec([]string{"sh", "-c", "nix-prefetch-url 2>&1 | grep 'error: you must specify a URL'"}).
		WithWorkdir("/app")
}
