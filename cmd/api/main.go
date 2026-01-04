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
	"github.com/blaisecz/sleep-tracker/internal/repository"
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

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sleepLogRepo := repository.NewSleepLogRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo)
	sleepLogService := service.NewSleepLogService(sleepLogRepo, userRepo)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	sleepLogHandler := handler.NewSleepLogHandler(sleepLogService)

	// Setup router
	router := api.NewRouter(userHandler, sleepLogHandler)
	handler := router.Setup()

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
