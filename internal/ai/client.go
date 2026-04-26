package ai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"git.romanzipp.net/romanzipp/news/internal/config"
)

type Client struct {
	client    *openai.Client
	model     string
	maxTokens int
}

func New(cfg *config.Config) *Client {
	ocfg := openai.DefaultConfig(cfg.AIAPIKey)
	ocfg.BaseURL = cfg.AIEndpoint

	return &Client{
		client:    openai.NewClientWithConfig(ocfg),
		model:     cfg.AIModel,
		maxTokens: cfg.AIMaxTokens,
	}
}

func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
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
		return "", fmt.Errorf("ai completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("ai: no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
