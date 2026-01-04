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

// MockSleepLogService is a mock implementation of SleepLogService
type MockSleepLogService struct {
	createFunc func(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error)
	listFunc   func(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error)
}

func (m *MockSleepLogService) Create(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, req)
	}
	return &domain.SleepLog{
		ID:      uuid.New(),
		UserID:  userID,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
		Quality: req.Quality,
		Type:    req.Type,
	}, false, nil
}

func (m *MockSleepLogService) List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, filter)
	}
	return &domain.SleepLogListResponse{
		Data:       []domain.SleepLogResponse{},
		Pagination: domain.PaginationResponse{HasMore: false},
	}, nil
}

func TestSleepLogHandler_Create(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		body           string
		mockService    *MockSleepLogService
		wantStatusCode int
	}{
		{
			name:   "valid CORE sleep",
			userID: userID.String(),
			body:   `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "CORE"}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:      uuid.New(),
						UserID:  uid,
						StartAt: req.StartAt,
						EndAt:   req.EndAt,
						Quality: req.Quality,
						Type:    req.Type,
					}, false, nil
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:   "valid NAP",
			userID: userID.String(),
			body:   `{"start_at": "2024-01-16T14:00:00Z", "end_at": "2024-01-16T15:00:00Z", "quality": 6, "type": "NAP"}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:      uuid.New(),
						UserID:  uid,
						StartAt: req.StartAt,
						EndAt:   req.EndAt,
						Quality: req.Quality,
						Type:    req.Type,
					}, false, nil
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "invalid user ID",
			userID:         "not-a-uuid",
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "CORE"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			userID:         userID.String(),
			body:           `{invalid}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "quality too low",
			userID:         userID.String(),
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 0, "type": "CORE"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "quality too high",
			userID:         userID.String(),
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 11, "type": "CORE"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid type",
			userID:         userID.String(),
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "INVALID"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: uuid.New().String(),
			body:   `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "CORE"}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return nil, false, domain.ErrNotFound
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:   "overlapping sleep",
			userID: userID.String(),
			body:   `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "CORE"}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return nil, false, domain.ErrOverlappingSleep
				},
			},
			wantStatusCode: http.StatusConflict,
		},
		{
			name:   "idempotent request returns 200",
			userID: userID.String(),
			body:   `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "CORE", "client_request_id": "req-123"}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:              uuid.New(),
						UserID:          uid,
						StartAt:         req.StartAt,
						EndAt:           req.EndAt,
						Quality:         req.Quality,
						Type:            req.Type,
						ClientRequestID: req.ClientRequestID,
					}, true, nil // isExisting = true
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSleepLogHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPost, "/v1/users/"+tt.userID+"/sleep-logs", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Add chi URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.Create(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Create() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
			}
		})
	}
}

func TestSleepLogHandler_List(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockService    *MockSleepLogService
		wantStatusCode int
	}{
		{
			name:        "list all logs",
			userID:      userID.String(),
			queryParams: "",
			mockService: &MockSleepLogService{
				listFunc: func(ctx context.Context, uid uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
					return &domain.SleepLogListResponse{
						Data: []domain.SleepLogResponse{
							{
								ID:      uuid.New(),
								UserID:  uid,
								StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
								EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
								Quality: 8,
								Type:    domain.SleepTypeCore,
							},
						},
						Pagination: domain.PaginationResponse{HasMore: false},
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "list with filters",
			userID:      userID.String(),
			queryParams: "?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z&limit=10",
			mockService: &MockSleepLogService{
				listFunc: func(ctx context.Context, uid uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
					// Verify filters are parsed
					if filter.From == nil || filter.To == nil {
						t.Error("Expected from and to filters to be set")
					}
					if filter.Limit != 10 {
						t.Errorf("Expected limit 10, got %d", filter.Limit)
					}
					return &domain.SleepLogListResponse{
						Data:       []domain.SleepLogResponse{},
						Pagination: domain.PaginationResponse{HasMore: false},
					}, nil
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid user ID",
			userID:         "not-a-uuid",
			queryParams:    "",
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid from parameter",
			userID:         userID.String(),
			queryParams:    "?from=invalid-date",
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: uuid.New().String(),
			mockService: &MockSleepLogService{
				listFunc: func(ctx context.Context, uid uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
					return nil, domain.ErrNotFound
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSleepLogHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodGet, "/v1/users/"+tt.userID+"/sleep-logs"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			// Add chi URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.List(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("List() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
			}

			// Verify response structure for successful requests
			if tt.wantStatusCode == http.StatusOK {
				var response domain.SleepLogListResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
			}
		})
	}
}
