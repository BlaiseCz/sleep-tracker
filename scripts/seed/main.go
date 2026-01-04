package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/config"
	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()

	db, err := config.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&domain.User{}, &domain.SleepLog{}); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	// Create sample users
	users := []domain.User{
		{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Timezone: "Europe/Amsterdam"},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Timezone: "America/New_York"},
		{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), Timezone: "Asia/Tokyo"},
	}

	for _, user := range users {
		result := db.FirstOrCreate(&user, domain.User{ID: user.ID})
		if result.Error != nil {
			log.Printf("Failed to create user %s: %v", user.ID, result.Error)
		} else {
			log.Printf("User %s (%s) ready", user.ID, user.Timezone)
		}
	}

	// Create sample sleep logs for the past 14 days
	sleepTypes := []domain.SleepType{domain.SleepTypeCore, domain.SleepTypeNap}
	now := time.Now()

	for _, user := range users {
		for i := 0; i < 14; i++ {
			// Core sleep (night)
			date := now.AddDate(0, 0, -i)
			bedtime := time.Date(date.Year(), date.Month(), date.Day(), 22+rand.Intn(2), rand.Intn(60), 0, 0, time.UTC)
			wakeup := bedtime.Add(time.Duration(6+rand.Intn(3)) * time.Hour)

			clientReqID := fmt.Sprintf("seed-core-%s-%d", user.ID, i)
			sleepLog := domain.SleepLog{
				UserID:          user.ID,
				StartAt:         bedtime,
				EndAt:           wakeup,
				Quality:         5 + rand.Intn(6), // 5-10
				Type:            domain.SleepTypeCore,
				LocalTimezone:   user.Timezone,
				ClientRequestID: &clientReqID,
			}

			result := db.FirstOrCreate(&sleepLog, domain.SleepLog{ClientRequestID: &clientReqID})
			if result.Error != nil {
				log.Printf("Failed to create sleep log: %v", result.Error)
			}

			// Random nap (50% chance)
			if rand.Float32() < 0.5 {
				napStart := time.Date(date.Year(), date.Month(), date.Day(), 13+rand.Intn(3), rand.Intn(60), 0, 0, time.UTC)
				napEnd := napStart.Add(time.Duration(20+rand.Intn(40)) * time.Minute)

				napClientReqID := fmt.Sprintf("seed-nap-%s-%d", user.ID, i)
				napLog := domain.SleepLog{
					UserID:          user.ID,
					StartAt:         napStart,
					EndAt:           napEnd,
					Quality:         4 + rand.Intn(7), // 4-10
					Type:            sleepTypes[1],
					LocalTimezone:   user.Timezone,
					ClientRequestID: &napClientReqID,
				}

				result := db.FirstOrCreate(&napLog, domain.SleepLog{ClientRequestID: &napClientReqID})
				if result.Error != nil {
					log.Printf("Failed to create nap log: %v", result.Error)
				}
			}
		}
		log.Printf("Created sleep logs for user %s", user.ID)
	}

	log.Println("Seed completed!")
	fmt.Println("\nSample user IDs for testing:")
	for _, user := range users {
		fmt.Printf("  %s (%s)\n", user.ID, user.Timezone)
	}
}
