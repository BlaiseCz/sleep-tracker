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

// CreateUserRequest is the request body for creating a user
type CreateUserRequest struct {
	Timezone string `json:"timezone" validate:"required,timezone"`
}

// UserResponse is the response body for user endpoints
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Timezone  string    `json:"timezone"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Timezone:  u.Timezone,
		CreatedAt: u.CreatedAt,
	}
}
