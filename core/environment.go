package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dagger.io/dagger/telemetry"
	"github.com/dagger/dagger/dagql"
	"github.com/opencontainers/go-digest"
	"github.com/vektah/gqlparser/v2/ast"
)

var _ = `
type LLM {
  model: String!
  withPrompt(prompt: String!): LLM!
  history: [LLMMessage!]!
  lastReply(): String!

  withEnvironment(Environment!): LLM!
  environment: Environment
}

type Environment {
  with[Type]Binding(key: String!, value: [Type], overwrite: Bool, overwriteType: Bool): Environment!
  bindings: [Binding!]
  binding(key: String!): Binding
  encode: File!
}

type Binding {
  key: String!
  as[Type]: [Type]!
}
`

type Environment struct {
	// Bindings by name
	bindings map[string]*Binding
	// Objects by hash
	objects map[digest.Digest]dagql.Typed
}

func NewEnvironment() *Environment {
	return &Environment{
		bindings: map[string]*Binding{},
		objects:  map[digest.Digest]dagql.Typed{},
	}
}

func (env *Environment) Clone() *Environment {
	cp := *env
	cp.bindings = cloneMap(cp.bindings)
	cp.objects = cloneMap(cp.objects)
	return &cp
}

// Lookup dagql typedef for a given dagql value
func (env *LLMEnv) typedef(srv *dagql.Server, val dagql.Typed) *ast.Definition {
	return srv.Schema().Types[val.Type().Name()]
}

func (env *Environment) WithBinding(key string, value dagql.Typed, mask []string) *Environment {
	// FIXME: we don't handle prompt engineering (ie. send clues when value changes)
	// move that concern into BBI builtins
	env.bindings[key] = &Binding{
		Key:   key,
		Value: value,
		Mask:  mask,
	}
	if obj, ok := dagql.UnwrapAs[dagql.Object](value); ok {
		env.objects[obj.ID().Digest()] = value
	}
	return env
}

func (env *Environment) WithoutBinding(key string) *Environment {
	if _, exists := env.bindings[key]; exists {
		env = env.Clone()
		delete(env.bindings, key)
	}
	return env
}

func (env *Environment) Bindings() []*Binding {
	bindings := make([]*Binding, 0, len(env.bindings))
	for _, b := range env.bindings {
		bindings = append(bindings, b)
	}
	return bindings
}

type Binding struct {
	Key   string
	Value dagql.Typed
	Mask  []string
}

func (env *Environment) Tools(srv *dagql.Server) []LLMTool {
	var tools []LLMTool
	if len(env.bindings) > 0 {
		tools = env.Builtins()
	}
	typedefs := make(map[string]*ast.Definition)
	for _, val := range env.vars {
		typedef := env.typedef(srv, val)
		typedefs[typedef.Name] = typedef
	}
	if env.Current() == nil {
		return tools
	}
	typedef := env.typedef(srv, env.Current())
	typeName := typedef.Name
	for _, field := range typedef.Fields {
		if strings.HasPrefix(field.Name, "_") {
			continue
		}
		if strings.HasPrefix(field.Name, "load") && strings.HasSuffix(field.Name, "FromID") {
			continue
		}
		tools = append(tools, LLMTool{
			Name:        typeName + "_" + field.Name,
			Description: field.Description,
			Schema:      fieldArgsToJSONSchema(field),
			Call: func(ctx context.Context, args any) (_ any, rerr error) {
				ctx, span := Tracer(ctx).Start(ctx,
					fmt.Sprintf("🤖💻 %s %v", typeName+"."+field.Name, args),
					telemetry.Passthrough(),
					telemetry.Reveal())
				defer telemetry.End(span, func() error {
					return rerr
				})
				result, err := env.call(ctx, srv, field, args)
				if err != nil {
					return nil, err
				}
				stdio := telemetry.SpanStdio(ctx, InstrumentationLibrary)
				defer stdio.Close()
				switch v := result.(type) {
				case string:
					fmt.Fprint(stdio.Stdout, v)
				default:
					enc := json.NewEncoder(stdio.Stdout)
					enc.SetIndent("", "  ")
					enc.Encode(v)
				}
				return result, nil
			},
		})
	}
	return tools
}
