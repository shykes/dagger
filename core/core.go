package core

import (
	bkclient "github.com/moby/buildkit/client"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.dagger.io/dagger/core/base"
	"go.dagger.io/dagger/core/git"
	"go.dagger.io/dagger/project"
	"go.dagger.io/dagger/router"
	"go.dagger.io/dagger/secret"
)

type InitializeArgs struct {
	Router        *router.Router
	SecretStore   *secret.Store
	SSHAuthSockID string
	WorkdirID     string
	Gateway       bkgw.Client
	BKClient      *bkclient.Client
	SolveOpts     bkclient.SolveOpt
	SolveCh       chan *bkclient.SolveStatus
	Platform      specs.Platform
}

func New(params InitializeArgs) (router.ExecutableSchema, error) {
	b := &base.BaseSchema{
		Router:        params.Router,
		SecretStore:   params.SecretStore,
		Gw:            params.Gateway,
		BkClient:      params.BKClient,
		SolveOpts:     params.SolveOpts,
		SolveCh:       params.SolveCh,
		Platform:      params.Platform,
		SSHAuthSockID: params.SSHAuthSockID,
	}
	return router.MergeExecutableSchemas("core",
		&coreSchema{b, params.WorkdirID},
		&git.GitSchema{BaseSchema: b},
		&filesystemSchema{b},
		&projectSchema{
			BaseSchema:      b,
			compiledSchemas: make(map[string]*project.CompiledRemoteSchema),
		},
		&execSchema{b},
		&dockerBuildSchema{b},
		&secretSchema{b},
	)
}
