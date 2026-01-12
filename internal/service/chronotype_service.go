package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// MinDurationMinutes is the minimum sleep duration to consider (90 minutes).
	MinDurationMinutes = 90

	// Default values for chronotype calculation
	DefaultChronotypeWindowDays = 30
	DefaultChronotypeMinSleeps  = 7

	// Chronotype thresholds (minutes after midnight for mid-sleep)
	EarlyBirdThreshold    = 150 // < 150 = early bird (mid-sleep before 2:30 AM)
	IntermediateThreshold = 270 // 150-269 = intermediate, >= 270 = night owl (4:30 AM)
)

// ChronotypeService computes chronotype from sleep logs.
type ChronotypeService interface {
	// Compute calculates the user's chronotype based on sleep logs in the given window.
	Compute(ctx context.Context, userID uuid.UUID, windowDays, minSleeps int) (*domain.ChronotypeResult, error)
}

type chronotypeService struct {
	sleepLogRepo repository.SleepLogRepository
	userRepo     repository.UserRepository
}

// NewChronotypeService creates a new ChronotypeService.
func NewChronotypeService(sleepLogRepo repository.SleepLogRepository, userRepo repository.UserRepository) ChronotypeService {
	return &chronotypeService{
		sleepLogRepo: sleepLogRepo,
		userRepo:     userRepo,
	}
}

func (s *chronotypeService) Compute(ctx context.Context, userID uuid.UUID, windowDays, minSleeps int) (*domain.ChronotypeResult, error) {
	tracer := otel.Tracer("sleep-tracker-api/chronotype")
	ctx, span := tracer.Start(ctx, "ChronotypeService.Compute")
	defer span.End()

	// Validate user exists
	exists, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Apply defaults
	if windowDays <= 0 {
		windowDays = DefaultChronotypeWindowDays
	}
	if minSleeps <= 0 {
		minSleeps = DefaultChronotypeMinSleeps
	}

	span.SetAttributes(
		attribute.String("user.id", userID.String()),
		attribute.Int("window_days", windowDays),
		attribute.Int("min_sleeps", minSleeps),
		attribute.String("window.description", fmt.Sprintf("%dd window", windowDays)),
	)

	// Calculate time window
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -windowDays)

	// Attach input payload for Langfuse
	inputPayload := map[string]any{
		"user_id":     userID.String(),
		"window_days": windowDays,
		"min_sleeps":  minSleeps,
		"from":        from.Format(time.RFC3339),
		"to":          now.Format(time.RFC3339),
	}
	if inputJSON, err := json.Marshal(inputPayload); err == nil {
		span.SetAttributes(attribute.String("langfuse.observation.input", string(inputJSON)))
	}

	// Fetch sleep logs in the window (by EndAt)
	logs, err := s.sleepLogRepo.ListByEndRange(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}

	// Calculate mid-sleep minutes for each valid log
	var midMinutes []int
	for _, log := range logs {
		// Convert to local timezone
		loc := time.UTC
		if log.LocalTimezone != "" {
			if l, err := time.LoadLocation(log.LocalTimezone); err == nil {
				loc = l
			}
		}

		startLocal := log.StartAt.In(loc)
		endLocal := log.EndAt.In(loc)
		durationMinutes := endLocal.Sub(startLocal).Minutes()

		// Filter out extremely short logs (< 90 minutes)
		if durationMinutes < MinDurationMinutes {
			continue
		}

		// Calculate mid-sleep time
		midSleep := startLocal.Add(time.Duration(durationMinutes/2) * time.Minute)
		midMin := midSleepMinutesAfterMidnight(midSleep)
		midMinutes = append(midMinutes, midMin)
	}

	// Build result
	result := &domain.ChronotypeResult{
		WindowDays: windowDays,
		SleepsUsed: len(midMinutes),
	}

	// If not enough valid sleeps, return unknown
	if len(midMinutes) < minSleeps {
		result.Chronotype = domain.ChronotypeUnknown
		result.MidSleepLocalTime = ""
		result.MidSleepMinutesAfterMidnight = 0
		return result, nil
	}

	// Compute median of mid-sleep minutes
	medianMid := median(midMinutes)
	result.MidSleepMinutesAfterMidnight = medianMid
	result.MidSleepLocalTime = minutesToTimeString(medianMid)

	// Classify chronotype
	result.Chronotype = classifyChronotype(medianMid)

	// Attach output payload for Langfuse
	if outputJSON, err := json.Marshal(result); err == nil {
		span.SetAttributes(attribute.String("langfuse.observation.output", string(outputJSON)))
	}

	return result, nil
}

// midSleepMinutesAfterMidnight calculates minutes after midnight for a given time.
// Handles times that span midnight (e.g., 11 PM to 7 AM).
func midSleepMinutesAfterMidnight(t time.Time) int {
	hour := t.Hour()
	minute := t.Minute()
	return hour*60 + minute
}

// median calculates the median of a slice of integers.
func median(values []int) int {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// minutesToTimeString converts minutes after midnight to HH:MM format.
func minutesToTimeString(minutes int) string {
	// Handle negative or > 24h values
	minutes = ((minutes % 1440) + 1440) % 1440
	h := minutes / 60
	m := minutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// classifyChronotype determines chronotype based on mid-sleep minutes.
func classifyChronotype(midMinutes int) domain.ChronotypeType {
	if midMinutes < EarlyBirdThreshold {
		return domain.ChronotypeEarlyBird
	}
	if midMinutes < IntermediateThreshold {
		return domain.ChronotypeIntermediate
	}
	return domain.ChronotypeNightOwl
}
