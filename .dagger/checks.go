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
		WithJob("SDKs", func(ctx context.Context) error {
			return parallel.New().
				// FIXME: we shouldn't hardcode the SDK list here, but native checks will remove this anyway
				WithJob("go", checkJob[CheckStatus](dev.SDK().Go.ReleaseDryRun)).
				WithJob("python", checkJob[CheckStatus](dev.SDK().Python.ReleaseDryRun)).
				WithJob("typescript", checkJob[CheckStatus](dev.SDK().Typescript.ReleaseDryRun)).
				WithJob("php", checkJob[CheckStatus](dev.SDK().PHP.ReleaseDryRun)).
				WithJob("java", checkJob[CheckStatus](dev.SDK().Java.ReleaseDryRun)).
				WithJob("elixir", checkJob[CheckStatus](dev.SDK().Elixir.ReleaseDryRun)).
				WithJob("rust", checkJob[CheckStatus](dev.SDK().Rust.ReleaseDryRun)).
				WithJob("dotnet", checkJob[CheckStatus](dev.SDK().Dotnet.ReleaseDryRun)).
				Run(ctx)
		}).
		Run(ctx)
}

func checkJob[ReturnType any](check func(context.Context) (ReturnType, error)) parallel.JobFunc {
	return parallel.NewJobFunc[ReturnType](check)
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

// Lint the documentation
func (dev *DaggerDev) LintDocs(ctx context.Context) (CheckStatus, error) {
	// FIXME: temporary wrapper
	_, err := CheckCompleted, dag.Docs().Lint(ctx)
}

// Lint the install scripts
func (dev *DaggerDev) LintScripts(ctx context.Context) (CheckStatus, error) {
	// FIXME: temporary wrapper
	return CheckCompleted, dev.Scripts().Lint(ctx)
}

// Lint the Go codebase
func (dev *DaggerDev) LintGo(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dev.godev().CheckLint(ctx)
}

// Verify that scripts work correctly
func (dev *DaggerDev) TestInstallScripts(ctx context.Context) (CheckStatus, error) {
	return CheckCompleted, dev.Scripts().Test(ctx)
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
