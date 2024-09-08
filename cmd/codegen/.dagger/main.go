package main

import "dagger/codegen/internal/dagger"

type Codegen struct{}

// Build the codegen binary
func (m *Codegen) Build(
	// +optional
	// +defaultPath="/"
	// +ignore=["!cmd/codegen", "!**/go.mod", "!**/go.sum", "!sdk/go"]
	source *dagger.Directory,
	// +optional
	platform dagger.Platform,
) *dagger.Directory {
	return dag.Go(source).
		Build(dagger.GoBuildOpts{
			Pkgs:      []string{"./cmd/codegen"},
			NoSymbols: true,
			NoDwarf:   true,
			Platform:  platform,
		})
}

func (m *Codegen) Dev(
	// +optional
	// +defaultPath="/"
	// +ignore=["!cmd/codegen", "!**/go.mod"]
	source *dagger.Directory,
	// +optional
	platform dagger.Platform,
) *dagger.Container {
	return dag.Go(source).Env(dagger.GoEnvOpts{
		Platform: platform,
	})
}
