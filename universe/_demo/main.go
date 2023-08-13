package main

import (
	"fmt"

	"dagger.io/dagger"
)

func main() {
	DaggerClient().Environment().
		WithCommand_(PublishAll).
		WithCheck_(IntegTest).
		WithCheck_(FooTest).
		WithShell_(DevShell).
		Serve()
}

func PublishAll(ctx dagger.Context, version string) (string, error) {
	// First, publish the server
	_, err := DaggerClient().DemoServer().Publish(ctx, version)
	if err != nil {
		return "", fmt.Errorf("failed to publish go server: %w", err)
	}

	// if that worked, publish the client app
	_, err = DaggerClient().DemoClient().Publish(ctx, version)
	if err != nil {
		return "", fmt.Errorf("failed to publish python app: %w", err)
	}

	return "", nil
}

// TODO: func UnitTest(ctx dagger.Context) (*dagger.EnvironmentCheckResult, error) {
func FooTest(ctx dagger.Context) *dagger.EnvironmentCheck {
	// TODO: sugar to make this less annoying
	return DaggerClient().EnvironmentCheck().
		WithSubcheck(DaggerClient().DemoClient().UnitTest()).
		WithSubcheck(DaggerClient().DemoServer().UnitTest())
}

func IntegTest(ctx dagger.Context) (*dagger.EnvironmentCheckResult, error) {
	clientApp := DaggerClient().DemoClient().Build()

	// TODO: need combined stdout/stderr really badly now
	stdout, err := clientApp.
		WithServiceBinding("server", DaggerClient().DemoServer().Container()).
		WithExec(nil).
		Stdout(ctx)
	// TODO: this is all boilerplatey, sugar to support other return types will fix
	if err != nil {
		return DaggerClient().EnvironmentCheckResult().WithOutput(err.Error()), nil
	}
	return DaggerClient().EnvironmentCheckResult().WithSuccess(true).WithOutput(stdout), nil
}

func DevShell(ctx dagger.Context) (*dagger.Container, error) {
	clientApp := DaggerClient().DemoClient().Build()

	return clientApp.
		WithServiceBinding("server", DaggerClient().DemoServer().Container()).
		WithEntrypoint([]string{"sh"}), nil
}
