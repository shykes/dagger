package core

import (
	"fmt"

	"github.com/dagger/cloak/router"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

var secretIDResolver = router.ScalarResolver{
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			return v
		default:
			panic(fmt.Sprintf("unexpected secret type %T", v))
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			return v
		default:
			panic(fmt.Sprintf("unexpected secret value type %T: %+v", v, v))
		}
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return valueAST.Value
		default:
			panic(fmt.Sprintf("unexpected secret literal type: %T", valueAST))
		}
	},
}

var _ router.ExecutableSchema = &secretSchema{}

type secretSchema struct {
	*baseSchema
}

func (s *secretSchema) Schema() string {
	return `
	scalar SecretID

	extend type Core {
		"Look up a secret by ID"
		secret(id: SecretID!): String!

		"Add a secret"
		addSecret(plaintext: String!): SecretID!
	}
	`
}

func (s *secretSchema) Operations() string {
	return `
	query Secret($id: SecretID!) {
		core {
			secret(id: $id)
		}
	}
	`
}

func (r *secretSchema) Resolvers() router.Resolvers {
	return router.Resolvers{
		"SecretID": secretIDResolver,
		"Core": router.ObjectResolver{
			"secret":    r.secret,
			"addSecret": r.addSecret,
		},
	}
}

func (r *secretSchema) secret(p graphql.ResolveParams) (any, error) {
	plaintext, err := r.secretStore.GetSecret(p.Context, p.Args["id"].(string))
	if err != nil {
		return nil, err
	}
	return string(plaintext), nil
}

func (r *secretSchema) addSecret(p graphql.ResolveParams) (any, error) {
	plaintext := p.Args["plaintext"].(string)
	return r.secretStore.AddSecret(p.Context, []byte(plaintext)), nil
}
