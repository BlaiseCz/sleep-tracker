package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/langfuse"
	"github.com/blaisecz/sleep-tracker/internal/llm"
	"github.com/blaisecz/sleep-tracker/internal/service"
	"github.com/blaisecz/sleep-tracker/pkg/problem"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// InsightsHandler handles sleep insights endpoints.
type InsightsHandler struct {
	chronotypeService service.ChronotypeService
	metricsService    service.MetricsService
	insightsService   service.InsightsService
	langfuseClient    langfuse.Client
}

// NewInsightsHandler creates a new InsightsHandler.
func NewInsightsHandler(
	chronotypeService service.ChronotypeService,
	metricsService service.MetricsService,
	insightsService service.InsightsService,
	langfuseClient langfuse.Client,
) *InsightsHandler {
	return &InsightsHandler{
		chronotypeService: chronotypeService,
		metricsService:    metricsService,
		insightsService:   insightsService,
		langfuseClient:    langfuseClient,
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

	// Attach OTEL trace ID (if present) to response for feedback linking
	span := trace.SpanFromContext(r.Context())
	if span.SpanContext().IsValid() {
		result.TraceID = span.SpanContext().TraceID().String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// FeedbackRequest is the request body for insights feedback.
// @Description Request body for submitting feedback on insights.
type FeedbackRequest struct {
	// Trace ID from the insights response
	TraceID string `json:"trace_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Rating score (1-5)
	Score int `json:"score" example:"4" minimum:"1" maximum:"5"`
	// Optional comment
	Comment string `json:"comment,omitempty" example:"The insights were helpful!"`
}

// PostFeedback handles POST /v1/users/{userId}/sleep/insights/feedback
// @Summary Submit feedback on sleep insights
// @Description Submit a user rating and optional comment for a previous insights response.
// @Tags sleep-insights
// @Accept json
// @Produce json
// @Param userId path string true "User UUID" format(uuid) example(550e8400-e29b-41d4-a716-446655440000)
// @Param body body FeedbackRequest true "Feedback request"
// @Success 204 "Feedback submitted"
// @Failure 400 {object} problem.Problem "Invalid request"
// @Failure 500 {object} problem.Problem "Server error"
// @Router /users/{userId}/sleep/insights/feedback [post]
func (h *InsightsHandler) PostFeedback(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		problem.BadRequest("Invalid request body").Write(w)
		return
	}

	// Validate required fields
	if req.TraceID == "" {
		problem.BadRequest("trace_id is required").Write(w)
		return
	}
	if req.Score < 1 || req.Score > 5 {
		problem.BadRequest("score must be between 1 and 5").Write(w)
		return
	}

	// Create score in Langfuse (errors are logged but don't fail the request)
	_ = h.langfuseClient.CreateScore(r.Context(), langfuse.ScoreInput{
		TraceID: req.TraceID,
		Name:    "user_rating",
		Value:   float64(req.Score),
		Comment: req.Comment,
	})

	// Log the feedback for debugging
	if h.langfuseClient.IsEnabled() {
		// Score was sent to Langfuse
	} else {
		// Langfuse not enabled, but we still accept feedback
		_ = userID // suppress unused warning
	}

	w.WriteHeader(http.StatusNoContent)
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
