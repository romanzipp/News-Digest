package ai

import (
	"context"

	"git.romanzipp.net/romanzipp/news/internal/config"
)

type Provider interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

func New(cfg *config.Config) Provider {
	switch cfg.AIProvider {
	case "anthropic":
		return newAnthropicClient(cfg)
	default:
		return newOpenAIClient(cfg)
	}
}
