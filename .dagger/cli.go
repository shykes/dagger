package main

import (
	"github.com/dagger/dagger/.dagger/internal/dagger"
)

type CLI struct {
	Dagger *DaggerDev // +private
}

// Build the CLI binary
func (cli *CLI) Binary(
	// +optional
	platform dagger.Platform,
) *dagger.File {
	return dag.DaggerCli().Binary(dagger.DaggerCliBinaryOpts{
		Platform: platform,
		Version:  cli.Dagger.Version.String(),
		Tag:      cli.Dagger.Tag,
	})
}

const (
	// https://github.com/goreleaser/goreleaser/releases
	goReleaserVersion = "v2.2.0"
)
