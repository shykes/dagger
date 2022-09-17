package builtins

import (
	"context"

	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.dagger.io/dagger/core/filesystem"
	"go.dagger.io/dagger/router"
)

type Builtin interface {
	router.ExecutableSchema
}

// What a builtin can see of its surrounding environment
type Environment interface {
	AddSecret(context.Context, []byte) string
	GetSecret(context.Context, string) ([]byte, error)
	Solve(context.Context, llb.State, ...llb.ConstraintsOpt) (*filesystem.Filesystem, error)
	Export(context.Context, *filesystem.Filesystem, bkclient.ExportEntry) error
	WorkdirPath() string
	WorkdirID() string
	Platform() specs.Platform
	SSHAuthSockID() string
	Buildkit() bkgw.Client
	GetSchema(string) router.ExecutableSchema
	AddSchema(router.ExecutableSchema) error
}
