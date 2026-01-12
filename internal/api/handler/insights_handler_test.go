package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/langfuse"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// Mock services for insights handler tests

type mockChronotypeService struct{}

func (m *mockChronotypeService) Compute(ctx context.Context, userID uuid.UUID, windowDays, minSleeps int) (*domain.ChronotypeResult, error) {
	return &domain.ChronotypeResult{
		Chronotype:                   domain.ChronotypeIntermediate,
		MidSleepLocalTime:            "03:30",
		MidSleepMinutesAfterMidnight: 210,
		WindowDays:                   windowDays,
		SleepsUsed:                   10,
	}, nil
}

type mockMetricsService struct{}

func (m *mockMetricsService) Compute(ctx context.Context, userID uuid.UUID, windowDays int) (*domain.MetricsResponse, error) {
	return &domain.MetricsResponse{}, nil
}

func (m *mockMetricsService) ComputeWindow(ctx context.Context, userID uuid.UUID, from, to time.Time) (*domain.WindowMetrics, error) {
	return &domain.WindowMetrics{}, nil
}

type mockInsightsService struct{}

func (m *mockInsightsService) Generate(ctx context.Context, userID uuid.UUID) (*domain.InsightsResponse, error) {
	return &domain.InsightsResponse{
		Chronotype: domain.ChronotypeResult{
			Chronotype: domain.ChronotypeIntermediate,
		},
		Insights: domain.LLMInsightsOutput{
			Summary:      "Your sleep is good.",
			Observations: []string{"Consistent bedtime"},
			Guidance:     []string{"Keep it up"},
		},
	}, nil
}

// mockLangfuseClient for testing
type mockLangfuseClient struct {
	enabled    bool
	scoreCalls int
}

func (m *mockLangfuseClient) IsEnabled() bool {
	return m.enabled
}

func (m *mockLangfuseClient) CreateTrace(ctx context.Context, in langfuse.TraceInput) (string, error) {
	return "", nil
}

func (m *mockLangfuseClient) CreateScore(ctx context.Context, in langfuse.ScoreInput) error {
	m.scoreCalls++
	return nil
}

func TestGetInsights_IncludesTraceID(t *testing.T) {
	userID := uuid.New()

	mockLangfuse := &mockLangfuseClient{enabled: true}

	handler := NewInsightsHandler(
		&mockChronotypeService{},
		&mockMetricsService{},
		&mockInsightsService{},
		mockLangfuse,
	)

	// Setup router with chi context
	r := chi.NewRouter()
	r.Get("/users/{userId}/sleep/insights", handler.GetInsights)

	// Attach a span with a valid TraceID to the request context so the handler can pick it up.
	tp := trace.NewNoopTracerProvider()
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/sleep/insights", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response domain.InsightsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify trace_id is present (non-empty) when a span is in the context
	if response.TraceID == "" {
		t.Errorf("expected non-empty trace_id when span is present in context")
	}
}

func TestGetInsights_NoTraceIDWhenDisabled(t *testing.T) {
	userID := uuid.New()

	mockLangfuse := &mockLangfuseClient{enabled: false}

	handler := NewInsightsHandler(
		&mockChronotypeService{},
		&mockMetricsService{},
		&mockInsightsService{},
		mockLangfuse,
	)

	r := chi.NewRouter()
	r.Get("/users/{userId}/sleep/insights", handler.GetInsights)

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String()+"/sleep/insights", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Check raw JSON - trace_id should be omitted (omitempty)
	body := w.Body.String()
	if strings.Contains(body, `"trace_id"`) {
		t.Error("expected trace_id to be omitted when Langfuse is disabled")
	}
}

func TestPostFeedback_Success(t *testing.T) {
	userID := uuid.New()

	mockLangfuse := &mockLangfuseClient{enabled: true}

	handler := NewInsightsHandler(
		&mockChronotypeService{},
		&mockMetricsService{},
		&mockInsightsService{},
		mockLangfuse,
	)

	r := chi.NewRouter()
	r.Post("/users/{userId}/sleep/insights/feedback", handler.PostFeedback)

	body := `{"trace_id": "trace-123", "score": 4, "comment": "Helpful!"}`
	req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/sleep/insights/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	if mockLangfuse.scoreCalls != 1 {
		t.Errorf("expected 1 CreateScore call, got %d", mockLangfuse.scoreCalls)
	}
}

func TestPostFeedback_ValidationErrors(t *testing.T) {
	userID := uuid.New()

	handler := NewInsightsHandler(
		&mockChronotypeService{},
		&mockMetricsService{},
		&mockInsightsService{},
		&mockLangfuseClient{enabled: true},
	)

	r := chi.NewRouter()
	r.Post("/users/{userId}/sleep/insights/feedback", handler.PostFeedback)

	tests := []struct {
		name string
		body string
	}{
		{"missing trace_id", `{"score": 4}`},
		{"score too low", `{"trace_id": "abc", "score": 0}`},
		{"score too high", `{"trace_id": "abc", "score": 6}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/sleep/insights/feedback", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}
