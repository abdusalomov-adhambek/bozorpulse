package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	Env         string
	JWTSecret   string // unused placeholder for future signed tokens

	// AI / LLM settings. The LLM is enabled by default; when no API key is
	// configured the composition root falls back to the rule-based assistant.
	AILLMEnabled    bool
	LLMAPIURL       string
	LLMAPIKey       string
	LLMModel        string
	LLMMaxTokens    int
	LLMTimeout      time.Duration
	AIMaxToolRounds int
}

func Load() Config {
	return Config{
		Addr:        get("BP_ADDR", ":8080"),
		DatabaseURL: get("DATABASE_URL", "postgres://bozorpulse:bozorpulse@localhost:5432/bozorpulse?sslmode=disable"),
		Env:         get("APP_ENV", "development"),
		JWTSecret:   get("JWT_SECRET", "dev-only-secret"),

		AILLMEnabled:    envBool("AI_LLM_ENABLED", true),
		LLMAPIURL:       get("LLM_API_URL", "https://api.meta.ai/v1"),
		LLMAPIKey:       firstNonEmpty(os.Getenv("MODEL_API_KEY"), os.Getenv("LLM_API_KEY"), os.Getenv("META_API_KEY")),
		LLMModel:        get("LLM_MODEL", "muse-spark-1.3"),
		LLMMaxTokens:    envInt("LLM_MAX_TOKENS", 1600),
		LLMTimeout:      envDuration("LLM_TIMEOUT", 15*time.Second),
		AIMaxToolRounds: envInt("AI_MAX_TOOL_ROUNDS", 5),
	}
}

func get(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func envBool(k string, fallback bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(k string, fallback int) int {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envDuration(k string, fallback time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
