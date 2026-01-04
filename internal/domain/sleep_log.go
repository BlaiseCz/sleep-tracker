package domain

import (
	"time"

	"github.com/google/uuid"
)

// SleepType represents the category of sleep session.
// @Description Type of sleep: CORE for main night sleep, NAP for daytime naps.
type SleepType string

const (
	// SleepTypeCore is the primary overnight sleep session
	SleepTypeCore SleepType = "CORE"
	// SleepTypeNap is a short daytime sleep session
	SleepTypeNap SleepType = "NAP"
)

type SleepLog struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index:idx_sleep_logs_user_start" json:"user_id"`
	StartAt         time.Time `gorm:"not null;index:idx_sleep_logs_user_start,sort:desc" json:"start_at"`
	EndAt           time.Time `gorm:"not null" json:"end_at"`
	Quality         int       `gorm:"type:smallint;not null" json:"quality"`
	Type            SleepType `gorm:"type:varchar(10);not null" json:"type"`
	LocalTimezone   string    `gorm:"type:varchar(64);not null;default:'UTC'" json:"local_timezone"`
	ClientRequestID *string   `gorm:"type:varchar(255);uniqueIndex:idx_user_client_request,where:client_request_id IS NOT NULL" json:"client_request_id,omitempty"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Associations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (SleepLog) TableName() string {
	return "sleep_logs"
}

// CreateSleepLogRequest is the request body for creating a sleep log.
// @Description Request payload for recording a sleep session.
type CreateSleepLogRequest struct {
	// Sleep start time in RFC3339 format (UTC recommended)
	StartAt time.Time `json:"start_at" validate:"required" example:"2024-01-15T23:00:00Z"`
	// Sleep end time in RFC3339 format (must be after start_at)
	EndAt time.Time `json:"end_at" validate:"required,gtfield=StartAt" example:"2024-01-16T07:00:00Z"`
	// Sleep quality rating from 1 (poor) to 10 (excellent)
	Quality int `json:"quality" validate:"required,min=1,max=10" example:"7" minimum:"1" maximum:"10"`
	// Sleep type: CORE (main sleep) or NAP (daytime nap)
	Type SleepType `json:"type" validate:"required,oneof=CORE NAP" example:"CORE" enums:"CORE,NAP"`
	// Optional client-generated ID for idempotent requests (max 255 chars)
	ClientRequestID *string `json:"client_request_id,omitempty" validate:"omitempty,max=255" example:"client-uuid-12345"`
	// Optional IANA timezone for local time display (defaults to user's timezone)
	LocalTimezone *string `json:"local_timezone,omitempty" validate:"omitempty,timezone" example:"Europe/Prague"`
}

// SleepLogResponse is the response body for sleep log endpoints.
// @Description Sleep session record with UTC and local times.
type SleepLogResponse struct {
	// Unique sleep log identifier
	ID uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Owner user ID
	UserID uuid.UUID `json:"user_id" example:"660e8400-e29b-41d4-a716-446655440001"`
	// Sleep start time (UTC)
	StartAt time.Time `json:"start_at" example:"2024-01-15T23:00:00Z"`
	// Sleep end time (UTC)
	EndAt time.Time `json:"end_at" example:"2024-01-16T07:00:00Z"`
	// Sleep quality (1-10)
	Quality int `json:"quality" example:"7"`
	// Sleep type
	Type SleepType `json:"type" example:"CORE"`
	// Client-provided request ID (if any)
	ClientRequestID *string `json:"client_request_id,omitempty" example:"client-uuid-12345"`
	// Record creation timestamp
	CreatedAt time.Time `json:"created_at" example:"2024-01-16T07:05:00Z"`
	// Timezone used for local times
	LocalTimezone string `json:"local_timezone" example:"Europe/Prague"`
	// Sleep start in local timezone
	LocalStartAt time.Time `json:"local_start_at" example:"2024-01-16T00:00:00+01:00"`
	// Sleep end in local timezone
	LocalEndAt time.Time `json:"local_end_at" example:"2024-01-16T08:00:00+01:00"`
}

func (s *SleepLog) ToResponse() SleepLogResponse {
	loc := time.UTC
	if s.LocalTimezone != "" {
		if l, err := time.LoadLocation(s.LocalTimezone); err == nil {
			loc = l
		}
	}

	return SleepLogResponse{
		ID:              s.ID,
		UserID:          s.UserID,
		StartAt:         s.StartAt,
		EndAt:           s.EndAt,
		Quality:         s.Quality,
		Type:            s.Type,
		ClientRequestID: s.ClientRequestID,
		CreatedAt:       s.CreatedAt,
		LocalTimezone:   s.LocalTimezone,
		LocalStartAt:    s.StartAt.In(loc),
		LocalEndAt:      s.EndAt.In(loc),
	}
}

// SleepLogListResponse is the response body for listing sleep logs.
// @Description Paginated list of sleep logs.
type SleepLogListResponse struct {
	// Array of sleep log records
	Data []SleepLogResponse `json:"data"`
	// Pagination metadata
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse contains pagination metadata.
// @Description Cursor-based pagination info.
type PaginationResponse struct {
	// Cursor for fetching the next page (empty if no more pages)
	NextCursor string `json:"next_cursor,omitempty" example:"eyJpZCI6IjU1MGU4NDAwLWUyOWItNDFkNC1hNzE2LTQ0NjY1NTQ0MDAwMCJ9"`
	// True if more results are available
	HasMore bool `json:"has_more" example:"true"`
}

// SleepLogFilter contains filter parameters for listing sleep logs
type SleepLogFilter struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor string
}
