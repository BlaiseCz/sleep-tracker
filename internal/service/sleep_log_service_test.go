package service

import (
	"context"
	"testing"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// MockSleepLogRepository is a mock implementation of SleepLogRepository
type MockSleepLogRepository struct {
	logs            map[uuid.UUID]*domain.SleepLog
	clientRequestID map[string]*domain.SleepLog
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

func (m *MockSleepLogRepository) List(ctx context.Context, userID uuid.UUID, filter domain.SleepLogFilter) ([]domain.SleepLog, error) {
	if m.err != nil {
		return nil, m.err
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
			// For CORE: overlaps with CORE
			// For NAP: overlaps with CORE only
			if log.Type == domain.SleepTypeCore {
				return true, nil
			}
			if sleepType == domain.SleepTypeCore && log.Type == domain.SleepTypeCore {
				return true, nil
			}
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

func TestSleepLogService_Create(t *testing.T) {
	userID := uuid.New()
	
	// Setup user repo with existing user
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	tests := []struct {
		name        string
		req         *domain.CreateSleepLogRequest
		setupLogs   func(*MockSleepLogRepository)
		wantErr     error
		wantExist   bool
	}{
		{
			name: "valid CORE sleep",
			req: &domain.CreateSleepLogRequest{
				StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality: 8,
				Type:    domain.SleepTypeCore,
			},
			wantErr: nil,
		},
		{
			name: "valid NAP",
			req: &domain.CreateSleepLogRequest{
				StartAt: time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
				EndAt:   time.Date(2024, 1, 16, 15, 0, 0, 0, time.UTC),
				Quality: 6,
				Type:    domain.SleepTypeNap,
			},
			wantErr: nil,
		},
		{
			name: "overlapping CORE sleep",
			req: &domain.CreateSleepLogRequest{
				StartAt: time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC),
				EndAt:   time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
				Quality: 7,
				Type:    domain.SleepTypeCore,
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				repo.logs[uuid.New()] = &domain.SleepLog{
					ID:      uuid.New(),
					UserID:  userID,
					StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
					Type:    domain.SleepTypeCore,
				}
			},
			wantErr: domain.ErrOverlappingSleep,
		},
		{
			name: "idempotent request returns existing",
			req: &domain.CreateSleepLogRequest{
				StartAt:         time.Date(2024, 1, 17, 23, 0, 0, 0, time.UTC),
				EndAt:           time.Date(2024, 1, 18, 7, 0, 0, 0, time.UTC),
				Quality:         8,
				Type:            domain.SleepTypeCore,
				ClientRequestID: strPtr("req-123"),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				existingLog := &domain.SleepLog{
					ID:              uuid.New(),
					UserID:          userID,
					StartAt:         time.Date(2024, 1, 17, 23, 0, 0, 0, time.UTC),
					EndAt:           time.Date(2024, 1, 18, 7, 0, 0, 0, time.UTC),
					Quality:         8,
					Type:            domain.SleepTypeCore,
					ClientRequestID: strPtr("req-123"),
				}
				repo.logs[existingLog.ID] = existingLog
				repo.clientRequestID[userID.String()+":req-123"] = existingLog
			},
			wantErr:   nil,
			wantExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := NewMockSleepLogRepository()
			if tt.setupLogs != nil {
				tt.setupLogs(logRepo)
			}

			svc := NewSleepLogService(logRepo, userRepo)
			log, isExisting, err := svc.Create(context.Background(), userID, tt.req)

			if err != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil {
				if log == nil {
					t.Error("Create() returned nil log")
					return
				}
				if isExisting != tt.wantExist {
					t.Errorf("Create() isExisting = %v, want %v", isExisting, tt.wantExist)
				}
			}
		})
	}
}

func TestSleepLogService_Create_UserNotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	logRepo := NewMockSleepLogRepository()
	svc := NewSleepLogService(logRepo, userRepo)

	req := &domain.CreateSleepLogRequest{
		StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality: 8,
		Type:    domain.SleepTypeCore,
	}

	_, _, err := svc.Create(context.Background(), uuid.New(), req)
	if err != domain.ErrNotFound {
		t.Errorf("Create() error = %v, want %v", err, domain.ErrNotFound)
	}
}

func strPtr(s string) *string {
	return &s
}
