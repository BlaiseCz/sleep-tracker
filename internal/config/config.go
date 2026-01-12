package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	LogLevel    string
	Seed        bool

	// OpenAI configuration
	OpenAIAPIKey             string
	OpenAISleepInsightsModel string

	// Langfuse configuration
	LangfuseBaseURL   string
	LangfusePublicKey string
	LangfuseSecretKey string
	LangfuseEnv       string
}

func Load() *Config {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://sleepuser:sleeppass@localhost:5432/sleeptracker?sslmode=disable"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Seed:        getEnv("SEED", "false") == "true",

		OpenAIAPIKey:             getEnv("OPENAI_API_KEY", ""),
		OpenAISleepInsightsModel: getEnv("OPENAI_SLEEP_INSIGHTS_MODEL", "gpt-4o-mini"),

		LangfuseBaseURL:   getEnv("LANGFUSE_BASE_URL", ""),
		LangfusePublicKey: getEnv("LANGFUSE_PUBLIC_KEY", ""),
		LangfuseSecretKey: getEnv("LANGFUSE_SECRET_KEY", ""),
		LangfuseEnv:       getEnv("LANGFUSE_ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
