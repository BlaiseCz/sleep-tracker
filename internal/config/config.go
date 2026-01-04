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
}

func Load() *Config {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://sleepuser:sleeppass@localhost:5432/sleeptracker?sslmode=disable"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Seed:        getEnv("SEED", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
