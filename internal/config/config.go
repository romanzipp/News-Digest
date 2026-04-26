package config

import (
	"os"
	"strconv"
)

type Config struct {
	ListenAddr          string
	BaseURL             string
	DBDriver            string
	DBDsn               string
	SessionSecret       string
	RegistrationEnabled bool
	AIEndpoint          string
	AIAPIKey            string
	AIModel             string
	AIMaxTokens         int
	AIMaxContext         int
	FetchCron           string
	DigestCron          string
	ImageProxyEnabled bool
}

func Load() *Config {
	return &Config{
		ListenAddr:          envOr("LISTEN_ADDR", ":8080"),
		BaseURL:             envOr("BASE_URL", "http://localhost:8080"),
		DBDriver:            envOr("DB_DRIVER", "sqlite"),
		DBDsn:               envOr("DB_DSN", "file:data/news.db?_journal=WAL&_fk=1"),
		SessionSecret:       envOr("SESSION_SECRET", "change-me"),
		RegistrationEnabled: envBool("REGISTRATION_ENABLED", true),
		AIEndpoint:          envOr("AI_ENDPOINT", "https://api.openai.com/v1"),
		AIAPIKey:            os.Getenv("AI_API_KEY"),
		AIModel:             envOr("AI_MODEL", "gpt-4o"),
		AIMaxTokens:         envInt("AI_MAX_TOKENS", 4096),
		AIMaxContext:         envInt("AI_MAX_CONTEXT", 128000),
		FetchCron:           envOr("FETCH_CRON", "0 5 * * *"),
		DigestCron:          envOr("DIGEST_CRON", "0 6 * * *"),
		ImageProxyEnabled: envBool("IMAGE_PROXY_ENABLED", true),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

