package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MockUserService is a mock implementation of UserService
type MockUserService struct {
	createFunc  func(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error)
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

func (m *MockUserService) Create(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return &domain.User{ID: uuid.New(), Timezone: req.Timezone}, nil
}

func (m *MockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func TestUserHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockService    *MockUserService
		wantStatusCode int
	}{
		{
			name: "valid request",
			body: `{"timezone": "Europe/Budapest"}`,
			mockService: &MockUserService{
				createFunc: func(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
					return &domain.User{
						ID:       uuid.New(),
						Timezone: req.Timezone,
					}, nil
				},
			},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "invalid JSON",
			body:           `{invalid}`,
			mockService:    &MockUserService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing timezone",
			body:           `{}`,
			mockService:    &MockUserService{},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid timezone",
			body:           `{"timezone": "Invalid/Zone"}`,
			mockService:    &MockUserService{},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.mockService)

			req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("Create() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
			}
		})
	}
}

func TestUserHandler_GetByID(t *testing.T) {
	existingUserID := uuid.New()
	existingUser := &domain.User{
		ID:       existingUserID,
		Timezone: "UTC",
	}

	tests := []struct {
		name           string
		userID         string
		mockService    *MockUserService
		wantStatusCode int
	}{
		{
			name:   "existing user",
			userID: existingUserID.String(),
			mockService: &MockUserService{
				getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
					if id == existingUserID {
						return existingUser, nil
					}
					return nil, domain.ErrNotFound
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:   "non-existing user",
			userID: uuid.New().String(),
			mockService: &MockUserService{
				getByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
					return nil, domain.ErrNotFound
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "invalid UUID",
			userID:         "not-a-uuid",
			mockService:    &MockUserService{},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewUserHandler(tt.mockService)

			// Setup chi router context for URL params
			req := httptest.NewRequest(http.MethodGet, "/v1/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()

			// Add chi URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetByID(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("GetByID() status = %d, want %d, body: %s", rec.Code, tt.wantStatusCode, rec.Body.String())
			}

			// Verify response body for successful request
			if tt.wantStatusCode == http.StatusOK {
				var response domain.UserResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
			}
		})
	}
}
