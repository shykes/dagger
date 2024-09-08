package main

import (
	"dagger/dagger/internal/dagger"
)

type DaggerCli struct{}

func (cli DaggerCli) Build(
	// +optional
	// +defaultPath="/"
	// +ignore=["!cmd/dagger", "!sdk/go"]
	source *dagger.Directory,
	// +optional
	platform dagger.Platform,
	// +optional
	version string,
	// +optional
	tag string,
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
	// +optional
	// +defaultPath="/"
	// +ignore=["!cmd/dagger", "!sdk/go"]
	source *dagger.Directory,
	// +optional
	platform dagger.Platform,
	// +optional
	version string,
	// +optional
	tag string,
) *dagger.File {
	return cli.
		Build(source, platform, version, tag).
		File("bin/dagger")
}
