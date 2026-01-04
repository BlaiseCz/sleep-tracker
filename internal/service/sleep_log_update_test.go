package service

import (
	"context"
	"testing"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

func TestSleepLogService_Update(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	// Setup user repo with existing user
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	baseLog := &domain.SleepLog{
		ID:            logID,
		UserID:        userID,
		StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality:       7,
		Type:          domain.SleepTypeCore,
		LocalTimezone: "UTC",
	}

	tests := []struct {
		name      string
		req       *domain.UpdateSleepLogRequest
		setupLogs func(*MockSleepLogRepository)
		wantErr   error
		validate  func(*testing.T, *domain.SleepLog)
	}{
		{
			name: "update quality only",
			req: &domain.UpdateSleepLogRequest{
				Quality: intPtr(9),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: nil,
			validate: func(t *testing.T, log *domain.SleepLog) {
				if log.Quality != 9 {
					t.Errorf("Quality = %d, want 9", log.Quality)
				}
				// Other fields should remain unchanged
				if log.Type != domain.SleepTypeCore {
					t.Errorf("Type changed unexpectedly to %s", log.Type)
				}
			},
		},
		{
			name: "update type from CORE to NAP",
			req: &domain.UpdateSleepLogRequest{
				Type: sleepTypePtr(domain.SleepTypeNap),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: nil,
			validate: func(t *testing.T, log *domain.SleepLog) {
				if log.Type != domain.SleepTypeNap {
					t.Errorf("Type = %s, want NAP", log.Type)
				}
			},
		},
		{
			name: "update start and end times",
			req: &domain.UpdateSleepLogRequest{
				StartAt: timePtr(time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC)),
				EndAt:   timePtr(time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC)),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: nil,
			validate: func(t *testing.T, log *domain.SleepLog) {
				expectedStart := time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC)
				expectedEnd := time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC)
				if !log.StartAt.Equal(expectedStart) {
					t.Errorf("StartAt = %v, want %v", log.StartAt, expectedStart)
				}
				if !log.EndAt.Equal(expectedEnd) {
					t.Errorf("EndAt = %v, want %v", log.EndAt, expectedEnd)
				}
			},
		},
		{
			name: "update timezone",
			req: &domain.UpdateSleepLogRequest{
				LocalTimezone: strPtr("America/New_York"),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: nil,
			validate: func(t *testing.T, log *domain.SleepLog) {
				if log.LocalTimezone != "America/New_York" {
					t.Errorf("LocalTimezone = %s, want America/New_York", log.LocalTimezone)
				}
			},
		},
		{
			name: "update multiple fields",
			req: &domain.UpdateSleepLogRequest{
				Quality:       intPtr(10),
				Type:          sleepTypePtr(domain.SleepTypeNap),
				LocalTimezone: strPtr("Europe/Warsaw"),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: nil,
			validate: func(t *testing.T, log *domain.SleepLog) {
				if log.Quality != 10 {
					t.Errorf("Quality = %d, want 10", log.Quality)
				}
				if log.Type != domain.SleepTypeNap {
					t.Errorf("Type = %s, want NAP", log.Type)
				}
				if log.LocalTimezone != "Europe/Warsaw" {
					t.Errorf("LocalTimezone = %s, want Europe/Warsaw", log.LocalTimezone)
				}
			},
		},
		{
			name: "log not found",
			req: &domain.UpdateSleepLogRequest{
				Quality: intPtr(9),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				// Don't add any logs
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "end time before start time",
			req: &domain.UpdateSleepLogRequest{
				EndAt: timePtr(time.Date(2024, 1, 15, 20, 0, 0, 0, time.UTC)), // Before existing start
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				logCopy := *baseLog
				repo.logs[logID] = &logCopy
			},
			wantErr: domain.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := NewMockSleepLogRepository()
			if tt.setupLogs != nil {
				tt.setupLogs(logRepo)
			}

			svc := NewSleepLogService(logRepo, userRepo)
			log, err := svc.Update(context.Background(), userID, logID, tt.req)

			if err != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil && tt.validate != nil {
				tt.validate(t, log)
			}
		})
	}
}

func TestSleepLogService_Update_UserNotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	logRepo := NewMockSleepLogRepository()
	svc := NewSleepLogService(logRepo, userRepo)

	req := &domain.UpdateSleepLogRequest{
		Quality: intPtr(9),
	}

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), req)
	if err != domain.ErrNotFound {
		t.Errorf("Update() error = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestSleepLogService_Update_WrongOwner(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	logID := uuid.New()

	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}
	userRepo.users[otherUserID] = &domain.User{ID: otherUserID, Timezone: "UTC"}

	logRepo := NewMockSleepLogRepository()
	logRepo.logs[logID] = &domain.SleepLog{
		ID:      logID,
		UserID:  otherUserID, // Owned by different user
		StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality: 7,
		Type:    domain.SleepTypeCore,
	}

	svc := NewSleepLogService(logRepo, userRepo)

	req := &domain.UpdateSleepLogRequest{
		Quality: intPtr(9),
	}

	_, err := svc.Update(context.Background(), userID, logID, req)
	if err != domain.ErrNotFound {
		t.Errorf("Update() error = %v, want %v (ownership check)", err, domain.ErrNotFound)
	}
}

func TestSleepLogService_Update_OverlapDetection(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()
	existingLogID := uuid.New()

	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	tests := []struct {
		name      string
		req       *domain.UpdateSleepLogRequest
		setupLogs func(*MockSleepLogRepository)
		wantErr   error
	}{
		{
			name: "update causes overlap with existing CORE",
			req: &domain.UpdateSleepLogRequest{
				StartAt: timePtr(time.Date(2024, 1, 16, 22, 0, 0, 0, time.UTC)),
				EndAt:   timePtr(time.Date(2024, 1, 17, 6, 0, 0, 0, time.UTC)),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				// Log being updated
				repo.logs[logID] = &domain.SleepLog{
					ID:      logID,
					UserID:  userID,
					StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
					Quality: 7,
					Type:    domain.SleepTypeCore,
				}
				// Existing log that would overlap
				repo.logs[existingLogID] = &domain.SleepLog{
					ID:      existingLogID,
					UserID:  userID,
					StartAt: time.Date(2024, 1, 16, 23, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 17, 7, 0, 0, 0, time.UTC),
					Quality: 8,
					Type:    domain.SleepTypeCore,
				}
			},
			wantErr: domain.ErrOverlappingSleep,
		},
		{
			name: "update does not cause self-overlap",
			req: &domain.UpdateSleepLogRequest{
				Quality: intPtr(9), // Just updating quality, same times
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				repo.logs[logID] = &domain.SleepLog{
					ID:      logID,
					UserID:  userID,
					StartAt: time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
					Quality: 7,
					Type:    domain.SleepTypeCore,
				}
			},
			wantErr: nil,
		},
		{
			name: "NAP overlap now blocked",
			req: &domain.UpdateSleepLogRequest{
				StartAt: timePtr(time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC)),
				EndAt:   timePtr(time.Date(2024, 1, 16, 15, 30, 0, 0, time.UTC)),
			},
			setupLogs: func(repo *MockSleepLogRepository) {
				// NAP being updated
				repo.logs[logID] = &domain.SleepLog{
					ID:      logID,
					UserID:  userID,
					StartAt: time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 16, 13, 0, 0, 0, time.UTC),
					Quality: 6,
					Type:    domain.SleepTypeNap,
				}
				// Existing NAP that would overlap
				repo.logs[existingLogID] = &domain.SleepLog{
					ID:      existingLogID,
					UserID:  userID,
					StartAt: time.Date(2024, 1, 16, 15, 0, 0, 0, time.UTC),
					EndAt:   time.Date(2024, 1, 16, 16, 0, 0, 0, time.UTC),
					Quality: 5,
					Type:    domain.SleepTypeNap,
				}
			},
			wantErr: domain.ErrOverlappingSleep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRepo := NewMockSleepLogRepository()
			if tt.setupLogs != nil {
				tt.setupLogs(logRepo)
			}

			svc := NewSleepLogService(logRepo, userRepo)
			_, err := svc.Update(context.Background(), userID, logID, tt.req)

			if err != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSleepLogService_Update_EmptyTimezoneIgnored(t *testing.T) {
	userID := uuid.New()
	logID := uuid.New()

	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	logRepo := NewMockSleepLogRepository()
	logRepo.logs[logID] = &domain.SleepLog{
		ID:            logID,
		UserID:        userID,
		StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality:       7,
		Type:          domain.SleepTypeCore,
		LocalTimezone: "Europe/Warsaw",
	}

	svc := NewSleepLogService(logRepo, userRepo)

	// Empty timezone should not change existing value
	req := &domain.UpdateSleepLogRequest{
		LocalTimezone: strPtr(""),
	}

	log, err := svc.Update(context.Background(), userID, logID, req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if log.LocalTimezone != "Europe/Warsaw" {
		t.Errorf("LocalTimezone = %s, want Europe/Warsaw (should not change with empty string)", log.LocalTimezone)
	}
}
