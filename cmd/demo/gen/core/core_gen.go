// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package core

import (
	"context"

	"github.com/Khan/genqlient/graphql"
	"github.com/dagger/cloak/sdk/go/dagger"
)

// DockerfileCore includes the requested fields of the GraphQL type Core.
type DockerfileCore struct {
	Dockerfile dagger.FS `json:"dockerfile"`
}

// GetDockerfile returns DockerfileCore.Dockerfile, and is useful for accessing the field via an interface.
func (v *DockerfileCore) GetDockerfile() dagger.FS { return v.Dockerfile }

// DockerfileResponse is returned by Dockerfile on success.
type DockerfileResponse struct {
	Core DockerfileCore `json:"core"`
}

// GetCore returns DockerfileResponse.Core, and is useful for accessing the field via an interface.
func (v *DockerfileResponse) GetCore() DockerfileCore { return v.Core }

// ExecCore includes the requested fields of the GraphQL type Core.
type ExecCore struct {
	Exec ExecCoreExec `json:"exec"`
}

// GetExec returns ExecCore.Exec, and is useful for accessing the field via an interface.
func (v *ExecCore) GetExec() ExecCoreExec { return v.Exec }

// ExecCoreExec includes the requested fields of the GraphQL type CoreExec.
type ExecCoreExec struct {
	Fs dagger.FS `json:"fs"`
}

// GetFs returns ExecCoreExec.Fs, and is useful for accessing the field via an interface.
func (v *ExecCoreExec) GetFs() dagger.FS { return v.Fs }

// ExecResponse is returned by Exec on success.
type ExecResponse struct {
	Core ExecCore `json:"core"`
}

// GetCore returns ExecResponse.Core, and is useful for accessing the field via an interface.
func (v *ExecResponse) GetCore() ExecCore { return v.Core }

// ImageCore includes the requested fields of the GraphQL type Core.
type ImageCore struct {
	Image ImageCoreImage `json:"image"`
}

// GetImage returns ImageCore.Image, and is useful for accessing the field via an interface.
func (v *ImageCore) GetImage() ImageCoreImage { return v.Image }

// ImageCoreImage includes the requested fields of the GraphQL type CoreImage.
type ImageCoreImage struct {
	Fs dagger.FS `json:"fs"`
}

// GetFs returns ImageCoreImage.Fs, and is useful for accessing the field via an interface.
func (v *ImageCoreImage) GetFs() dagger.FS { return v.Fs }

// ImageResponse is returned by Image on success.
type ImageResponse struct {
	Core ImageCore `json:"core"`
}

// GetCore returns ImageResponse.Core, and is useful for accessing the field via an interface.
func (v *ImageResponse) GetCore() ImageCore { return v.Core }

// ImportImportPackage includes the requested fields of the GraphQL type Package.
type ImportImportPackage struct {
	Name string    `json:"name"`
	Fs   dagger.FS `json:"fs"`
}

// GetName returns ImportImportPackage.Name, and is useful for accessing the field via an interface.
func (v *ImportImportPackage) GetName() string { return v.Name }

// GetFs returns ImportImportPackage.Fs, and is useful for accessing the field via an interface.
func (v *ImportImportPackage) GetFs() dagger.FS { return v.Fs }

// ImportResponse is returned by Import on success.
type ImportResponse struct {
	Import ImportImportPackage `json:"import"`
}

// GetImport returns ImportResponse.Import, and is useful for accessing the field via an interface.
func (v *ImportResponse) GetImport() ImportImportPackage { return v.Import }

// __DockerfileInput is used internally by genqlient
type __DockerfileInput struct {
	Context        dagger.FS `json:"context"`
	DockerfileName string    `json:"dockerfileName"`
}

// GetContext returns __DockerfileInput.Context, and is useful for accessing the field via an interface.
func (v *__DockerfileInput) GetContext() dagger.FS { return v.Context }

// GetDockerfileName returns __DockerfileInput.DockerfileName, and is useful for accessing the field via an interface.
func (v *__DockerfileInput) GetDockerfileName() string { return v.DockerfileName }

// __ExecInput is used internally by genqlient
type __ExecInput struct {
	Fs   dagger.FS `json:"fs"`
	Args []string  `json:"args"`
}

// GetFs returns __ExecInput.Fs, and is useful for accessing the field via an interface.
func (v *__ExecInput) GetFs() dagger.FS { return v.Fs }

// GetArgs returns __ExecInput.Args, and is useful for accessing the field via an interface.
func (v *__ExecInput) GetArgs() []string { return v.Args }

// __ImageInput is used internally by genqlient
type __ImageInput struct {
	Ref string `json:"ref"`
}

// GetRef returns __ImageInput.Ref, and is useful for accessing the field via an interface.
func (v *__ImageInput) GetRef() string { return v.Ref }

// __ImportInput is used internally by genqlient
type __ImportInput struct {
	Name string    `json:"name"`
	Fs   dagger.FS `json:"fs"`
}

// GetName returns __ImportInput.Name, and is useful for accessing the field via an interface.
func (v *__ImportInput) GetName() string { return v.Name }

// GetFs returns __ImportInput.Fs, and is useful for accessing the field via an interface.
func (v *__ImportInput) GetFs() dagger.FS { return v.Fs }

func Dockerfile(
	ctx context.Context,
	context dagger.FS,
	dockerfileName string,
) (*DockerfileResponse, error) {
	req := &graphql.Request{
		OpName: "Dockerfile",
		Query: `
query Dockerfile ($context: FS!, $dockerfileName: String!) {
	core {
		dockerfile(context: $context, dockerfileName: $dockerfileName)
	}
}
`,
		Variables: &__DockerfileInput{
			Context:        context,
			DockerfileName: dockerfileName,
		},
	}
	var err error
	var client graphql.Client

	client, err = dagger.Client(ctx)
	if err != nil {
		return nil, err
	}

	var data DockerfileResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func Exec(
	ctx context.Context,
	fs dagger.FS,
	args []string,
) (*ExecResponse, error) {
	req := &graphql.Request{
		OpName: "Exec",
		Query: `
query Exec ($fs: FS!, $args: [String]!) {
	core {
		exec(fs: $fs, args: $args) {
			fs
		}
	}
}
`,
		Variables: &__ExecInput{
			Fs:   fs,
			Args: args,
		},
	}
	var err error
	var client graphql.Client

	client, err = dagger.Client(ctx)
	if err != nil {
		return nil, err
	}

	var data ExecResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func Image(
	ctx context.Context,
	ref string,
) (*ImageResponse, error) {
	req := &graphql.Request{
		OpName: "Image",
		Query: `
query Image ($ref: String!) {
	core {
		image(ref: $ref) {
			fs
		}
	}
}
`,
		Variables: &__ImageInput{
			Ref: ref,
		},
	}
	var err error
	var client graphql.Client

	client, err = dagger.Client(ctx)
	if err != nil {
		return nil, err
	}

	var data ImageResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

func Import(
	ctx context.Context,
	name string,
	fs dagger.FS,
) (*ImportResponse, error) {
	req := &graphql.Request{
		OpName: "Import",
		Query: `
mutation Import ($name: String!, $fs: FS!) {
	import(name: $name, fs: $fs) {
		name
		fs
	}
}
`,
		Variables: &__ImportInput{
			Name: name,
			Fs:   fs,
		},
	}
	var err error
	var client graphql.Client

	client, err = dagger.Client(ctx)
	if err != nil {
		return nil, err
	}

	var data ImportResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
