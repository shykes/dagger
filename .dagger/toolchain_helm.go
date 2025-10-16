package main

import "context"

type Helm struct{}

func (dev *DaggerDev) Helm() Helm {
	return Helm{}
}

func (tc Helm) Lint(ctx context.Context) (CheckStatus, error) {
	status, err := dag.Helm().Lint(ctx)
	return CheckStatus(status), err
}

// Verify that helm works correctly
func (tc Helm) Test(ctx context.Context) (CheckStatus, error) {
	status, err := dag.Helm().Test(ctx)
	return CheckStatus(status), err
}

// Perform a dry run of a helm release
func (tc Helm) ReleaseDryRun(ctx context.Context) (CheckStatus, error) {
	status, err := dag.Helm().ReleaseDryRun(ctx)
	return CheckStatus(status), err
}
