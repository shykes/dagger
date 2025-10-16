package main

import (
	"context"

	"github.com/dagger/dagger/.dagger/internal/dagger"
)

type Go struct {
	Toolchain *dagger.Go //+private
}

func (dev *DaggerDev) Go(
	source *dagger.Directory,
	exclude []string, // +optional
) Go {
	return Go{
		Toolchain: dag.Go(),
	}
}

func (dev *DaggerDev) godev() *dagger.Go {
	return dag.Go(dev.Source, dagger.GoOpts{
		// FIXME: differentiate between:
		// 1) lint exclusions,
		// 2) go mod tidy exclusions,
		// 3) dagger runtime generation exclusions
		// 4) actually building & testing stuff
		// --> maybe it's a "check exclusion"?
		Exclude: []string{
			"docs/**",
			"core/integration/**",
			"dagql/idtui/viztest/broken/**",
			"modules/evals/**",
			"**/broken*/**",
		},
		Values: []string{
			"github.com/dagger/dagger/engine.Version=" + dev.Version,
			"github.com/dagger/dagger/engine.Tag=" + dev.Tag,
		},
	})
}

// Check that go modules have up-to-date go.mod and go.sum
func (tc Go) CheckTidy(ctx context.Context) (CheckStatus, error) {
	_, err := dev.godev().CheckTidy(ctx)
	return CheckCompleted, err
}

// Verify that helm works correctly
func (tc Go) Test(ctx context.Context) (CheckStatus, error) {
	status, err := dag.Helm().Test(ctx)
	return CheckStatus(status), err
}
