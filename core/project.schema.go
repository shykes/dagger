package core

import (
	"fmt"
	"sync"

	"github.com/graphql-go/graphql"
	"go.dagger.io/dagger/core/base"
	"go.dagger.io/dagger/core/filesystem"
	"go.dagger.io/dagger/project"
	"go.dagger.io/dagger/router"
	"golang.org/x/sync/singleflight"
)

type Project struct {
	Name         string
	Schema       string
	Dependencies []*Project
	Scripts      []*project.Script
	Extensions   []*project.Extension
	schema       *project.RemoteSchema // internal-only, for convenience in `install` resolver
}

var _ router.ExecutableSchema = &projectSchema{}

type projectSchema struct {
	*base.BaseSchema
	compiledSchemas map[string]*project.CompiledRemoteSchema
	l               sync.RWMutex
	sf              singleflight.Group
}

func (s *projectSchema) Name() string {
	return "project"
}

func (s *projectSchema) Schema() string {
	return `
"A set of scripts and/or extensions"
type Project {
	"name of the project"
	name: String!

	"schema provided by the project"
	schema: String

	"extensions in this project"
	extensions: [Extension!]!

	"scripts in this project"
	scripts: [Script!]!

	"other projects with schema this project depends on"
	dependencies: [Project!]

	"install the project's schema"
	install: Boolean!

	"Code files generated by the SDKs in the project"
	generatedCode: Filesystem!
}

"A schema extension provided by a project"
type Extension {
	"path to the extension's code within the project's filesystem"
	path: String!

	"schema contributed to the project by this extension"
	schema: String!

	"sdk used to generate code for and/or execute this extension"
	sdk: String!
}

"An executable script that uses the project's dependencies and/or extensions"
type Script {
	"path to the script's code within the project's filesystem"
	path: String!

	"sdk used to generate code for and/or execute this script"
	sdk: String!
}

extend type Filesystem {
	"load a project's metadata"
	loadProject(configPath: String!): Project!
}

extend type Core {
	"Look up a project by name"
	project(name: String!): Project!
}
`
}

func (s *projectSchema) Resolvers() router.Resolvers {
	return router.Resolvers{
		"Filesystem": router.ObjectResolver{
			"loadProject": s.loadProject,
		},
		"Core": router.ObjectResolver{
			"project": s.project,
		},
		"Project": router.ObjectResolver{
			"install":       s.install,
			"generatedCode": s.generatedCode,
		},
	}
}

func (s *projectSchema) Dependencies() []router.ExecutableSchema {
	return nil
}

func (s *projectSchema) install(p graphql.ResolveParams) (any, error) {
	obj := p.Source.(*Project)

	executableSchema, err := obj.schema.Compile(p.Context, s.compiledSchemas, &s.l, &s.sf)
	if err != nil {
		return nil, err
	}

	if err := s.Router.Add(executableSchema); err != nil {
		return nil, err
	}

	return true, nil
}

func (s *projectSchema) loadProject(p graphql.ResolveParams) (any, error) {
	obj, err := filesystem.FromSource(p.Source)
	if err != nil {
		return nil, err
	}

	configPath := p.Args["configPath"].(string)
	schema, err := project.Load(p.Context, s.Gw, s.Platform, obj, configPath, s.SSHAuthSockID)
	if err != nil {
		return nil, err
	}

	return remoteSchemaToProject(schema), nil
}

func (s *projectSchema) project(p graphql.ResolveParams) (any, error) {
	name := p.Args["name"].(string)

	schema := s.Router.Get(name)
	if schema == nil {
		return nil, fmt.Errorf("project %q not found", name)
	}

	return routerSchemaToProject(schema), nil
}

func (s *projectSchema) generatedCode(p graphql.ResolveParams) (any, error) {
	obj := p.Source.(*Project)
	coreSchema := s.Router.Get("core")
	return obj.schema.Generate(p.Context, coreSchema.Schema())
}

// TODO:(sipsma) guard against infinite recursion
func routerSchemaToProject(schema router.ExecutableSchema) *Project {
	ext := &Project{
		Name:   schema.Name(),
		Schema: schema.Schema(),
		//FIXME:(sipsma) Scripts, Extensions are not exposed on router.ExecutableSchema yet
	}
	for _, dep := range schema.Dependencies() {
		ext.Dependencies = append(ext.Dependencies, routerSchemaToProject(dep))
	}
	return ext
}

// TODO:(sipsma) guard against infinite recursion
func remoteSchemaToProject(schema *project.RemoteSchema) *Project {
	ext := &Project{
		Name:       schema.Name(),
		Schema:     schema.Schema(),
		Scripts:    schema.Scripts(),
		Extensions: schema.Extensions(),
		schema:     schema,
	}
	for _, dep := range schema.Dependencies() {
		ext.Dependencies = append(ext.Dependencies, remoteSchemaToProject(dep))
	}
	return ext
}
