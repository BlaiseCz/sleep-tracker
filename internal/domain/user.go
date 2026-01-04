package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Timezone  string    `gorm:"type:varchar(64);not null;default:'UTC'" json:"timezone"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (User) TableName() string {
	return "users"
}

// CreateUserRequest is the request body for creating a user.
// @Description Request payload for creating a new user account.
type CreateUserRequest struct {
	// IANA timezone identifier (e.g., "America/New_York", "Europe/London", "UTC").
	// See: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	Timezone string `json:"timezone" validate:"required,timezone" example:"Europe/Prague"`
}

// UserResponse is the response body for user endpoints.
// @Description User account details.
type UserResponse struct {
	// Unique user identifier
	ID uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// User's preferred IANA timezone
	Timezone string `json:"timezone" example:"Europe/Prague"`
	// Account creation timestamp (RFC3339)
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Timezone:  u.Timezone,
		CreatedAt: u.CreatedAt,
	}
}
