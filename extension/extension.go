package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/containerd/containerd/platforms"
	"github.com/dagger/cloak/core/filesystem"
	"github.com/dagger/cloak/router"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/moby/buildkit/client/llb"
	dockerfilebuilder "github.com/moby/buildkit/frontend/dockerfile/builder"
	bkgw "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/singleflight"
)

const (
	schemaPath     = "/schema.graphql"
	operationsPath = "/operations.graphql"
	entrypointPath = "/entrypoint"

	daggerSockName = "dagger-sock"
	daggerSockPath = "/dagger.sock"

	fsMountPath  = "/mnt"
	tmpMountPath = "/tmp"

	inputMountPath = "/inputs"
	inputFile      = "/dagger.json"

	outputMountPath = "/outputs"
	outputFile      = "/dagger.json"
)

// RemoteSchema holds the schema, operations and other configuration of an
// extension, but has not yet been "compiled" with an SDK to an executable
// extension. This allows obtaining the extension metadata without necessarily
// being able to build it yet.
type RemoteSchema struct {
	gw         bkgw.Client
	platform   specs.Platform
	contextFS  *filesystem.Filesystem
	configPath string

	router.LoadedSchema
	dependencies []*RemoteSchema
}

func Load(ctx context.Context, gw bkgw.Client, platform specs.Platform, contextFS *filesystem.Filesystem, configPath string) (*RemoteSchema, error) {
	cfgBytes, err := contextFS.ReadFile(ctx, gw, configPath)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseConfig(cfgBytes)
	if err != nil {
		return nil, err
	}

	s := &RemoteSchema{
		gw:         gw,
		platform:   platform,
		contextFS:  contextFS,
		configPath: configPath,
	}

	var sourceSchemas []router.LoadedSchema
	for _, src := range cfg.Sources {
		sdl, err := contextFS.ReadFile(ctx, gw, filepath.Join(
			filepath.Dir(configPath),
			src.Path,
			schemaPath,
		))
		if err != nil && !isGatewayFileNotFound(err) {
			return nil, err
		}

		operations, err := contextFS.ReadFile(ctx, gw, filepath.Join(
			filepath.Dir(configPath),
			src.Path,
			operationsPath,
		))
		if err != nil && !isGatewayFileNotFound(err) {
			return nil, err
		}

		sourceSchemas = append(sourceSchemas, router.StaticSchema(router.StaticSchemaParams{
			Schema:     string(sdl),
			Operations: string(operations),
		}))
	}
	s.LoadedSchema = router.MergeLoadedSchemas(cfg.Name, sourceSchemas...)

	for _, dep := range cfg.Dependencies {
		switch {
		case dep.Local != "":
			depConfigPath := filepath.Join(filepath.Dir(configPath), dep.Local)
			// TODO:(sipsma) guard against infinite recursion
			// TODO:(sipsma) deduplicate load of same dependencies (same as compile)
			depSchema, err := Load(ctx, gw, platform, contextFS, depConfigPath)
			if err != nil {
				return nil, err
			}
			s.dependencies = append(s.dependencies, depSchema)
		}
	}

	return s, nil
}

func (s *RemoteSchema) Dependencies() []*RemoteSchema {
	return s.dependencies
}

func (s RemoteSchema) Compile(ctx context.Context, cache map[string]*CompiledRemoteSchema, l *sync.RWMutex, sf *singleflight.Group) (*CompiledRemoteSchema, error) {
	res, err, _ := sf.Do(s.Name(), func() (interface{}, error) {
		// if we have already compiled a schema with this name, return it
		// TODO:(sipsma) should check that schema is actually the same, error out if not
		l.RLock()
		cached, ok := cache[s.Name()]
		l.RUnlock()
		if ok {
			return cached, nil
		}

		// TODO:(sipsma) hardcoding use of a "dockerfile sdk", should obviously be generalized
		def, err := s.contextFS.ToDefinition()
		if err != nil {
			return nil, err
		}

		opts := map[string]string{
			"platform": platforms.Format(s.platform),
			"filename": filepath.Join(filepath.Dir(s.configPath), "Dockerfile"),
		}
		inputs := map[string]*pb.Definition{
			dockerfilebuilder.DefaultLocalNameContext:    def,
			dockerfilebuilder.DefaultLocalNameDockerfile: def,
		}
		res, err := s.gw.Solve(ctx, bkgw.SolveRequest{
			Frontend:       "dockerfile.v0",
			FrontendOpt:    opts,
			FrontendInputs: inputs,
			Evaluate:       true,
		})
		if err != nil {
			return nil, err
		}

		bkref, err := res.SingleRef()
		if err != nil {
			return nil, err
		}
		st, err := bkref.ToState()
		if err != nil {
			return nil, err
		}

		runtimeFS, err := filesystem.FromState(ctx, st, s.platform)
		if err != nil {
			return nil, err
		}

		compiled := &CompiledRemoteSchema{
			RemoteSchema: s,
			resolvers:    router.Resolvers{},
		}

		doc, err := parser.Parse(parser.ParseParams{Source: s.Schema()})
		if err != nil {
			return nil, err
		}
		for _, def := range doc.Definitions {
			var obj *ast.ObjectDefinition

			if def, ok := def.(*ast.ObjectDefinition); ok {
				obj = def
			}

			if def, ok := def.(*ast.TypeExtensionDefinition); ok {
				obj = def.Definition
			}

			if obj == nil {
				continue
			}

			objResolver := router.ObjectResolver{}
			compiled.resolvers[obj.Name.Value] = objResolver
			for _, field := range obj.Fields {
				objResolver[field.Name.Value] = compiled.resolver(runtimeFS)
			}
		}

		for _, dep := range s.dependencies {
			// TODO:(sipsma) guard against infinite recursion
			depCompiled, err := dep.Compile(ctx, cache, l, sf)
			if err != nil {
				return nil, err
			}
			compiled.dependencies = append(compiled.dependencies, depCompiled)
		}

		l.Lock()
		cache[s.Name()] = compiled
		l.Unlock()
		return compiled, nil
	})
	if err != nil {
		return nil, err
	}
	return res.(*CompiledRemoteSchema), nil
}

// CompiledRemoteSchema is the compiled version of RemoteSchema where the
// SDK has built its input into an executable extension.
type CompiledRemoteSchema struct {
	RemoteSchema
	dependencies []router.ExecutableSchema
	resolvers    router.Resolvers
}

var _ router.ExecutableSchema = &CompiledRemoteSchema{}

func (s *CompiledRemoteSchema) Resolvers() router.Resolvers {
	return s.resolvers
}

func (s *CompiledRemoteSchema) Dependencies() []router.ExecutableSchema {
	return s.dependencies
}

func (s *CompiledRemoteSchema) resolver(runtimeFS *filesystem.Filesystem) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		pathArray := p.Info.Path.AsArray()
		name := fmt.Sprintf("%+v", pathArray)

		resolverName := fmt.Sprintf("%s.%s", p.Info.ParentType.Name(), p.Info.FieldName)
		inputMap := map[string]interface{}{
			"resolver": resolverName,
			"args":     p.Args,
			"parent":   p.Source,
		}
		inputBytes, err := json.Marshal(inputMap)
		if err != nil {
			return nil, err
		}
		input := llb.Scratch().File(llb.Mkfile(inputFile, 0644, inputBytes))

		fsState, err := runtimeFS.ToState()
		if err != nil {
			return nil, err
		}

		st := fsState.Run(
			llb.Args([]string{entrypointPath}),
			llb.AddSSHSocket(
				llb.SSHID(daggerSockName),
				llb.SSHSocketTarget(daggerSockPath),
			),
			llb.AddMount(inputMountPath, input, llb.Readonly),
			llb.AddMount(tmpMountPath, llb.Scratch(), llb.Tmpfs()),
			llb.ReadonlyRootFS(),
		)

		// TODO: /mnt should maybe be configurable?
		for path, fsid := range collectFSPaths(p.Args, fsMountPath, nil) {
			fs := filesystem.New(fsid)
			fsState, err := fs.ToState()
			if err != nil {
				return nil, err
			}
			// TODO: it should be possible for this to be outputtable by the action; the only question
			// is how to expose that ability in a non-confusing way, just needs more thought
			st.AddMount(path, fsState, llb.ForceNoOutput)
		}

		outputMnt := st.AddMount(outputMountPath, llb.Scratch())
		outputDef, err := outputMnt.Marshal(p.Context, llb.Platform(s.platform), llb.WithCustomName(name))
		if err != nil {
			return nil, err
		}

		res, err := s.gw.Solve(p.Context, bkgw.SolveRequest{
			Definition: outputDef.ToPB(),
		})
		if err != nil {
			return nil, err
		}
		ref, err := res.SingleRef()
		if err != nil {
			return nil, err
		}
		outputBytes, err := ref.ReadFile(p.Context, bkgw.ReadRequest{
			Filename: outputFile,
		})
		if err != nil {
			return nil, err
		}
		var output interface{}
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output: %w", err)
		}
		return output, nil
	}
}

func collectFSPaths(arg interface{}, curPath string, fsPaths map[string]filesystem.FSID) map[string]filesystem.FSID {
	if fsPaths == nil {
		fsPaths = make(map[string]filesystem.FSID)
	}

	switch arg := arg.(type) {
	case filesystem.FSID:
		// TODO: make sure there can't be any shenanigans with args named e.g. ../../../foo/bar
		fsPaths[curPath] = arg
	case map[string]interface{}:
		for k, v := range arg {
			fsPaths = collectFSPaths(v, filepath.Join(curPath, k), fsPaths)
		}
	case []interface{}:
		for i, v := range arg {
			// TODO: path format technically works but weird as hell, gotta be a better way
			fsPaths = collectFSPaths(v, fmt.Sprintf("%s/%d", curPath, i), fsPaths)
		}
	}
	return fsPaths
}

func isGatewayFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	// TODO:(sipsma) the underlying error type doesn't appear to be passed over grpc
	// from buildkit, so we have to resort to nasty substring checking, need a better way
	return strings.Contains(err.Error(), "no such file or directory")
}
