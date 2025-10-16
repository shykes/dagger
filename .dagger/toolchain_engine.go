package main

import "context"

type Engine struct{}

func (dev *DaggerDev) Toolchain() Engine {
	return Engine{}
}

func (tc Engine) ReleaseDryRun(ctx context.Context) (CheckStatus, error) {
	status, err := dag.DaggerEngine().ReleaseDryRun(ctx)
	return CheckStatus(status), err
}

func (tc Engine) Publish(ctx context.Context) error {
	reutrn dag.DaggerEngine().Publish(ctx)
}
