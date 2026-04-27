package ai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"git.romanzipp.net/romanzipp/news/internal/config"
)

type openAIClient struct {
	client    *openai.Client
	model     string
	maxTokens int
}

func newOpenAIClient(cfg *config.Config) *openAIClient {
	ocfg := openai.DefaultConfig(cfg.AIAPIKey)
	ocfg.BaseURL = cfg.AIEndpoint

	return &openAIClient{
		client:    openai.NewClientWithConfig(ocfg),
		model:     cfg.AIModel,
		maxTokens: cfg.AIMaxTokens,
	}
}

func (c *openAIClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		MaxTokens: c.maxTokens,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices returned")
	}

	if resp.Choices[0].FinishReason == "length" {
		return "", fmt.Errorf("openai: response truncated (hit max_tokens=%d), increase AI_MAX_TOKENS", c.maxTokens)
	}

	return resp.Choices[0].Message.Content, nil
}
