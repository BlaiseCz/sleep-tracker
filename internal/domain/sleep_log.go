package domain

import (
	"time"

	"github.com/google/uuid"
)

type SleepType string

const (
	SleepTypeCore SleepType = "CORE"
	SleepTypeNap  SleepType = "NAP"
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

// CreateSleepLogRequest is the request body for creating a sleep log
type CreateSleepLogRequest struct {
	StartAt         time.Time `json:"start_at" validate:"required"`
	EndAt           time.Time `json:"end_at" validate:"required,gtfield=StartAt"`
	Quality         int       `json:"quality" validate:"required,min=1,max=10"`
	Type            SleepType `json:"type" validate:"required,oneof=CORE NAP"`
	ClientRequestID *string   `json:"client_request_id,omitempty" validate:"omitempty,max=255"`
	LocalTimezone   *string   `json:"local_timezone,omitempty" validate:"omitempty,timezone"`
}

// SleepLogResponse is the response body for sleep log endpoints
type SleepLogResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	StartAt         time.Time  `json:"start_at"`
	EndAt           time.Time  `json:"end_at"`
	Quality         int        `json:"quality"`
	Type            SleepType  `json:"type"`
	ClientRequestID *string    `json:"client_request_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	LocalTimezone   string     `json:"local_timezone"`
	LocalStartAt    time.Time  `json:"local_start_at"`
	LocalEndAt      time.Time  `json:"local_end_at"`
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

// SleepLogListResponse is the response body for listing sleep logs
type SleepLogListResponse struct {
	Data       []SleepLogResponse `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse contains pagination metadata
type PaginationResponse struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// SleepLogFilter contains filter parameters for listing sleep logs
type SleepLogFilter struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor string
}
