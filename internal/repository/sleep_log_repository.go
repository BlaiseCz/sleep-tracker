package repository

import (
	"context"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/pkg/pagination"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SleepLogRepository interface {
	Create(ctx context.Context, log *domain.SleepLog) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SleepLog, error)
	List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) ([]domain.SleepLog, error)
	HasOverlap(ctx context.Context, userID uuid.UUID, startAt, endAt time.Time, sleepType domain.SleepType) (bool, error)
	GetByClientRequestID(ctx context.Context, userID uuid.UUID, clientRequestID string) (*domain.SleepLog, error)
}

type sleepLogRepository struct {
	db *gorm.DB
}

func NewSleepLogRepository(db *gorm.DB) SleepLogRepository {
	return &sleepLogRepository{db: db}
}

func (r *sleepLogRepository) Create(ctx context.Context, log *domain.SleepLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *sleepLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SleepLog, error) {
	var log domain.SleepLog
	err := r.db.WithContext(ctx).First(&log, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &log, nil
}

func (r *sleepLogRepository) List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) ([]domain.SleepLog, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("start_at DESC")

	// Apply time filters
	if filter.From != nil {
		query = query.Where("start_at >= ?", filter.From)
	}
	if filter.To != nil {
		query = query.Where("start_at <= ?", filter.To)
	}

	// Apply cursor pagination
	if filter.Cursor != "" {
		cursor, err := pagination.DecodeCursor(filter.Cursor)
		if err == nil && cursor != nil {
			// For DESC order: get records with start_at < cursor.StartAt
			// or same start_at but id < cursor.ID
			query = query.Where(
				"(start_at < ?) OR (start_at = ? AND id < ?)",
				cursor.StartAt, cursor.StartAt, cursor.ID,
			)
		}
	}

	// Fetch one extra to determine if there are more results
	limit := pagination.NormalizeLimit(filter.Limit)
	query = query.Limit(limit + 1)

	var logs []domain.SleepLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// HasOverlap checks if there's an overlapping sleep period
// For CORE: checks overlap with any CORE sleep
// For NAP: checks overlap with CORE sleep only
func (r *sleepLogRepository) HasOverlap(ctx context.Context, userID uuid.UUID, startAt, endAt time.Time, sleepType domain.SleepType) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.SleepLog{}).
		Where("user_id = ?", userID).
		Where("start_at < ?", endAt).
		Where("end_at > ?", startAt)

	// CORE can't overlap with CORE
	// NAP can't overlap with CORE (but can overlap with NAP)
	if sleepType == domain.SleepTypeCore {
		query = query.Where("type = ?", domain.SleepTypeCore)
	} else {
		query = query.Where("type = ?", domain.SleepTypeCore)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *sleepLogRepository) GetByClientRequestID(ctx context.Context, userID uuid.UUID, clientRequestID string) (*domain.SleepLog, error) {
	var log domain.SleepLog
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND client_request_id = ?", userID, clientRequestID).
		First(&log).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Not found is not an error for idempotency check
		}
		return nil, err
	}
	return &log, nil
}
