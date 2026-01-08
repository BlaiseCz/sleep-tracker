package config

import "testing"

func TestGetEnv(t *testing.T) {
    t.Setenv("CFG_VALUE", "custom")
    if got := getEnv("CFG_VALUE", "default"); got != "custom" {
        t.Fatalf("getEnv returned %q, want custom", got)
    }

    // Empty environment value should fall back to default
    t.Setenv("CFG_EMPTY", "")
    if got := getEnv("CFG_EMPTY", "fallback"); got != "fallback" {
        t.Fatalf("getEnv returned %q, want fallback", got)
    }
}

func TestLoad(t *testing.T) {
    // Ensure defaults when env vars are empty.
    t.Setenv("PORT", "")
    t.Setenv("DATABASE_URL", "")
    t.Setenv("LOG_LEVEL", "")
    t.Setenv("SEED", "")
    t.Setenv("OPENAI_API_KEY", "")
    t.Setenv("OPENAI_SLEEP_INSIGHTS_MODEL", "")

    cfg := Load()
    if cfg.Port != "8080" || cfg.DatabaseURL == "" || cfg.LogLevel != "info" {
        t.Fatalf("defaults not applied: %+v", cfg)
    }
    if cfg.Seed {
        t.Fatalf("expected Seed default false")
    }

    // Custom values override defaults
    t.Setenv("PORT", "9090")
    t.Setenv("DATABASE_URL", "postgres://example")
    t.Setenv("LOG_LEVEL", "debug")
    t.Setenv("SEED", "true")
    t.Setenv("OPENAI_API_KEY", "key")
    t.Setenv("OPENAI_SLEEP_INSIGHTS_MODEL", "model")

    cfg = Load()
    if cfg.Port != "9090" || cfg.DatabaseURL != "postgres://example" || cfg.LogLevel != "debug" || !cfg.Seed {
        t.Fatalf("env overrides not applied: %+v", cfg)
    }
    if cfg.OpenAIAPIKey != "key" || cfg.OpenAISleepInsightsModel != "model" {
        t.Fatalf("openai env overrides missing: %+v", cfg)
    }
}
