package service

import (
	"context"
	"testing"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/google/uuid"
)

// Mocks are defined in mocks_test.go

func TestSleepLogService_Create(t *testing.T) {
	userID := uuid.New()

	// Setup user repo with existing user
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	tests := []struct {
		name      string
		req       *domain.CreateSleepLogRequest
		setupLogs func(*MockSleepLogRepository)
		wantErr   error
		wantExist bool
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

func TestSleepLogService_List_DefaultsAndCursor(t *testing.T) {
	userID := uuid.New()
	userRepo := NewMockUserRepository()
	userRepo.users[userID] = &domain.User{ID: userID, Timezone: "UTC"}

	logs := make([]domain.SleepLog, 25)
	base := time.Date(2024, 1, 31, 23, 0, 0, 0, time.UTC)
	for i := 0; i < len(logs); i++ {
		logs[i] = domain.SleepLog{
			ID:      uuid.New(),
			UserID:  userID,
			StartAt: base.Add(-time.Duration(i) * time.Hour),
			EndAt:   base.Add(-time.Duration(i) * time.Hour).Add(8 * time.Hour),
		}
	}

	logRepo := NewMockSleepLogRepository()
	logRepo.listResult = logs

	svc := NewSleepLogService(logRepo, userRepo)

	resp, err := svc.List(context.Background(), userID, domain.SleepLogFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(resp.Data) != 20 {
		t.Fatalf("expected default 20 results, got %d", len(resp.Data))
	}
	if !resp.Pagination.HasMore {
		t.Fatalf("expected has_more true when more records exist")
	}
	if resp.Pagination.NextCursor == "" {
		t.Fatalf("expected next cursor to be populated")
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

// strPtr is defined in mocks_test.go

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

// TestSleepLogService_Create_ClientRequestIDScopedPerUser verifies that the same
// client_request_id can be reused by different users without being treated as
// the same request, while idempotency still holds per user.
func TestSleepLogService_Create_ClientRequestIDScopedPerUser(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()

	userRepo := NewMockUserRepository()
	userRepo.users[userA] = &domain.User{ID: userA, Timezone: "UTC"}
	userRepo.users[userB] = &domain.User{ID: userB, Timezone: "UTC"}

	logRepo := NewMockSleepLogRepository()
	svc := NewSleepLogService(logRepo, userRepo)

	clientReqID := "req-123"

	reqA := &domain.CreateSleepLogRequest{
		StartAt:         time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
		EndAt:           time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
		Quality:         8,
		Type:            domain.SleepTypeCore,
		ClientRequestID: strPtr(clientReqID),
	}
	reqB := &domain.CreateSleepLogRequest{
		StartAt:         time.Date(2024, 1, 16, 23, 0, 0, 0, time.UTC),
		EndAt:           time.Date(2024, 1, 17, 7, 0, 0, 0, time.UTC),
		Quality:         7,
		Type:            domain.SleepTypeCore,
		ClientRequestID: strPtr(clientReqID),
	}

	// First call for userA should create a new log
	logA, isExistingA, err := svc.Create(context.Background(), userA, reqA)
	if err != nil {
		t.Fatalf("Create() for userA returned error: %v", err)
	}
	if isExistingA {
		t.Fatalf("Create() for userA should not be treated as existing on first call")
	}
	if logA == nil || logA.UserID != userA {
		t.Fatalf("Create() for userA returned invalid log: %+v", logA)
	}

	// Second call for the same user and same client_request_id should be idempotent
	logA2, isExistingA2, err := svc.Create(context.Background(), userA, reqA)
	if err != nil {
		t.Fatalf("Second Create() for userA returned error: %v", err)
	}
	if !isExistingA2 {
		t.Fatalf("Second Create() for userA should be treated as existing (idempotent)")
	}
	if logA2.ID != logA.ID {
		t.Fatalf("Second Create() for userA returned different log ID: first=%v second=%v", logA.ID, logA2.ID)
	}

	// Call for a different user with the same client_request_id should create a new log
	logB, isExistingB, err := svc.Create(context.Background(), userB, reqB)
	if err != nil {
		t.Fatalf("Create() for userB returned error: %v", err)
	}
	if isExistingB {
		t.Fatalf("Create() for userB should not be treated as existing when using same client_request_id as another user")
	}
	if logB == nil || logB.UserID != userB {
		t.Fatalf("Create() for userB returned invalid log: %+v", logB)
	}
	if logB.ID == logA.ID {
		t.Fatalf("Create() for userB should produce a different log ID than userA; both are %v", logB.ID)
	}
}
