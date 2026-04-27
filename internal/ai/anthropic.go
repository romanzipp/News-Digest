package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"git.romanzipp.net/romanzipp/news/internal/config"
)

type anthropicClient struct {
	client    anthropic.Client
	model     string
	maxTokens int
	schema    anthropic.ToolInputSchemaParam
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
		schema:    generateDigestSchema(),
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
		Tools: []anthropic.ToolUnionParam{
			{OfTool: &anthropic.ToolParam{
				Name:        "publish_digest",
				Description: anthropic.String("Publish the curated news digest"),
				InputSchema: c.schema,
			}},
		},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{
				Name: "publish_digest",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic completion: %w", err)
	}

	if msg.StopReason == "max_tokens" {
		return "", fmt.Errorf("anthropic: response truncated (hit max_tokens=%d), increase AI_MAX_TOKENS", c.maxTokens)
	}

	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			raw, err := json.Marshal(block.Input)
			if err != nil {
				return "", fmt.Errorf("anthropic: marshal tool input: %w", err)
			}
			return string(raw), nil
		}
	}

	return "", fmt.Errorf("anthropic: no tool_use block in response")
}
