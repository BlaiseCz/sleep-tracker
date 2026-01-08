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
	"log"
	"net/http"

	"github.com/blaisecz/sleep-tracker/internal/api"
	"github.com/blaisecz/sleep-tracker/internal/api/handler"
	"github.com/blaisecz/sleep-tracker/internal/config"
	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/llm"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/blaisecz/sleep-tracker/internal/seed"
	"github.com/blaisecz/sleep-tracker/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

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
	openaiClient := llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAISleepInsightsModel)
	if openaiClient == nil {
		log.Println("Warning: OpenAI API key not configured, insights endpoint will be unavailable")
	}

	// Initialize insights service
	insightsService := service.NewInsightsService(chronotypeService, metricsService, openaiClient, sleepLogRepo, userRepo)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	sleepLogHandler := handler.NewSleepLogHandler(sleepLogService)
	insightsHandler := handler.NewInsightsHandler(chronotypeService, metricsService, insightsService)

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
