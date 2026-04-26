package ai

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"git.romanzipp.net/romanzipp/news/internal/config"
)

type anthropicClient struct {
	client    anthropic.Client
	model     string
	maxTokens int
}

func newAnthropicClient(cfg *config.Config) *anthropicClient {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.AIAPIKey),
	}
	if cfg.AIEndpoint != "" && cfg.AIEndpoint != "https://api.openai.com/v1" {
		opts = append(opts, option.WithBaseURL(cfg.AIEndpoint))
	}

	return &anthropicClient{
		client:    anthropic.NewClient(opts...),
		model:     cfg.AIModel,
		maxTokens: cfg.AIMaxTokens,
	}
}

func (c *anthropicClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	msg, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(c.maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userPrompt),
			),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic completion: %w", err)
	}

	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("anthropic: no text content in response")
}
