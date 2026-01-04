package main

import (
	"log"

	"github.com/blaisecz/sleep-tracker/internal/config"
	"github.com/blaisecz/sleep-tracker/internal/seed"
)

func main() {
	cfg := config.Load()

	db, err := config.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := seed.Run(db); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}
}
