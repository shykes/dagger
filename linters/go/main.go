// A generated module for Go functions

package main

import (
	"context"
	"encoding/json"
)

const (
	// Tied to the container configuration
	shellHistoryPath = "/root/.ash_history"
	lintCommand = "golangci-lint run  --out-format=json --timeout 5m --issues-exit-code=0"
)

type Go struct{}

// The official Go linter used to develop Dagger
func (m *Go) Container(
	// The Go source directory to lint
	source *Directory,
) *Container {
	return dag.
		Container().
		From("golangci/golangci-lint:v1.54-alpine").
		WithMountedDirectory("/app", source).
		WithWorkdir("/app").
		WithNewFile(shellHistoryPath, ContainerWithNewFileOpts{
			Contents: lintCommand + "\n",
		})
}

func (m *Go) Lint(
	ctx context.Context,
	// The Go source directory to lint
	source *Directory,
) (*Report, error) {
	raw, err := m.
		Container(source).
		WithExec([]string{"sh", "-c", lintCommand}).
		Stdout(ctx)
	return &Report{
		JSON: raw,
	}, err
}

type Report struct {
	JSON string
}

func (r *Report) Checks() ([]*Check, error) {
	var checks []*Check
	if err := json.Unmarshal([]byte(r.JSON), &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

type Check struct {
	FIXME string
}
