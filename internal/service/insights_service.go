package service

import (
	"context"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/llm"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/google/uuid"
)

const (
	// Window sizes for insights
	HistoryWindowDays = 30
	RecentWindowDays  = 7
)

// InsightsService generates comprehensive sleep insights.
type InsightsService interface {
	// Generate creates sleep insights for a user.
	Generate(ctx context.Context, userID uuid.UUID) (*domain.InsightsResponse, error)
}

type insightsService struct {
	chronotypeService ChronotypeService
	metricsService    MetricsService
	llmClient         llm.InsightsLLM
	sleepLogRepo      repository.SleepLogRepository
	userRepo          repository.UserRepository
}

// NewInsightsService creates a new InsightsService.
func NewInsightsService(
	chronotypeService ChronotypeService,
	metricsService MetricsService,
	llmClient llm.InsightsLLM,
	sleepLogRepo repository.SleepLogRepository,
	userRepo repository.UserRepository,
) InsightsService {
	return &insightsService{
		chronotypeService: chronotypeService,
		metricsService:    metricsService,
		llmClient:         llmClient,
		sleepLogRepo:      sleepLogRepo,
		userRepo:          userRepo,
	}
}

func (s *insightsService) Generate(ctx context.Context, userID uuid.UUID) (*domain.InsightsResponse, error) {
	// Validate user exists
	exists, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}

	now := time.Now().UTC()

	// Compute chronotype (using history window)
	chronotype, err := s.chronotypeService.Compute(ctx, userID, HistoryWindowDays, DefaultChronotypeMinSleeps)
	if err != nil {
		return nil, err
	}

	// Compute history metrics (~30 days)
	historyFrom := now.AddDate(0, 0, -HistoryWindowDays)
	historyMetrics, err := s.metricsService.ComputeWindow(ctx, userID, historyFrom, now)
	if err != nil {
		return nil, err
	}

	// Compute recent metrics (~7 days)
	recentFrom := now.AddDate(0, 0, -RecentWindowDays)
	recentMetrics, err := s.metricsService.ComputeWindow(ctx, userID, recentFrom, now)
	if err != nil {
		return nil, err
	}

	// Find the most recent day with sleep data for "last night"
	lastNightMetrics, err := s.computeLastNightMetrics(ctx, userID, now)
	if err != nil {
		return nil, err
	}

	// Build insights context for LLM
	insightsCtx := &domain.InsightsContext{
		Chronotype: *chronotype,
		History:    *historyMetrics,
		Recent:     *recentMetrics,
		LastNight:  *lastNightMetrics,
	}

	// Generate LLM insights
	llmOutput, err := s.llmClient.GenerateInsights(ctx, insightsCtx)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &domain.InsightsResponse{
		Chronotype: *chronotype,
		Insights:   *llmOutput,
	}
	response.Metrics.History = *historyMetrics
	response.Metrics.Recent = *recentMetrics
	response.Metrics.LastNight = *lastNightMetrics

	return response, nil
}

// computeLastNightMetrics finds the most recent day with sleep data and computes metrics for it.
func (s *insightsService) computeLastNightMetrics(ctx context.Context, userID uuid.UUID, now time.Time) (*domain.WindowMetrics, error) {
	// Look back up to 7 days to find the most recent day with sleep
	for daysBack := 0; daysBack < 7; daysBack++ {
		targetDate := now.AddDate(0, 0, -daysBack)
		
		// Define the day boundaries (midnight to midnight in UTC, but we'll use a generous window)
		dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := dayStart.AddDate(0, 0, 1)

		// Check if there are any logs ending on this day
		logs, err := s.sleepLogRepo.ListByEndRange(ctx, userID, dayStart, dayEnd)
		if err != nil {
			return nil, err
		}

		if len(logs) > 0 {
			// Found sleep data for this day
			return s.metricsService.ComputeWindow(ctx, userID, dayStart, dayEnd)
		}
	}

	// No recent sleep data found, return empty metrics for today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.AddDate(0, 0, 1)
	
	return &domain.WindowMetrics{
		From: todayStart,
		To:   todayEnd,
	}, nil
}
