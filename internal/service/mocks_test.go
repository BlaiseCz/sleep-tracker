package service

import (
	"context"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// MockSleepLogRepository is a mock implementation of SleepLogRepository
type MockSleepLogRepository struct {
	logs            map[uuid.UUID]*domain.SleepLog
	clientRequestID map[string]*domain.SleepLog
	listResult      []domain.SleepLog
	err             error
}

func NewMockSleepLogRepository() *MockSleepLogRepository {
	return &MockSleepLogRepository{
		logs:            make(map[uuid.UUID]*domain.SleepLog),
		clientRequestID: make(map[string]*domain.SleepLog),
	}
}

func (m *MockSleepLogRepository) Create(ctx context.Context, log *domain.SleepLog) error {
	if m.err != nil {
		return m.err
	}
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now()
	m.logs[log.ID] = log
	if log.ClientRequestID != nil {
		key := log.UserID.String() + ":" + *log.ClientRequestID
		m.clientRequestID[key] = log
	}
	return nil
}

func (m *MockSleepLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SleepLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	log, ok := m.logs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return log, nil
}

func (m *MockSleepLogRepository) Update(ctx context.Context, log *domain.SleepLog) error {
	if m.err != nil {
		return m.err
	}
	m.logs[log.ID] = log
	return nil
}

func (m *MockSleepLogRepository) List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) ([]domain.SleepLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.listResult != nil {
		result := make([]domain.SleepLog, len(m.listResult))
		copy(result, m.listResult)
		return result, nil
	}
	var result []domain.SleepLog
	for _, log := range m.logs {
		if log.UserID == userID {
			result = append(result, *log)
		}
	}
	return result, nil
}

func (m *MockSleepLogRepository) HasOverlap(ctx context.Context, userID uuid.UUID, startAt, endAt time.Time, sleepType domain.SleepType) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	for _, log := range m.logs {
		if log.UserID != userID {
			continue
		}
		// Check overlap: new period overlaps if start < existing.end AND end > existing.start
		if startAt.Before(log.EndAt) && endAt.After(log.StartAt) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockSleepLogRepository) HasOverlapExcluding(ctx context.Context, userID uuid.UUID, excludeID uuid.UUID, startAt, endAt time.Time, sleepType domain.SleepType) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	for _, log := range m.logs {
		if log.UserID != userID || log.ID == excludeID {
			continue
		}
		// Check overlap: new period overlaps if start < existing.end AND end > existing.start
		if startAt.Before(log.EndAt) && endAt.After(log.StartAt) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockSleepLogRepository) GetByClientRequestID(ctx context.Context, userID uuid.UUID, clientRequestID string) (*domain.SleepLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := userID.String() + ":" + clientRequestID
	log, ok := m.clientRequestID[key]
	if !ok {
		return nil, nil
	}
	return log, nil
}

func (m *MockSleepLogRepository) ListByEndRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]domain.SleepLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []domain.SleepLog
	for _, log := range m.logs {
		if log.UserID == userID && !log.EndAt.Before(from) && !log.EndAt.After(to) {
			result = append(result, *log)
		}
	}
	return result, nil
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	users map[uuid.UUID]*domain.User
	err   error
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

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func sleepTypePtr(t domain.SleepType) *domain.SleepType {
	return &t
}

func timePtr(t time.Time) *time.Time {
	return &t
}
