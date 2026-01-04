package service

import (
	"context"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/repository"
	"github.com/blaisecz/sleep-tracker/pkg/pagination"
	"github.com/google/uuid"
)

type SleepLogService interface {
	Create(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error)
	Update(ctx context.Context, userID uuid.UUID, logID uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error)
	List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error)
}

type sleepLogService struct {
	repo     repository.SleepLogRepository
	userRepo repository.UserRepository
}

func NewSleepLogService(repo repository.SleepLogRepository, userRepo repository.UserRepository) SleepLogService {
	return &sleepLogService{
		repo:     repo,
		userRepo: userRepo,
	}
}

// Create creates a new sleep log
// Returns (log, isExisting, error) - isExisting is true if returning existing log due to idempotency
func (s *sleepLogService) Create(ctx context.Context, userID uuid.UUID, req *domain.CreateSleepLogRequest) (*domain.SleepLog, bool, error) {
	// Load user to confirm existence and get their home timezone
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, false, domain.ErrNotFound
		}
		return nil, false, err
	}

	// Determine local timezone for this log
	localTZ := user.Timezone
	if req.LocalTimezone != nil && *req.LocalTimezone != "" {
		localTZ = *req.LocalTimezone
	}
	if localTZ == "" {
		localTZ = "UTC"
	}

	// Normalize timestamps to UTC for storage and overlap checks
	startUTC := req.StartAt.UTC()
	endUTC := req.EndAt.UTC()

	// Check for idempotency (duplicate client_request_id)
	if req.ClientRequestID != nil && *req.ClientRequestID != "" {
		existing, err := s.repo.GetByClientRequestID(ctx, userID, *req.ClientRequestID)
		if err != nil {
			return nil, false, err
		}
		if existing != nil {
			return existing, true, nil // Return existing log
		}
	}

	// Check for overlapping sleep periods
	hasOverlap, err := s.repo.HasOverlap(ctx, userID, startUTC, endUTC, req.Type)
	if err != nil {
		return nil, false, err
	}
	if hasOverlap {
		return nil, false, domain.ErrOverlappingSleep
	}

	// Create the sleep log
	log := &domain.SleepLog{
		UserID:          userID,
		StartAt:         startUTC,
		EndAt:           endUTC,
		Quality:         req.Quality,
		Type:            req.Type,
		LocalTimezone:   localTZ,
		ClientRequestID: req.ClientRequestID,
	}

	if err := s.repo.Create(ctx, log); err != nil {
		return nil, false, err
	}

	return log, false, nil
}

// Update updates an existing sleep log
func (s *sleepLogService) Update(ctx context.Context, userID uuid.UUID, logID uuid.UUID, req *domain.UpdateSleepLogRequest) (*domain.SleepLog, error) {
	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Get existing log
	log, err := s.repo.GetByID(ctx, logID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if log.UserID != userID {
		return nil, domain.ErrNotFound
	}

	// Apply updates
	if req.StartAt != nil {
		log.StartAt = req.StartAt.UTC()
	}
	if req.EndAt != nil {
		log.EndAt = req.EndAt.UTC()
	}
	if req.Quality != nil {
		log.Quality = *req.Quality
	}
	if req.Type != nil {
		log.Type = *req.Type
	}
	if req.LocalTimezone != nil && *req.LocalTimezone != "" {
		log.LocalTimezone = *req.LocalTimezone
	}

	// Validate end > start after applying updates
	if !log.EndAt.After(log.StartAt) {
		return nil, domain.ErrInvalidInput
	}

	// Check for overlapping sleep periods (excluding this log)
	hasOverlap, err := s.repo.HasOverlapExcluding(ctx, userID, logID, log.StartAt, log.EndAt, log.Type)
	if err != nil {
		return nil, err
	}
	if hasOverlap {
		return nil, domain.ErrOverlappingSleep
	}

	// Save updates
	if err := s.repo.Update(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

func (s *sleepLogService) List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) (*domain.SleepLogListResponse, error) {
	// Check if user exists
	exists, err := s.userRepo.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}

	logs, err := s.repo.List(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	limit := pagination.NormalizeLimit(filter.Limit)
	hasMore := len(logs) > limit

	// Trim to actual limit
	if hasMore {
		logs = logs[:limit]
	}

	// Build response
	response := &domain.SleepLogListResponse{
		Data: make([]domain.SleepLogResponse, len(logs)),
		Pagination: domain.PaginationResponse{
			HasMore: hasMore,
		},
	}

	for i, log := range logs {
		response.Data[i] = log.ToResponse()
	}

	// Set next cursor if there are more results
	if hasMore && len(logs) > 0 {
		lastLog := logs[len(logs)-1]
		cursor := &pagination.Cursor{
			ID:      lastLog.ID,
			StartAt: lastLog.StartAt,
		}
		response.Pagination.NextCursor = cursor.Encode()
	}

	return response, nil
}
