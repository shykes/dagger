package main

import (
	"context"
	"dagger/security/internal/dagger"

	"github.com/dagger/dagger/.dagger/internal/dagger"
)

type Trivy struct {
	Source      *dagger.Directory
	Containers []*dagger.Containers
	ConfigFiles *dagger.Directory
}

func New(
	// The source tree to run the security toolchain against
	// +defaultPath="/"
	source *dagger.Directory,
) *Trivy {
	return &Trivy{
		Source:      source,
	}
}

// Scan source code for security vulnerabilities
// +cache="session"
func (trivy *Trivy) ScanSource(
	ctx context.Context,
	// +optional
	exclude []string,
) (MyCheckStatus, error) {
	src := trivy.Source
	if len(exclude) > 0 {
		src = trivy.Source.Filter(dagger.DirectoryFilterOpts{
			Exclude: exclude,
		})
	}
	_, err := dag.Container().
		From("aquasec/trivy:0.67.2@sha256:e2b22eac59c02003d8749f5b8d9bd073b62e30fefaef5b7c8371204e0a4b0c08").
		WithMountedCache("/root/.cache/", dag.CacheVolume("trivy-cache")).
		WithMountedDirectory("/src", src).
		WithExec([]string{
			"trivy", "fs", "--pkg-types=library", "--scanners=vuln",
			"--format=json",
			"--exit-code=1",
			"--severity=CRITICAL,HIGH",
			"--show-suppressed",
		}).
		Sync(ctx)
	return CheckCompleted, err
}

func (trivy *Trivyscan) ScanContainers(ctx context.Context) (MyCheckStatus, error) {
}

// Scan container images for security vulnerabilities
func (trivy *Trivyscan) ScanContainers(ctx context.Context, container []*dagger.Container) (MyCheckStatus, error) {
	// FIXME: project-specific code below:
		args := []string{
			"trivy",
			"image",
			"--pkg-types=os,library",
		}
		args = append(args, commonArgs...)
		engineTarball := "/mnt/engine.tar"
		args = append(args, "--input", engineTarball)

		target := dag.DaggerEngine().Container()
		_, err = ctr.
			WithMountedFile(engineTarball, target.AsTarball()).
			WithExec(args).
			Sync(ctx)
		return err
	}).
		Run(ctx)
}
