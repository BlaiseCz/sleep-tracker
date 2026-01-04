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

// TestSleepLogService_Create_TravelScenario tests the Poznan → San Francisco travel scenario
// where a user sleeps 11 hours after a long flight
func TestSleepLogService_Create_TravelScenario(t *testing.T) {
	userID := uuid.New()

	// User's home timezone is Europe/Warsaw (Poznan)
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "Europe/Warsaw"}

	tests := []struct {
		name              string
		req               *domain.CreateSleepLogRequest
		wantLocalTimezone string
		wantDuration      time.Duration
		wantErr           error
	}{
		{
			name: "11h sleep in San Francisco after flight from Poznan",
			// User flew from Poznan to SF, fell asleep at 10 PM SF time
			// Woke up at 9 AM SF time = 11 hours of sleep
			// SF is UTC-8 in January, so:
			// 10 PM Jan 15 SF = 06:00 UTC Jan 16
			// 9 AM Jan 16 SF = 17:00 UTC Jan 16
			req: &domain.CreateSleepLogRequest{
				StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          domain.SleepTypeCore,
				LocalTimezone: strPtr("America/Los_Angeles"),
			},
			wantLocalTimezone: "America/Los_Angeles",
			wantDuration:      11 * time.Hour,
			wantErr:           nil,
		},
		{
			name: "Last night in Poznan before flight",
			// User slept in Poznan before the flight
			// 11 PM Poznan = 22:00 UTC (UTC+1 in winter)
			// 7 AM Poznan = 06:00 UTC
			req: &domain.CreateSleepLogRequest{
				StartAt:       time.Date(2024, 1, 14, 22, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),
				Quality:       7,
				Type:          domain.SleepTypeCore,
				LocalTimezone: strPtr("Europe/Warsaw"),
			},
			wantLocalTimezone: "Europe/Warsaw",
			wantDuration:      8 * time.Hour,
			wantErr:           nil,
		},
		{
			name: "Sleep uses user default timezone when not specified",
			req: &domain.CreateSleepLogRequest{
				StartAt: time.Date(2024, 1, 17, 22, 0, 0, 0, time.UTC),
				EndAt:   time.Date(2024, 1, 18, 6, 0, 0, 0, time.UTC),
				Quality: 7,
				Type:    domain.SleepTypeCore,
				// No LocalTimezone specified - should use user's home timezone
			},
			wantLocalTimezone: "Europe/Warsaw", // User's default
			wantDuration:      8 * time.Hour,
			wantErr:           nil,
		},
		{
			name: "Nap during layover",
			// Short nap during a layover
			req: &domain.CreateSleepLogRequest{
				StartAt:       time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC),
				Quality:       5,
				Type:          domain.SleepTypeNap,
				LocalTimezone: strPtr("Europe/London"), // Layover in London
			},
			wantLocalTimezone: "Europe/London",
			wantDuration:      2 * time.Hour,
			wantErr:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := NewMockSleepLogRepository()
			svc := NewSleepLogService(logRepo, userRepo)

			log, isExisting, err := svc.Create(context.Background(), userID, tt.req)

			if err != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr != nil {
				return
			}

			if log == nil {
				t.Fatal("Create() returned nil log")
			}

			if isExisting {
				t.Error("Create() isExisting = true, want false for new log")
			}

			// Verify timezone is set correctly
			if log.LocalTimezone != tt.wantLocalTimezone {
				t.Errorf("LocalTimezone = %s, want %s", log.LocalTimezone, tt.wantLocalTimezone)
			}

			// Verify duration
			duration := log.EndAt.Sub(log.StartAt)
			if duration != tt.wantDuration {
				t.Errorf("Duration = %v, want %v", duration, tt.wantDuration)
			}

			// Verify times are stored in UTC
			if log.StartAt.Location() != time.UTC {
				t.Error("StartAt should be in UTC")
			}
			if log.EndAt.Location() != time.UTC {
				t.Error("EndAt should be in UTC")
			}
		})
	}
}

// TestSleepLogService_Create_TimezoneEdgeCases tests edge cases with timezone handling
func TestSleepLogService_Create_TimezoneEdgeCases(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name              string
		userTimezone      string
		reqLocalTimezone  *string
		wantLocalTimezone string
	}{
		{
			name:              "User has no timezone, request has timezone",
			userTimezone:      "",
			reqLocalTimezone:  strPtr("America/New_York"),
			wantLocalTimezone: "America/New_York",
		},
		{
			name:              "User has no timezone, request has no timezone",
			userTimezone:      "",
			reqLocalTimezone:  nil,
			wantLocalTimezone: "UTC",
		},
		{
			name:              "User has timezone, request overrides",
			userTimezone:      "Europe/Warsaw",
			reqLocalTimezone:  strPtr("Asia/Tokyo"),
			wantLocalTimezone: "Asia/Tokyo",
		},
		{
			name:              "User has timezone, request has empty string",
			userTimezone:      "Europe/Warsaw",
			reqLocalTimezone:  strPtr(""),
			wantLocalTimezone: "Europe/Warsaw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := NewMockUserRepository()
			userRepo.users[userID] = &domain.User{ID: userID, Timezone: tt.userTimezone}
			logRepo := NewMockSleepLogRepository()
			svc := NewSleepLogService(logRepo, userRepo)

			req := &domain.CreateSleepLogRequest{
				StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          domain.SleepTypeCore,
				LocalTimezone: tt.reqLocalTimezone,
			}

			log, _, err := svc.Create(context.Background(), userID, req)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if log.LocalTimezone != tt.wantLocalTimezone {
				t.Errorf("LocalTimezone = %s, want %s", log.LocalTimezone, tt.wantLocalTimezone)
			}
		})
	}
}

// TestSleepLogService_Create_LongSleepDurations tests various sleep durations
func TestSleepLogService_Create_LongSleepDurations(t *testing.T) {
	userID := uuid.New()
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	tests := []struct {
		name         string
		startAt      time.Time
		endAt        time.Time
		wantDuration time.Duration
	}{
		{
			name:         "11 hours (jet lag recovery)",
			startAt:      time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
			endAt:        time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC),
			wantDuration: 11 * time.Hour,
		},
		{
			name:         "12 hours (extended recovery)",
			startAt:      time.Date(2024, 1, 16, 20, 0, 0, 0, time.UTC),
			endAt:        time.Date(2024, 1, 17, 8, 0, 0, 0, time.UTC),
			wantDuration: 12 * time.Hour,
		},
		{
			name:         "Standard 8 hours",
			startAt:      time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
			endAt:        time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
			wantDuration: 8 * time.Hour,
		},
		{
			name:         "Short 4 hours",
			startAt:      time.Date(2024, 1, 16, 2, 0, 0, 0, time.UTC),
			endAt:        time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
			wantDuration: 4 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := NewMockSleepLogRepository()
			svc := NewSleepLogService(logRepo, userRepo)

			req := &domain.CreateSleepLogRequest{
				StartAt: tt.startAt,
				EndAt:   tt.endAt,
				Quality: 8,
				Type:    domain.SleepTypeCore,
			}

			log, _, err := svc.Create(context.Background(), userID, req)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			duration := log.EndAt.Sub(log.StartAt)
			if duration != tt.wantDuration {
				t.Errorf("Duration = %v, want %v", duration, tt.wantDuration)
			}
		})
	}
}
