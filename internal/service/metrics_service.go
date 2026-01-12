package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// DefaultMetricsWindowDays is the default window for metrics calculation.
	DefaultMetricsWindowDays = 30

	// DefaultTargetHours is the default daily sleep target.
	DefaultTargetHours = 7.0
)

// MetricsService computes sleep metrics from sleep logs.
type MetricsService interface {
	// Compute calculates metrics for a user over the given window.
	Compute(ctx context.Context, userID uuid.UUID, windowDays int) (*domain.MetricsResponse, error)
	// ComputeWindow calculates WindowMetrics for a specific time range.
	ComputeWindow(ctx context.Context, userID uuid.UUID, from, to time.Time) (*domain.WindowMetrics, error)
}

type metricsService struct {
	sleepLogRepo repository.SleepLogRepository
	userRepo     repository.UserRepository
}

// NewMetricsService creates a new MetricsService.
func NewMetricsService(sleepLogRepo repository.SleepLogRepository, userRepo repository.UserRepository) MetricsService {
	return &metricsService{
		sleepLogRepo: sleepLogRepo,
		userRepo:     userRepo,
	}
}

func (s *metricsService) Compute(ctx context.Context, userID uuid.UUID, windowDays int) (*domain.MetricsResponse, error) {
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
		windowDays = DefaultMetricsWindowDays
	}

	// Calculate time window
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -windowDays)

	// Compute window metrics
	windowMetrics, err := s.ComputeWindow(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &domain.MetricsResponse{
		PerSleep:     windowMetrics.PerSleep,
		DailyOverall: windowMetrics.DailyOverall,
		Scores:       windowMetrics.Scores,
	}
	response.Window.From = windowMetrics.From
	response.Window.To = windowMetrics.To

	return response, nil
}

func (s *metricsService) ComputeWindow(ctx context.Context, userID uuid.UUID, from, to time.Time) (*domain.WindowMetrics, error) {
	tracer := otel.Tracer("sleep-tracker-api/metrics")
	ctx, span := tracer.Start(ctx, "MetricsService.ComputeWindow",
		trace.WithAttributes(
			attribute.String("user.id", userID.String()),
			attribute.String("window.from", from.Format(time.RFC3339)),
			attribute.String("window.to", to.Format(time.RFC3339)),
		),
	)
	defer span.End()

	// Derive window length in days for readability
	windowDuration := to.Sub(from)
	windowDays := int(windowDuration.Hours() / 24)
	if windowDays < 1 {
		windowDays = 1
	}
	span.SetAttributes(
		attribute.Int("window.days", windowDays),
		attribute.String("window.description", fmt.Sprintf("%dd window", windowDays)),
	)

	// Attach input payload for Langfuse
	inputPayload := map[string]any{
		"user_id":     userID.String(),
		"from":        from.Format(time.RFC3339),
		"to":          to.Format(time.RFC3339),
		"window_days": windowDays,
	}
	if inputJSON, err := json.Marshal(inputPayload); err == nil {
		span.SetAttributes(attribute.String("langfuse.observation.input", string(inputJSON)))
	}

	// Fetch sleep logs in the window (by EndAt)
	logs, err := s.sleepLogRepo.ListByEndRange(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	result := &domain.WindowMetrics{
		From: from,
		To:   to,
	}

	// Calculate per-sleep metrics
	result.PerSleep = computePerSleepMetrics(logs)

	// Calculate per-day metrics
	result.DailyOverall = computeDailyOverallMetrics(logs)

	// Calculate derived scores
	result.Scores = computeDerivedScores(result.PerSleep, result.DailyOverall)

	// Attach output payload for Langfuse
	if outputJSON, err := json.Marshal(result); err == nil {
		span.SetAttributes(attribute.String("langfuse.observation.output", string(outputJSON)))
	}

	return result, nil
}

// sleepData holds extracted data from a single sleep log.
type sleepData struct {
	durationHours  float64
	bedtimeMinutes int
	quality        int
	localDate      string // YYYY-MM-DD format for grouping
}

// extractSleepData extracts relevant data from a sleep log.
func extractSleepData(log domain.SleepLog) sleepData {
	loc := time.UTC
	if log.LocalTimezone != "" {
		if l, err := time.LoadLocation(log.LocalTimezone); err == nil {
			loc = l
		}
	}

	startLocal := log.StartAt.In(loc)
	endLocal := log.EndAt.In(loc)
	durationMinutes := endLocal.Sub(startLocal).Minutes()

	// Bedtime is minutes after midnight of the start time
	bedtimeMinutes := startLocal.Hour()*60 + startLocal.Minute()

	// Local date is based on EndAt (the day the sleep "belongs to")
	localDate := endLocal.Format("2006-01-02")

	return sleepData{
		durationHours:  durationMinutes / 60.0,
		bedtimeMinutes: bedtimeMinutes,
		quality:        log.Quality,
		localDate:      localDate,
	}
}

// computePerSleepMetrics calculates per-sleep statistics.
func computePerSleepMetrics(logs []domain.SleepLog) domain.PerSleepMetrics {
	result := domain.PerSleepMetrics{}

	if len(logs) == 0 {
		return result
	}

	var durations []float64
	var qualities []float64
	var bedtimes []float64

	for _, log := range logs {
		data := extractSleepData(log)

		// Filter out extremely short logs (< 90 minutes = 1.5 hours)
		if data.durationHours < float64(MinDurationMinutes)/60.0 {
			continue
		}

		durations = append(durations, data.durationHours)
		qualities = append(qualities, float64(data.quality))
		bedtimes = append(bedtimes, float64(data.bedtimeMinutes))
	}

	result.SleepCount = len(durations)

	if len(durations) > 0 {
		result.Duration = computeStats(durations)
		result.Quality = computeStats(qualities)
		result.Bedtime = computeStats(bedtimes)
	}

	return result
}

// computeDailyOverallMetrics calculates per-day total sleep statistics.
func computeDailyOverallMetrics(logs []domain.SleepLog) domain.DailyOverallMetrics {
	result := domain.DailyOverallMetrics{
		TargetHours: DefaultTargetHours,
	}

	if len(logs) == 0 {
		return result
	}

	// Group logs by local date and sum durations
	dailyTotals := make(map[string]float64)
	for _, log := range logs {
		data := extractSleepData(log)
		dailyTotals[data.localDate] += data.durationHours
	}

	if len(dailyTotals) == 0 {
		return result
	}

	// Convert to slice for statistics
	var totals []float64
	daysMeetingTarget := 0
	for _, total := range dailyTotals {
		totals = append(totals, total)
		if total >= DefaultTargetHours {
			daysMeetingTarget++
		}
	}

	result.DaysCount = len(totals)
	result.TotalDailyHours = computeStats(totals)
	result.DaysMeetingTarget = daysMeetingTarget

	// Calculate sufficiency score (percentage of days meeting target)
	if result.DaysCount > 0 {
		result.DailySufficiencyScore = math.Round(float64(daysMeetingTarget)/float64(result.DaysCount)*1000) / 10
	}

	return result
}

// computeDerivedScores calculates 0-100 scores from metrics.
func computeDerivedScores(perSleep domain.PerSleepMetrics, dailyOverall domain.DailyOverallMetrics) domain.DerivedScores {
	scores := domain.DerivedScores{}

	// Consistency score: based on bedtime variability (lower std = higher score)
	// Map std of 0-120 minutes to score of 100-0
	if perSleep.SleepCount > 0 {
		bedtimeStd := perSleep.Bedtime.Std
		// Clamp to reasonable range
		if bedtimeStd > 120 {
			bedtimeStd = 120
		}
		scores.ConsistencyScore = math.Round((1-bedtimeStd/120)*1000) / 10
		if scores.ConsistencyScore < 0 {
			scores.ConsistencyScore = 0
		}
	}

	// Sufficiency score: based on average duration meeting target
	// Map avg duration of 5-9 hours to score of 0-100
	if perSleep.SleepCount > 0 {
		avgDuration := perSleep.Duration.Avg
		if avgDuration < 5 {
			scores.SufficiencyScore = 0
		} else if avgDuration >= 9 {
			scores.SufficiencyScore = 100
		} else {
			scores.SufficiencyScore = math.Round((avgDuration-5)/4*1000) / 10
		}
	}

	// Overall sleep score: weighted combination
	// 40% consistency, 30% sufficiency, 30% daily sufficiency
	scores.OverallSleepScore = math.Round(
		(scores.ConsistencyScore*0.4+
			scores.SufficiencyScore*0.3+
			dailyOverall.DailySufficiencyScore*0.3)*10) / 10

	return scores
}

// computeStats calculates descriptive statistics for a slice of values.
func computeStats(values []float64) domain.DescriptiveStats {
	if len(values) == 0 {
		return domain.DescriptiveStats{}
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))

	// Calculate min/max
	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Calculate standard deviation
	sumSquares := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	std := 0.0
	if len(values) > 1 {
		std = math.Sqrt(sumSquares / float64(len(values)-1))
	}

	return domain.DescriptiveStats{
		Avg: math.Round(avg*100) / 100,
		Std: math.Round(std*100) / 100,
		Min: math.Round(minVal*100) / 100,
		Max: math.Round(maxVal*100) / 100,
	}
}
