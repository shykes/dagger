package core

import (
	"context"

	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.dagger.io/dagger/core/builtins"
	"go.dagger.io/dagger/core/filesystem"
	"go.dagger.io/dagger/core/git"
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
	env := &builtinEnv{
		router:        params.Router,
		secretStore:   params.SecretStore,
		gw:            params.Gateway,
		bkClient:      params.BKClient,
		solveOpts:     params.SolveOpts,
		solveCh:       params.SolveCh,
		platform:      params.Platform,
		sshAuthSockID: params.SSHAuthSockID,
		workdirID:     params.WorkdirID,
	}
	return router.MergeExecutableSchemas("core",
		newCoreBuiltin(env),
		git.New(env),
		newFilesystemBuiltin(env),
		newProjectBuiltin(env),
		newExecBuiltin(env),
		newDockerBuildBuiltin(env),
		newSecretBuiltin(env),
	)
}

var _ builtins.Environment = &builtinEnv{}

type builtinEnv struct {
	router        *router.Router
	secretStore   *secret.Store
	gw            bkgw.Client
	bkClient      *bkclient.Client
	solveOpts     bkclient.SolveOpt
	solveCh       chan *bkclient.SolveStatus
	platform      specs.Platform
	sshAuthSockID string
	workdirID     string
}

func (e *builtinEnv) AddSecret(ctx context.Context, value []byte) string {
	return e.secretStore.AddSecret(ctx, value)
}

func (e *builtinEnv) GetSecret(ctx context.Context, id string) ([]byte, error) {
	return e.secretStore.GetSecret(ctx, id)
}

func (e *builtinEnv) WorkdirPath() string {
	return e.solveOpts.LocalDirs[e.workdirID]
}

func (e *builtinEnv) WorkdirID() string {
	return e.workdirID
}

func (e *builtinEnv) Platform() specs.Platform {
	return e.platform
}

func (e *builtinEnv) SSHAuthSockID() string {
	return e.sshAuthSockID
}

func (e *builtinEnv) GetSchema(name string) router.ExecutableSchema {
	return e.router.Get(name)
}

func (e *builtinEnv) AddSchema(schema router.ExecutableSchema) error {
	return e.router.Add(schema)
}

func (e *builtinEnv) Buildkit() bkgw.Client {
	return e.gw
}

func (e *builtinEnv) Solve(ctx context.Context, st llb.State, marshalOpts ...llb.ConstraintsOpt) (*filesystem.Filesystem, error) {
	def, err := st.Marshal(ctx, append([]llb.ConstraintsOpt{llb.Platform(e.Platform())}, marshalOpts...)...)
	if err != nil {
		return nil, err
	}
	_, err = e.gw.Solve(ctx, bkgw.SolveRequest{
		Evaluate:   true,
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, err
	}

	// FIXME: should we create a filesystem from `res.SingleRef()`?
	return filesystem.FromDefinition(def), nil
}

func (e *builtinEnv) Export(ctx context.Context, fs *filesystem.Filesystem, export bkclient.ExportEntry) error {
	fsDef, err := fs.ToDefinition()
	if err != nil {
		return err
	}

	solveOpts := e.solveOpts
	// NOTE: be careful to not overwrite any values from original shared r.solveOpts (i.e. with append).
	solveOpts.Exports = []bkclient.ExportEntry{export}

	// Mirror events from the sub-Build into the main Build event channel.
	// Build() will close the channel after completion so we don't want to use the main channel directly.
	ch := make(chan *bkclient.SolveStatus)
	go func() {
		for event := range ch {
			e.solveCh <- event
		}
	}()

	_, err = e.bkClient.Build(ctx, solveOpts, "", func(ctx context.Context, gw bkgw.Client) (*bkgw.Result, error) {
		return gw.Solve(ctx, bkgw.SolveRequest{
			Evaluate:   true,
			Definition: fsDef,
		})
	}, ch)
	if err != nil {
		return err
	}
	return nil
}
