package ai

import (
	"context"
	"log"
	"time"

	"git.romanzipp.net/romanzipp/news/internal/config"
)

type Provider interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

func New(cfg *config.Config) Provider {
	var inner Provider
	switch cfg.AIProvider {
	case "anthropic":
		inner = newAnthropicClient(cfg)
	default:
		inner = newOpenAIClient(cfg)
	}
	return &loggingProvider{inner: inner, provider: cfg.AIProvider, model: cfg.AIModel}
}

type loggingProvider struct {
	inner    Provider
	provider string
	model    string
}

func (l *loggingProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	log.Printf("ai request: provider=%s model=%s system_len=%d user_len=%d", l.provider, l.model, len(systemPrompt), len(userPrompt))
	log.Printf("ai request system:\n%s", systemPrompt)
	log.Printf("ai request user:\n%s", userPrompt)

	start := time.Now()
	resp, err := l.inner.Complete(ctx, systemPrompt, userPrompt)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("ai error: provider=%s elapsed=%s err=%v", l.provider, elapsed, err)
		return "", err
	}

	log.Printf("ai response: provider=%s elapsed=%s len=%d", l.provider, elapsed, len(resp))
	log.Printf("ai response body:\n%s", resp)

	return resp, nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
