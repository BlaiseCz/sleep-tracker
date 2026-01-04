package service

import (
	"context"
	"testing"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// Mocks are defined in mocks_test.go

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
