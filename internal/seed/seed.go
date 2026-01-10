package seed

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const seededDays = 40

// Run seeds the database with sample users and sleep logs. Safe to call multiple times.
func Run(db *gorm.DB) error {
	if err := db.AutoMigrate(&domain.User{}, &domain.SleepLog{}); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	users := []domain.User{
		{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Timezone: "Europe/Amsterdam"},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Timezone: "America/New_York"},
		{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), Timezone: "Asia/Tokyo"},
		{ID: uuid.MustParse("44444444-4444-4444-4444-444444444444"), Timezone: "Australia/Sydney"},
	}

	for _, user := range users {
		if err := db.Where("id = ?", user.ID).FirstOrCreate(&user).Error; err != nil {
			return fmt.Errorf("failed to create user %s: %w", user.ID, err)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, user := range users {
		if err := seedSleepLogsForUser(db, user, rng); err != nil {
			return err
		}
	}

	log.Println("Seed completed")
	return nil
}

func seedSleepLogsForUser(db *gorm.DB, user domain.User, rng *rand.Rand) error {
	now := time.Now().UTC()
	for i := 0; i < seededDays; i++ {
		date := now.AddDate(0, 0, -i)
		bedtime := time.Date(date.Year(), date.Month(), date.Day(), 22+rng.Intn(2), rng.Intn(60), 0, 0, time.UTC)
		wakeup := bedtime.Add(time.Duration(6+rng.Intn(3)) * time.Hour)

		clientReqID := fmt.Sprintf("seed-core-%s-%d", user.ID, i)
		coreSleep := domain.SleepLog{
			UserID:          user.ID,
			StartAt:         bedtime,
			EndAt:           wakeup,
			Quality:         5 + rng.Intn(6),
			Type:            domain.SleepTypeCore,
			LocalTimezone:   user.Timezone,
			ClientRequestID: &clientReqID,
		}

		if err := db.Where("client_request_id = ?", clientReqID).FirstOrCreate(&coreSleep).Error; err != nil {
			return fmt.Errorf("failed to create core sleep log: %w", err)
		}

		if rng.Float32() < 0.5 {
			napStart := time.Date(date.Year(), date.Month(), date.Day(), 13+rng.Intn(3), rng.Intn(60), 0, 0, time.UTC)
			napEnd := napStart.Add(time.Duration(20+rng.Intn(40)) * time.Minute)

			napClientReqID := fmt.Sprintf("seed-nap-%s-%d", user.ID, i)
			napLog := domain.SleepLog{
				UserID:          user.ID,
				StartAt:         napStart,
				EndAt:           napEnd,
				Quality:         4 + rng.Intn(7),
				Type:            domain.SleepTypeNap,
				LocalTimezone:   user.Timezone,
				ClientRequestID: &napClientReqID,
			}

			if err := db.Where("client_request_id = ?", napClientReqID).FirstOrCreate(&napLog).Error; err != nil {
				return fmt.Errorf("failed to create nap log: %w", err)
			}
		}
	}
	return nil
}
