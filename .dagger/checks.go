package main

import (
	"context"

	"github.com/dagger/dagger/util/parallel"
)

// Verify that generated code is up to date
func (dev *DaggerDev) CheckGenerated(ctx context.Context) (CheckStatus, error) {
	_, err := dev.Generate(ctx, true)
	return CheckCompleted, err
}

func (dev *DaggerDev) ReleaseDryRun(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, parallel.New().
		WithJob("Helm chart", dag.Helm().ReleaseDryRun).
		WithJob("CLI", dag.DaggerCli().ReleaseDryRun).
		WithJob("Engine", dag.DaggerCli().ReleaseDryRun).
		WithJob("SDKs", func(context.Context) (any, error {
			type dryRunner interface {
				Name() string
				CheckReleaseDryRun(context.Context) error
			}
			jobs := parallel.New()
			for _, sdk := range allSDKs[dryRunner](dev) {
				jobs = jobs.WithJob(sdk.Name(), sdk.CheckReleaseDryRun)
			}
			return jobs.Run(ctx)
		}).
		Run(ctx)
}

func (dev *DaggerDev) Lint(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, parallel.New().
		WithJob("lint go packages", dev.LintGo).
		WithJob("lint docs", dev.CheckLintDocs).
		WithJob("lint helm chart", dev.CheckLintHelm).
		WithJob("lint install scripts", dev.CheckLintScripts).
		WithJob("lint SDKs", dev.CheckLintSDKs).
		Run(ctx)
}

// Check that go modules have up-to-date go.mod and go.sum
func (dev *DaggerDev) CheckTidy(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dev.godev().CheckTidy(ctx)
}

// Run linters for all SDKs
func (dev *DaggerDev) LintSDKs(ctx context.Context) (CheckStatus, error) {
	type linter interface {
		Name() string
		CheckLint(context.Context) error
	}
	jobs := parallel.New()
	for _, sdk := range allSDKs[linter](dev) {
		jobs = jobs.WithJob(sdk.Name(), sdk.CheckLint)
	}
	return CheckCompleted, jobs.Run(ctx)
}

// Lint the helm chart
func (dev *DaggerDev) LintHelm(ctx context.Context) (CheckStatus, error) {
	// FIXME: temporary wrapper
	return CheckCompleted, dag.Helm().CheckLint(ctx)
}

// Lint the documentation
func (dev *DaggerDev) LintDocs(ctx context.Context) (CheckStatus, error) {
	// FIXME: temporary wrapper
	return CheckCompleted, dag.Docs().CheckLint(ctx)
}

// Lint the install scripts
func (dev *DaggerDev) LintScripts(ctx context.Context) (CheckStatus, error) {
	// FIXME: temporary wrapper
	return CheckCompleted, dev.Scripts().CheckLint(ctx)
}

// Lint the Go codebase
func (dev *DaggerDev) LintGo(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dev.godev().CheckLint(ctx)
}

// Verify that scripts work correctly
func (dev *DaggerDev) TestInstallScripts(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dev.Scripts().Test(ctx)
}

// Verify that helm works correctly
func (dev *DaggerDev) TestHelm(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dag.Helm().Test(ctx)
}

// Run all checks for all SDKs
func (dev *DaggerDev) TestSDKs(ctx context.Context) (CheckStatus, error) {
	type tester interface {
		Name() string
		Test(context.Context) error
	}
	jobs := parallel.New()
	for _, sdk := range allSDKs[tester](dev) {
		jobs = jobs.WithJob(sdk.Name(), sdk.Test)
	}
	return CheckCompleted, jobs.Run(ctx)
}
