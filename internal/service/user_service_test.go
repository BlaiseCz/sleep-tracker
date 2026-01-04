package service

import (
	"context"
	"testing"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	users  map[uuid.UUID]*domain.User
	err    error
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[uuid.UUID]*domain.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.err != nil {
		return m.err
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *MockUserRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	_, ok := m.users[id]
	return ok, nil
}

func (m *MockUserRepository) SetError(err error) {
	m.err = err
}

func TestUserService_Create(t *testing.T) {
	tests := []struct {
		name     string
		req      *domain.CreateUserRequest
		wantErr  bool
	}{
		{
			name: "valid timezone",
			req: &domain.CreateUserRequest{
				Timezone: "Europe/Budapest",
			},
			wantErr: false,
		},
		{
			name: "UTC timezone",
			req: &domain.CreateUserRequest{
				Timezone: "UTC",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepository()
			svc := NewUserService(repo)

			user, err := svc.Create(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user == nil {
					t.Error("Create() returned nil user")
					return
				}
				if user.Timezone != tt.req.Timezone {
					t.Errorf("Create() timezone = %v, want %v", user.Timezone, tt.req.Timezone)
				}
				if user.ID == uuid.Nil {
					t.Error("Create() user ID should not be nil")
				}
			}
		})
	}
}

func TestUserService_GetByID(t *testing.T) {
	repo := NewMockUserRepository()
	svc := NewUserService(repo)

	// Create a user first
	req := &domain.CreateUserRequest{Timezone: "America/New_York"}
	created, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name:    "existing user",
			id:      created.ID,
			wantErr: nil,
		},
		{
			name:    "non-existing user",
			id:      uuid.New(),
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetByID(context.Background(), tt.id)
			if err != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil && user == nil {
				t.Error("GetByID() returned nil user for existing ID")
			}
		})
	}
}
