package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/llm"
	"github.com/blaisecz/sleep-tracker/internal/service"
	"github.com/blaisecz/sleep-tracker/pkg/problem"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// InsightsHandler handles sleep insights endpoints.
type InsightsHandler struct {
	chronotypeService service.ChronotypeService
	metricsService    service.MetricsService
	insightsService   service.InsightsService
}

// NewInsightsHandler creates a new InsightsHandler.
func NewInsightsHandler(
	chronotypeService service.ChronotypeService,
	metricsService service.MetricsService,
	insightsService service.InsightsService,
) *InsightsHandler {
	return &InsightsHandler{
		chronotypeService: chronotypeService,
		metricsService:    metricsService,
		insightsService:   insightsService,
	}
}

// GetChronotype handles GET /v1/users/{userId}/sleep/chronotype
// @Summary Get user chronotype
// @Description Compute the user's chronotype based on their sleep patterns over a configurable window.
// @Tags sleep-insights
// @Produce json
// @Param userId path string true "User UUID" format(uuid) example(550e8400-e29b-41d4-a716-446655440000)
// @Param window_days query integer false "Number of days to analyze" default(30) minimum(1) maximum(365)
// @Param min_sleeps query integer false "Minimum sleep logs required" default(7) minimum(1) maximum(100)
// @Success 200 {object} domain.ChronotypeResult "Chronotype analysis result"
// @Failure 400 {object} problem.Problem "Invalid query parameters"
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 500 {object} problem.Problem "Server error"
// @Router /users/{userId}/sleep/chronotype [get]
func (h *InsightsHandler) GetChronotype(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	// Parse query parameters
	windowDays := parseIntParam(r, "window_days", service.DefaultChronotypeWindowDays)
	minSleeps := parseIntParam(r, "min_sleeps", service.DefaultChronotypeMinSleeps)

	// Validate parameters
	if windowDays < 1 || windowDays > 365 {
		problem.BadRequest("window_days must be between 1 and 365").Write(w)
		return
	}
	if minSleeps < 1 || minSleeps > 100 {
		problem.BadRequest("min_sleeps must be between 1 and 100").Write(w)
		return
	}

	result, err := h.chronotypeService.Compute(r.Context(), userID, windowDays, minSleeps)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		problem.InternalError("Failed to compute chronotype").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetMetrics handles GET /v1/users/{userId}/sleep/metrics
// @Summary Get sleep metrics
// @Description Compute per-sleep and per-day sleep metrics over a configurable window.
// @Tags sleep-insights
// @Produce json
// @Param userId path string true "User UUID" format(uuid) example(550e8400-e29b-41d4-a716-446655440000)
// @Param window_days query integer false "Number of days to analyze" default(30) minimum(1) maximum(365)
// @Success 200 {object} domain.MetricsResponse "Sleep metrics"
// @Failure 400 {object} problem.Problem "Invalid query parameters"
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 500 {object} problem.Problem "Server error"
// @Router /users/{userId}/sleep/metrics [get]
func (h *InsightsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	// Parse query parameters
	windowDays := parseIntParam(r, "window_days", service.DefaultMetricsWindowDays)

	// Validate parameters
	if windowDays < 1 || windowDays > 365 {
		problem.BadRequest("window_days must be between 1 and 365").Write(w)
		return
	}

	result, err := h.metricsService.Compute(r.Context(), userID, windowDays)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		problem.InternalError("Failed to compute metrics").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetInsights handles GET /v1/users/{userId}/sleep/insights
// @Summary Get LLM-powered sleep insights
// @Description Generate comprehensive sleep insights using chronotype, metrics, and LLM analysis.
// @Tags sleep-insights
// @Produce json
// @Param userId path string true "User UUID" format(uuid) example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} domain.InsightsResponse "Sleep insights with LLM analysis"
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 500 {object} problem.Problem "Server error"
// @Failure 503 {object} problem.Problem "LLM service unavailable"
// @Router /users/{userId}/sleep/insights [get]
func (h *InsightsHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	result, err := h.insightsService.Generate(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		if errors.Is(err, llm.ErrOpenAIUnavailable) {
			problem.New(http.StatusServiceUnavailable, "service-unavailable", "Service Unavailable", "OpenAI service is not configured").Write(w)
			return
		}
		if errors.Is(err, llm.ErrOpenAIRequest) || errors.Is(err, llm.ErrOpenAIResponse) {
			problem.New(http.StatusBadGateway, "llm-error", "LLM Error", "Failed to generate insights from LLM").Write(w)
			return
		}
		problem.InternalError("Failed to generate insights").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(r *http.Request, name string, defaultValue int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}
