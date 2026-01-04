package handler

import (
	"context"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// MockSleepLogService is a mock implementation of SleepLogService
type MockSleepLogService struct {
	createFunc func(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error)
	updateFunc func(ctx context.Context, userID uuid.UUID, logID uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error)
	listFunc   func(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error)
}

func (m *MockSleepLogService) Create(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, req)
	}
	return &domain.SleepLog{
		ID:            uuid.New(),
		UserID:        userID,
		StartAt:       req.StartAt,
		EndAt:         req.EndAt,
		Quality:       req.Quality,
		Type:          req.Type,
		LocalTimezone: "UTC",
		CreatedAt:     time.Now(),
	}, false, nil
}

func (m *MockSleepLogService) Update(ctx context.Context, userID uuid.UUID, logID uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, userID, logID, req)
	}
	return &domain.SleepLog{
		ID:            logID,
		UserID:        userID,
		StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality:       8,
		Type:          domain.SleepTypeCore,
		LocalTimezone: "UTC",
		CreatedAt:     time.Now(),
	}, nil
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
