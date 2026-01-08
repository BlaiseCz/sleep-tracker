package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestSleepLogHandler_Update(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		logID          string
		body           string
		mockService    *MockSleepLogService
		wantStatusCode int
	}{
		{
			name:   "update quality only",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"quality": 9}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return &domain.SleepLog{
						ID:            lid,
						UserID:        uid,
						StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
						EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
						Quality:       9,
						Type:          domain.SleepTypeCore,
						LocalTimezone: "UTC",
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "update type",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"type": "NAP"}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return &domain.SleepLog{
						ID:            lid,
						UserID:        uid,
						StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
						EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
						Quality:       8,
						Type:          domain.SleepTypeNap,
						LocalTimezone: "UTC",
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "update times",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"start_at": "2024-01-15T22:00:00Z", "end_at": "2024-01-16T06:00:00Z"}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return &domain.SleepLog{
						ID:            lid,
						UserID:        uid,
						StartAt:       *req.StartAt,
						EndAt:         *req.EndAt,
						Quality:       8,
						Type:          domain.SleepTypeCore,
						LocalTimezone: "UTC",
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "update timezone",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"local_timezone": "America/New_York"}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return &domain.SleepLog{
						ID:            lid,
						UserID:        uid,
						StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
						EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
						Quality:       8,
						Type:          domain.SleepTypeCore,
						LocalTimezone: "America/New_York",
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid user ID",
			userID:         "not-a-uuid",
			logID:          logID.String(),
			body:           `{"quality": 9}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid log ID",
			userID:         userID.String(),
			logID:          "not-a-uuid",
			body:           `{"quality": 9}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			userID:         userID.String(),
			logID:          logID.String(),
			body:           `{invalid}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "quality too low",
			userID:         userID.String(),
			logID:          logID.String(),
			body:           `{"quality": 0}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "quality too high",
			userID:         userID.String(),
			logID:          logID.String(),
			body:           `{"quality": 11}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "invalid type",
			userID:         userID.String(),
			logID:          logID.String(),
			body:           `{"type": "INVALID"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:   "log not found",
			userID: userID.String(),
			logID:  uuid.New().String(),
			body:   `{"quality": 9}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return nil, domain.ErrNotFound
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:   "overlapping sleep",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"start_at": "2024-01-16T22:00:00Z", "end_at": "2024-01-17T06:00:00Z"}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return nil, domain.ErrOverlappingSleep
				},
			},
			wantStatusCode: http.StatusConflict,
		},
		{
			name:   "invalid time range",
			userID: userID.String(),
			logID:  logID.String(),
			body:   `{"end_at": "2024-01-15T20:00:00Z"}`,
			mockService: &MockSleepLogService{
				updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
					return nil, domain.ErrInvalidInput
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSleepLogHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPut, "/v1/users/"+tt.userID+"/sleep-logs/"+tt.logID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Add chi URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tt.userID)
			rctx.URLParams.Add("logId", tt.logID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.Update(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Update() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
			}
		})
	}
}

func TestSleepLogHandler_Update_ResponseFormat(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	mockService := &MockSleepLogService{
		updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
			return &domain.SleepLog{
				ID:            lid,
				UserID:        uid,
				StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality:       9,
				Type:          domain.SleepTypeCore,
				LocalTimezone: "America/New_York",
				CreatedAt:     time.Date(2024, 1, 16, 7, 5, 0, 0, time.UTC),
			}, nil
		},
	}

	handler := NewSleepLogHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/v1/users/"+userID.String()+"/sleep-logs/"+logID.String(), bytes.NewBufferString(`{"quality": 9}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userID.String())
	rctx.URLParams.Add("logId", logID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Update() status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response domain.SleepLogResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if response.ID != logID {
		t.Errorf("ID = %v, want %v", response.ID, logID)
	}
	if response.UserID != userID {
		t.Errorf("UserID = %v, want %v", response.UserID, userID)
	}
	if response.Quality != 9 {
		t.Errorf("Quality = %d, want 9", response.Quality)
	}
	if response.Type != domain.SleepTypeCore {
		t.Errorf("Type = %s, want CORE", response.Type)
	}
	if response.LocalTimezone != "America/New_York" {
		t.Errorf("LocalTimezone = %s, want America/New_York", response.LocalTimezone)
	}

	// Verify local times are converted
	if response.LocalStartAt.IsZero() {
		t.Error("LocalStartAt should not be zero")
	}
	if response.LocalEndAt.IsZero() {
		t.Error("LocalEndAt should not be zero")
	}
}

func TestSleepLogHandler_Update_EmptyBody(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	mockService := &MockSleepLogService{
		updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
			// Empty update should still work (no changes)
			return &domain.SleepLog{
				ID:            lid,
				UserID:        uid,
				StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          domain.SleepTypeCore,
				LocalTimezone: "UTC",
			}, nil
		},
	}

	handler := NewSleepLogHandler(mockService)

	req := httptest.NewRequest(http.MethodPut, "/v1/users/"+userID.String()+"/sleep-logs/"+logID.String(), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userID.String())
	rctx.URLParams.Add("logId", logID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Update() with empty body status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSleepLogHandler_Update_TravelScenario(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	// Scenario: User logged sleep in wrong timezone, needs to correct it
	mockService := &MockSleepLogService{
		updateFunc: func(ctx context.Context, uid uuid.UUID, lid uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
			return &domain.SleepLog{
				ID:            lid,
				UserID:        uid,
				StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          domain.SleepTypeCore,
				LocalTimezone: *req.LocalTimezone,
			}, nil
		},
	}

	handler := NewSleepLogHandler(mockService)

	// Update timezone from UTC to San Francisco
	body := `{"local_timezone": "America/Los_Angeles"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/users/"+userID.String()+"/sleep-logs/"+logID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userID.String())
	rctx.URLParams.Add("logId", logID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Update() status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response domain.SleepLogResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.LocalTimezone != "America/Los_Angeles" {
		t.Errorf("LocalTimezone = %s, want America/Los_Angeles", response.LocalTimezone)
	}

	// Verify local times are in SF timezone (10 PM to 9 AM)
	if response.LocalStartAt.Hour() != 22 {
		t.Errorf("LocalStartAt hour = %d, want 22 (10 PM SF)", response.LocalStartAt.Hour())
	}
	if response.LocalEndAt.Hour() != 9 {
		t.Errorf("LocalEndAt hour = %d, want 9 (9 AM SF)", response.LocalEndAt.Hour())
	}
}
