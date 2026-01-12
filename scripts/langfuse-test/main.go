// Script to test Langfuse connectivity by creating a test trace.
// Usage: go run scripts/langfuse-test/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/langfuse"
)

func main() {
	cfg := langfuse.Config{
		BaseURL:     getEnv("LANGFUSE_BASE_URL", "http://localhost:3001"),
		PublicKey:   os.Getenv("LANGFUSE_PUBLIC_KEY"),
		SecretKey:   os.Getenv("LANGFUSE_SECRET_KEY"),
		Environment: getEnv("LANGFUSE_ENV", "development"),
	}

	fmt.Println("=== Langfuse Connection Test ===")
	fmt.Printf("Base URL:    %s\n", cfg.BaseURL)
	fmt.Printf("Public Key:  %s\n", maskKey(cfg.PublicKey))
	fmt.Printf("Secret Key:  %s\n", maskKey(cfg.SecretKey))
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Println()

	client := langfuse.NewClient(cfg)

	if !client.IsEnabled() {
		log.Fatal("Langfuse client is disabled. Check your env vars.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test trace
	traceID, err := client.CreateTrace(ctx, langfuse.TraceInput{
		UserID: "test-user-123",
		Name:   "test-trace",
		Input: map[string]any{
			"message": "Hello from langfuse-test script",
			"time":    time.Now().Format(time.RFC3339),
		},
		Output: map[string]any{
			"status": "success",
		},
		Tags: []string{"test", "manual"},
	})

	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	fmt.Println("✓ Test trace created successfully!")
	fmt.Printf("  Trace ID: %s\n", traceID)
	fmt.Printf("  View at:  %s/trace/%s\n", cfg.BaseURL, traceID)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func maskKey(key string) string {
	if len(key) < 8 {
		if key == "" {
			return "(empty)"
		}
		return "***"
	}
	return key[:8] + "..."
}
