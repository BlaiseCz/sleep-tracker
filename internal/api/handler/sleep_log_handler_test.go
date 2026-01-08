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

// MockSleepLogService is defined in mocks_test.go

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
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "quality too high",
			userID:         userID.String(),
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 11, "type": "CORE"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "invalid type",
			userID:         userID.String(),
			body:           `{"start_at": "2024-01-15T23:00:00Z", "end_at": "2024-01-16T07:00:00Z", "quality": 8, "type": "INVALID"}`,
			mockService:    &MockSleepLogService{},
			wantStatusCode: http.StatusUnprocessableEntity,
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

// TestSleepLogHandler_Create_TravelScenario tests the Poznan → San Francisco scenario at HTTP level
func TestSleepLogHandler_Create_TravelScenario(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name               string
		body               string
		mockService        *MockSleepLogService
		wantStatusCode     int
		wantLocalTimezone  string
		wantLocalStartHour int
		wantLocalEndHour   int
	}{
		{
			name: "11h sleep in San Francisco after flight from Poznan",
			// 10 PM Jan 15 SF (UTC-8) = 06:00 UTC Jan 16
			// 9 AM Jan 16 SF = 17:00 UTC Jan 16
			body: `{
				"start_at": "2024-01-16T06:00:00Z",
				"end_at": "2024-01-16T17:00:00Z",
				"quality": 8,
				"type": "CORE",
				"local_timezone": "America/Los_Angeles"
			}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:            uuid.New(),
						UserID:        uid,
						StartAt:       req.StartAt.UTC(),
						EndAt:         req.EndAt.UTC(),
						Quality:       req.Quality,
						Type:          req.Type,
						LocalTimezone: *req.LocalTimezone,
					}, false, nil
				},
			},
			wantStatusCode:     http.StatusCreated,
			wantLocalTimezone:  "America/Los_Angeles",
			wantLocalStartHour: 22, // 10 PM SF time
			wantLocalEndHour:   9,  // 9 AM SF time
		},
		{
			name: "Sleep in Poznan before flight",
			// 11 PM Jan 14 Warsaw (UTC+1) = 22:00 UTC Jan 14
			// 7 AM Jan 15 Warsaw = 06:00 UTC Jan 15
			body: `{
				"start_at": "2024-01-14T22:00:00Z",
				"end_at": "2024-01-15T06:00:00Z",
				"quality": 7,
				"type": "CORE",
				"local_timezone": "Europe/Warsaw"
			}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:            uuid.New(),
						UserID:        uid,
						StartAt:       req.StartAt.UTC(),
						EndAt:         req.EndAt.UTC(),
						Quality:       req.Quality,
						Type:          req.Type,
						LocalTimezone: *req.LocalTimezone,
					}, false, nil
				},
			},
			wantStatusCode:     http.StatusCreated,
			wantLocalTimezone:  "Europe/Warsaw",
			wantLocalStartHour: 23, // 11 PM Warsaw time
			wantLocalEndHour:   7,  // 7 AM Warsaw time
		},
		{
			name: "Sleep without explicit timezone uses default",
			body: `{
				"start_at": "2024-01-15T23:00:00Z",
				"end_at": "2024-01-16T07:00:00Z",
				"quality": 8,
				"type": "CORE"
			}`,
			mockService: &MockSleepLogService{
				createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
					return &domain.SleepLog{
						ID:            uuid.New(),
						UserID:        uid,
						StartAt:       req.StartAt.UTC(),
						EndAt:         req.EndAt.UTC(),
						Quality:       req.Quality,
						Type:          req.Type,
						LocalTimezone: "UTC", // Service would set user's default
					}, false, nil
				},
			},
			wantStatusCode:     http.StatusCreated,
			wantLocalTimezone:  "UTC",
			wantLocalStartHour: 23,
			wantLocalEndHour:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSleepLogHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPost, "/v1/users/"+userID.String()+"/sleep-logs", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", userID.String())
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.Create(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Create() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
				return
			}

			if tt.wantStatusCode == http.StatusCreated {
				var response domain.SleepLogResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify timezone in response
				if response.LocalTimezone != tt.wantLocalTimezone {
					t.Errorf("LocalTimezone = %s, want %s", response.LocalTimezone, tt.wantLocalTimezone)
				}

				// Verify local times are converted correctly
				if response.LocalStartAt.Hour() != tt.wantLocalStartHour {
					t.Errorf("LocalStartAt hour = %d, want %d", response.LocalStartAt.Hour(), tt.wantLocalStartHour)
				}
				if response.LocalEndAt.Hour() != tt.wantLocalEndHour {
					t.Errorf("LocalEndAt hour = %d, want %d", response.LocalEndAt.Hour(), tt.wantLocalEndHour)
				}

				// Verify duration is preserved (11 hours for SF scenario)
				duration := response.EndAt.Sub(response.StartAt)
				localDuration := response.LocalEndAt.Sub(response.LocalStartAt)
				if duration != localDuration {
					t.Errorf("Duration mismatch: UTC=%v, Local=%v", duration, localDuration)
				}
			}
		})
	}
}

// TestSleepLogHandler_CreateThenList_TravelScenario tests the full round-trip:
// Create a sleep log, then List to retrieve it, verifying consistency
func TestSleepLogHandler_CreateThenList_TravelScenario(t *testing.T) {
	userID := uuid.New()

	// Shared state to simulate what the service would store
	var storedLog *domain.SleepLog

	mockService := &MockSleepLogService{
		createFunc: func(ctx context.Context, uid uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
			storedLog = &domain.SleepLog{
				ID:            uuid.New(),
				UserID:        uid,
				StartAt:       req.StartAt.UTC(),
				EndAt:         req.EndAt.UTC(),
				Quality:       req.Quality,
				Type:          req.Type,
				LocalTimezone: *req.LocalTimezone,
				CreatedAt:     time.Now(),
			}
			return storedLog, false, nil
		},
		listFunc: func(ctx context.Context, uid uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
			if storedLog == nil {
				return &domain.SleepLogListResponse{
					Data:       []domain.SleepLogResponse{},
					Pagination: domain.PaginationResponse{HasMore: false},
				}, nil
			}
			return &domain.SleepLogListResponse{
				Data:       []domain.SleepLogResponse{storedLog.ToResponse()},
				Pagination: domain.PaginationResponse{HasMore: false},
			}, nil
		},
	}

	handler := NewSleepLogHandler(mockService)

	// Step 1: Create sleep log (11h sleep in SF after Poznan flight)
	createBody := `{
		"start_at": "2024-01-16T06:00:00Z",
		"end_at": "2024-01-16T17:00:00Z",
		"quality": 8,
		"type": "CORE",
		"local_timezone": "America/Los_Angeles"
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/v1/users/"+userID.String()+"/sleep-logs", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("userId", userID.String())
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), chi.RouteCtxKey, rctx))

	handler.Create(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("Create() status = %d, want %d, body: %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	var createResponse domain.SleepLogResponse
	if err := json.NewDecoder(createRec.Body).Decode(&createResponse); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	// Step 2: List sleep logs to retrieve the created log
	listReq := httptest.NewRequest(http.MethodGet, "/v1/users/"+userID.String()+"/sleep-logs", nil)
	listRec := httptest.NewRecorder()

	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("userId", userID.String())
	listReq = listReq.WithContext(context.WithValue(listReq.Context(), chi.RouteCtxKey, rctx2))

	handler.List(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("List() status = %d, want %d, body: %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResponse domain.SleepLogListResponse
	if err := json.NewDecoder(listRec.Body).Decode(&listResponse); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}

	// Verify we got the log back
	if len(listResponse.Data) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(listResponse.Data))
	}

	retrievedLog := listResponse.Data[0]

	// Verify Create and List responses are consistent
	if retrievedLog.ID != createResponse.ID {
		t.Errorf("ID mismatch: create=%v, list=%v", createResponse.ID, retrievedLog.ID)
	}
	if !retrievedLog.StartAt.Equal(createResponse.StartAt) {
		t.Errorf("StartAt mismatch: create=%v, list=%v", createResponse.StartAt, retrievedLog.StartAt)
	}
	if !retrievedLog.EndAt.Equal(createResponse.EndAt) {
		t.Errorf("EndAt mismatch: create=%v, list=%v", createResponse.EndAt, retrievedLog.EndAt)
	}
	if retrievedLog.LocalTimezone != createResponse.LocalTimezone {
		t.Errorf("LocalTimezone mismatch: create=%s, list=%s", createResponse.LocalTimezone, retrievedLog.LocalTimezone)
	}
	if retrievedLog.LocalStartAt.Hour() != createResponse.LocalStartAt.Hour() {
		t.Errorf("LocalStartAt hour mismatch: create=%d, list=%d", createResponse.LocalStartAt.Hour(), retrievedLog.LocalStartAt.Hour())
	}
	if retrievedLog.LocalEndAt.Hour() != createResponse.LocalEndAt.Hour() {
		t.Errorf("LocalEndAt hour mismatch: create=%d, list=%d", createResponse.LocalEndAt.Hour(), retrievedLog.LocalEndAt.Hour())
	}

	// Verify the actual values for SF timezone
	if retrievedLog.LocalStartAt.Hour() != 22 {
		t.Errorf("LocalStartAt hour = %d, want 22 (10 PM SF)", retrievedLog.LocalStartAt.Hour())
	}
	if retrievedLog.LocalEndAt.Hour() != 9 {
		t.Errorf("LocalEndAt hour = %d, want 9 (9 AM SF)", retrievedLog.LocalEndAt.Hour())
	}

	// Verify duration is 11 hours
	duration := retrievedLog.EndAt.Sub(retrievedLog.StartAt)
	if duration != 11*time.Hour {
		t.Errorf("Duration = %v, want 11h", duration)
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
			wantStatusCode: http.StatusUnprocessableEntity,
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
