// Sleep Tracker API
//
// REST API for tracking sleep patterns and quality.
//
//	@title			Sleep Tracker API
//	@version		1.0
//	@description	Track sleep sessions with start/end times, quality ratings, and timezone support.
//
//	@BasePath	/v1
//
//	@tag.name			users
//	@tag.description	User management endpoints
//
//	@tag.name			sleep-logs
//	@tag.description	Sleep session tracking endpoints
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/api"
	"github.com/blaisecz/sleep-tracker/internal/api/handler"
	"github.com/blaisecz/sleep-tracker/internal/config"
	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/langfuse"
	"github.com/blaisecz/sleep-tracker/internal/llm"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/blaisecz/sleep-tracker/internal/seed"
	"github.com/blaisecz/sleep-tracker/internal/service"
	"github.com/blaisecz/sleep-tracker/internal/telemetry"
)

const defaultLocalPromptPath = "prompts/sleep_insights_system_prompt.md"
const promptCacheTTL = 30 * time.Second

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize OpenTelemetry tracer (exports to Langfuse when configured)
	ctx := context.Background()
	promptProvider := llm.CachedPromptProvider(buildSystemPromptProvider(cfg), promptCacheTTL)
	if _, err := promptProvider(ctx); err != nil {
		log.Printf("Failed to load system prompt at startup: %v", err)
	}

	tracerShutdown, err := telemetry.InitTracer(ctx, cfg, "sleep-tracker-api")
	if err != nil {
		log.Printf("Failed to initialize telemetry: %v", err)
	} else {
		defer func() {
			if err := tracerShutdown(context.Background()); err != nil {
				log.Printf("Failed to shutdown telemetry: %v", err)
			}
		}()
	}

	// Connect to database
	db, err := config.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database schema
	if err := db.AutoMigrate(&domain.User{}, &domain.SleepLog{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed")

	if cfg.Seed {
		log.Println("Seeding database with sample data (SEED=true)...")
		if err := seed.Run(db); err != nil {
			log.Fatalf("Failed to seed database: %v", err)
		}
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sleepLogRepo := repository.NewSleepLogRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo)
	sleepLogService := service.NewSleepLogService(sleepLogRepo, userRepo)
	chronotypeService := service.NewChronotypeService(sleepLogRepo, userRepo)
	metricsService := service.NewMetricsService(sleepLogRepo, userRepo)

	// Initialize OpenAI client (may be nil if not configured)
	openaiClient := llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAISleepInsightsModel, promptProvider)
	if openaiClient == nil {
		log.Println("Warning: OpenAI API key not configured, insights endpoint will be unavailable")
	}

	// Initialize Langfuse client (logs its own status)
	langfuseClient := langfuse.NewClient(langfuse.Config{
		BaseURL:     cfg.LangfuseBaseURL,
		PublicKey:   cfg.LangfusePublicKey,
		SecretKey:   cfg.LangfuseSecretKey,
		Environment: cfg.LangfuseEnv,
	})

	// Initialize insights service
	insightsService := service.NewInsightsService(chronotypeService, metricsService, openaiClient, sleepLogRepo, userRepo)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	sleepLogHandler := handler.NewSleepLogHandler(sleepLogService)
	insightsHandler := handler.NewInsightsHandler(chronotypeService, metricsService, insightsService, langfuseClient)

	// Setup router
	router := api.NewRouter(userHandler, sleepLogHandler, insightsHandler)
	routerHandler := router.Setup()

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, routerHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func buildSystemPromptProvider(cfg *config.Config) llm.SystemPromptProvider {
	localPath := cfg.LangfusePromptSavePath
	if localPath == "" {
		localPath = defaultLocalPromptPath
	}

	return func(ctx context.Context) (string, error) {
		if cfg.LangfusePromptName != "" {
			prompt, err := langfuse.LoadPrompt(ctx, langfuse.PromptLoaderConfig{
				BaseURL:     cfg.LangfuseBaseURL,
				PublicKey:   cfg.LangfusePublicKey,
				SecretKey:   cfg.LangfuseSecretKey,
				PromptName:  cfg.LangfusePromptName,
				PromptLabel: cfg.LangfusePromptLabel,
				SavePath:    localPath,
			})
			if err == nil {
				return prompt, nil
			}
			log.Printf("Langfuse prompt '%s' unavailable (%v); attempting local fallback", cfg.LangfusePromptName, err)
		}

		if localPath != "" {
			prompt, err := langfuse.LoadPrompt(ctx, langfuse.PromptLoaderConfig{
				SavePath: localPath,
			})
			if err == nil {
				return prompt, nil
			}
			log.Printf("Failed to load system prompt from %s: %v; using built-in default", localPath, err)
		}

		return llm.DefaultSystemPrompt, nil
	}
}
