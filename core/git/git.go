package git

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/moby/buildkit/client/llb"
	"go.dagger.io/dagger/core/builtins"
	"go.dagger.io/dagger/router"
)

var _ builtins.Builtin = &git{}

func New(env builtins.Environment) builtins.Builtin {
	return &git{
		env,
	}
}

type git struct {
	builtins.Environment
}

func (s *git) Name() string {
	return "git"
}

func (s *git) Schema() string {
	return `
	extend type Query {
		"Query a git repository"
		git(url: String!): GitRepository!
	}
	
	"A git repository"
	type GitRepository {
		"List of branches on the repository"
		branches: [String!]!
		"Details on one branch"
		branch(name: String!): GitRef!
		"List of tags on the repository"
		tags: [String!]!
		"Details on one tag"
		tag(name: String!): GitRef!
	}
	
	"A git ref (tag or branch)"
	type GitRef {
		"The digest of the current value of this ref"
		digest: String!
		"The filesystem tree at this ref"
		tree: Filesystem!
	}

	# Compat with old API
	extend type Core {
		git(remote: String!, ref: String): Filesystem! @deprecated(reason: "use top-level 'query { git }'")
	}
`
}

func (s *git) Resolvers() router.Resolvers {
	return router.Resolvers{
		"Query": router.ObjectResolver{
			"git": s.git,
		},
		"GitRepository": router.ObjectResolver{
			"branches": s.branches,
			"branch":   s.branch,
			"tags":     s.tags,
			"tag":      s.tag,
		},
		"GitRef": router.ObjectResolver{
			"digest": s.digest,
			"tree":   s.tree,
		},
		"Core": router.ObjectResolver{
			"git": s.gitOld,
		},
	}
}

func (s *git) Dependencies() []router.ExecutableSchema {
	return nil
}

// Compat with old git API
func (s *git) gitOld(p graphql.ResolveParams) (any, error) {
	remote := p.Args["remote"].(string)
	ref, _ := p.Args["ref"].(string)

	var opts []llb.GitOption
	if sockID := s.SSHAuthSockID(); sockID != "" {
		opts = append(opts, llb.MountSSHSock(sockID))
	}
	st := llb.Git(remote, ref, opts...)
	return s.Solve(p.Context, st)
}

type repository struct {
	url string
}

type ref struct {
	repository repository
	name       string
}

func (s *git) git(p graphql.ResolveParams) (any, error) {
	url := p.Args["url"].(string)

	return repository{
		url: url,
	}, nil
}

func (s *git) branch(p graphql.ResolveParams) (any, error) {
	repo := p.Source.(repository)
	return ref{
		repository: repo,
		name:       p.Args["name"].(string),
	}, nil
}

func (s *git) branches(p graphql.ResolveParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *git) tag(p graphql.ResolveParams) (any, error) {
	repo := p.Source.(repository)
	return ref{
		repository: repo,
		name:       p.Args["name"].(string),
	}, nil
}

func (s *git) tags(p graphql.ResolveParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *git) digest(p graphql.ResolveParams) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *git) tree(p graphql.ResolveParams) (any, error) {
	ref := p.Source.(ref)
	var opts []llb.GitOption
	if sockID := s.SSHAuthSockID(); sockID != "" {
		opts = append(opts, llb.MountSSHSock(sockID))
	}
	st := llb.Git(ref.repository.url, ref.name, opts...)
	return s.Solve(p.Context, st)
}
