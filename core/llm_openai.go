package core

import (
	"context"
	"net/url"

	"github.com/dagger/dagger/core/bbi"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type OpenAIClient struct {
	client *openai.Client
	config *LlmConfig
}

func newOpenAIClient(config *LlmConfig) *OpenAIClient {
	var opts []option.RequestOption
	opts = append(opts, option.WithHeader("Content-Type", "application/json"))
	if config.Key != "" {
		opts = append(opts, option.WithAPIKey(config.Key))
	}
	if config.Host != "" || config.Path != "" {
		var base url.URL
		base.Scheme = "https"
		base.Host = config.Host
		base.Path = config.Path
		opts = append(opts, option.WithBaseURL(base.String()))
	}
	return &OpenAIClient{
		client: openai.NewClient(opts...),
		config: config,
	}
}

func (c *OpenAIClient) SendQuery(ctx context.Context, history []Message, tools []bbi.Tool) (*LLMResponse, error) {
	// Convert generic Message to OpenAI specific format
	var openAIMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range history {
		switch msg.Role {
		case "user":
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Content.(string)))
		case "assistant":
			openAIMessages = append(openAIMessages, openai.AssistantMessage(msg.Content.(string)))
		case "system":
			openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Content.(string)))
		}
	}

	params := openai.ChatCompletionNewParams{
		Seed:     openai.Int(0),
		Model:    openai.F(openai.ChatModel(c.config.Model)),
		Messages: openai.F(openAIMessages),
	}

	if len(tools) > 0 {
		var toolParams []openai.ChatCompletionToolParam
		for _, tool := range tools {
			toolParams = append(toolParams, openai.ChatCompletionToolParam{
				Type: openai.F(openai.ChatCompletionToolTypeFunction),
				Function: openai.F(openai.FunctionDefinitionParam{
					Name:        openai.String(tool.Name),
					Description: openai.String(tool.Description),
					Parameters:  openai.F(openai.FunctionParameters(tool.Schema)),
				}),
			})
		}
		params.Tools = openai.F(toolParams)
	}

	res, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert OpenAI response to generic LLMResponse
	return &LLMResponse{
		Content:   res.Choices[0].Message.Content,
		ToolCalls: convertOpenAIToolCalls(res.Choices[0].Message.ToolCalls),
	}, nil
}

func convertOpenAIToolCalls(calls []openai.ChatCompletionMessageToolCall) []ToolCall {
	var toolCalls []ToolCall
	for _, call := range calls {
		toolCalls = append(toolCalls, ToolCall{
			ID: call.ID,
			Function: FuncCall{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
			Type: string(call.Type),
		})
	}
	return toolCalls
}
