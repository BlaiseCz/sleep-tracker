package domain

import (
	"testing"
	"time"
	_ "time/tzdata" // Embed timezone database for CI/minimal containers

	"github.com/google/uuid"
)

func TestSleepLog_ToResponse_TimezoneConversion(t *testing.T) {
	tests := []struct {
		name               string
		sleepLog           SleepLog
		wantLocalStartHr   int
		wantLocalEndHr     int
		wantLocalStartDay  int
		wantLocalEndDay    int
		wantLocalStartZone string // Expected timezone name in LocalStartAt.Location()
	}{
		{
			name: "Poznan to San Francisco - 11h sleep in SF timezone",
			// Scenario: Person flew from Poznan to San Francisco
			// Fell asleep at 10 PM SF time (06:00 UTC next day)
			// Woke up at 9 AM SF time (17:00 UTC)
			// Duration: 11 hours
			// America/Los_Angeles in Jan = PST (UTC-8)
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),  // 10 PM Jan 15 in SF
				EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC), // 9 AM Jan 16 in SF
				Quality:       8,
				Type:          SleepTypeCore,
				LocalTimezone: "America/Los_Angeles",
			},
			wantLocalStartHr:   22, // 10 PM
			wantLocalEndHr:     9,  // 9 AM
			wantLocalStartDay:  15, // Jan 15
			wantLocalEndDay:    16, // Jan 16
			wantLocalStartZone: "PST",
		},
		{
			name: "Sleep in Poznan before flight - Europe/Warsaw timezone",
			// Scenario: Last night in Poznan before flying
			// Fell asleep at 11 PM Poznan time (22:00 UTC)
			// Woke up at 7 AM Poznan time (06:00 UTC)
			// Duration: 8 hours
			// Europe/Warsaw in Jan = CET (UTC+1)
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 14, 22, 0, 0, 0, time.UTC), // 11 PM Jan 14 in Poznan (UTC+1 winter)
				EndAt:         time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),  // 7 AM Jan 15 in Poznan
				Quality:       7,
				Type:          SleepTypeCore,
				LocalTimezone: "Europe/Warsaw",
			},
			wantLocalStartHr:   23, // 11 PM
			wantLocalEndHr:     7,  // 7 AM
			wantLocalStartDay:  14, // Jan 14
			wantLocalEndDay:    15, // Jan 15
			wantLocalStartZone: "CET",
		},
		{
			name: "UTC timezone explicit",
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          SleepTypeCore,
				LocalTimezone: "UTC",
			},
			wantLocalStartHr:   23,
			wantLocalEndHr:     7,
			wantLocalStartDay:  15,
			wantLocalEndDay:    16,
			wantLocalStartZone: "UTC",
		},
		{
			name: "Tokyo timezone - crossing midnight",
			// Sleep from 11 PM to 8 AM Tokyo time
			// Asia/Tokyo = JST (UTC+9, no DST)
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), // 11 PM Jan 15 in Tokyo
				EndAt:         time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC), // 8 AM Jan 16 in Tokyo
				Quality:       9,
				Type:          SleepTypeCore,
				LocalTimezone: "Asia/Tokyo",
			},
			wantLocalStartHr:   23, // 11 PM
			wantLocalEndHr:     8,  // 8 AM
			wantLocalStartDay:  15, // Jan 15
			wantLocalEndDay:    16, // Jan 16
			wantLocalStartZone: "JST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.sleepLog.ToResponse()

			// Verify UTC times are preserved (instant equality)
			if !resp.StartAt.Equal(tt.sleepLog.StartAt) {
				t.Errorf("StartAt instant mismatch: got %v, want %v", resp.StartAt, tt.sleepLog.StartAt)
			}
			if !resp.EndAt.Equal(tt.sleepLog.EndAt) {
				t.Errorf("EndAt instant mismatch: got %v, want %v", resp.EndAt, tt.sleepLog.EndAt)
			}

			// Verify local wall-clock times
			if resp.LocalStartAt.Hour() != tt.wantLocalStartHr {
				t.Errorf("LocalStartAt hour = %d, want %d", resp.LocalStartAt.Hour(), tt.wantLocalStartHr)
			}
			if resp.LocalEndAt.Hour() != tt.wantLocalEndHr {
				t.Errorf("LocalEndAt hour = %d, want %d", resp.LocalEndAt.Hour(), tt.wantLocalEndHr)
			}
			if resp.LocalStartAt.Day() != tt.wantLocalStartDay {
				t.Errorf("LocalStartAt day = %d, want %d", resp.LocalStartAt.Day(), tt.wantLocalStartDay)
			}
			if resp.LocalEndAt.Day() != tt.wantLocalEndDay {
				t.Errorf("LocalEndAt day = %d, want %d", resp.LocalEndAt.Day(), tt.wantLocalEndDay)
			}

			// Verify local times are tagged with correct timezone
			zoneName, _ := resp.LocalStartAt.Zone()
			if zoneName != tt.wantLocalStartZone {
				t.Errorf("LocalStartAt zone = %s, want %s", zoneName, tt.wantLocalStartZone)
			}

			// Verify LocalTimezone string is preserved as-is (current contract)
			if resp.LocalTimezone != tt.sleepLog.LocalTimezone {
				t.Errorf("LocalTimezone = %s, want %s", resp.LocalTimezone, tt.sleepLog.LocalTimezone)
			}
		})
	}
}

func TestSleepLog_ToResponse_DurationPreserved(t *testing.T) {
	// The 11-hour sleep scenario: duration should be the same regardless of timezone
	sleepLog := SleepLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),  // 10 PM Jan 15 SF
		EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC), // 9 AM Jan 16 SF
		Quality:       8,
		Type:          SleepTypeCore,
		LocalTimezone: "America/Los_Angeles",
	}

	resp := sleepLog.ToResponse()

	// Duration in UTC
	utcDuration := resp.EndAt.Sub(resp.StartAt)
	if utcDuration != 11*time.Hour {
		t.Errorf("UTC duration = %v, want 11h", utcDuration)
	}

	// Duration in local time should be the same
	localDuration := resp.LocalEndAt.Sub(resp.LocalStartAt)
	if localDuration != 11*time.Hour {
		t.Errorf("Local duration = %v, want 11h", localDuration)
	}
}

// TestSleepLog_ToResponse_TimezoneFallback tests the contract for invalid/empty timezones:
// - LocalTimezone string is preserved as-is (even if invalid)
// - Local times are computed using UTC when timezone is empty or invalid
func TestSleepLog_ToResponse_TimezoneFallback(t *testing.T) {
	tests := []struct {
		name                  string
		inputTimezone         string
		wantLocalTimezone     string // What LocalTimezone field should contain
		wantLocalStartHr      int    // Expected hour (UTC fallback = same as input)
		wantLocalStartZoneName string // Zone name from LocalStartAt.Zone()
	}{
		{
			name:                  "Empty timezone - falls back to UTC for computation, preserves empty string",
			inputTimezone:         "",
			wantLocalTimezone:     "", // Contract: preserve as-is
			wantLocalStartHr:      23, // Same as UTC input
			wantLocalStartZoneName: "UTC",
		},
		{
			name:                  "Invalid timezone - falls back to UTC for computation, preserves invalid string",
			inputTimezone:         "Invalid/Timezone",
			wantLocalTimezone:     "Invalid/Timezone", // Contract: preserve as-is
			wantLocalStartHr:      23,                 // Same as UTC input
			wantLocalStartZoneName: "UTC",
		},
		{
			name:                  "Gibberish timezone - falls back to UTC",
			inputTimezone:         "NotATimezone",
			wantLocalTimezone:     "NotATimezone", // Contract: preserve as-is
			wantLocalStartHr:      23,
			wantLocalStartZoneName: "UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sleepLog := SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC),
				EndAt:         time.Date(2024, 1, 16, 7, 0, 0, 0, time.UTC),
				Quality:       8,
				Type:          SleepTypeCore,
				LocalTimezone: tt.inputTimezone,
			}

			resp := sleepLog.ToResponse()

			// Verify LocalTimezone is preserved as-is (current contract)
			if resp.LocalTimezone != tt.wantLocalTimezone {
				t.Errorf("LocalTimezone = %q, want %q", resp.LocalTimezone, tt.wantLocalTimezone)
			}

			// Verify local times fall back to UTC
			if resp.LocalStartAt.Hour() != tt.wantLocalStartHr {
				t.Errorf("LocalStartAt hour = %d, want %d (UTC fallback)", resp.LocalStartAt.Hour(), tt.wantLocalStartHr)
			}

			// Verify the Location is actually UTC
			zoneName, _ := resp.LocalStartAt.Zone()
			if zoneName != tt.wantLocalStartZoneName {
				t.Errorf("LocalStartAt zone = %s, want %s", zoneName, tt.wantLocalStartZoneName)
			}
		})
	}
}

// TestSleepLog_ToResponse_DSTSpringForward tests behavior during DST "spring forward"
// In America/Los_Angeles, 2024-03-10 at 2:00 AM clocks jump to 3:00 AM (losing 1 hour)
// This is a critical edge case for sleep tracking apps
func TestSleepLog_ToResponse_DSTSpringForward(t *testing.T) {
	// Sleep from 1:30 AM to 3:30 AM local time on DST transition day
	// Wall-clock shows 2 hours, but elapsed time is only 1 hour
	//
	// 2024-03-10 01:30 PST (UTC-8) = 2024-03-10 09:30 UTC
	// 2024-03-10 03:30 PDT (UTC-7) = 2024-03-10 10:30 UTC
	// Elapsed: 1 hour (not 2!)
	sleepLog := SleepLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		StartAt:       time.Date(2024, 3, 10, 9, 30, 0, 0, time.UTC),  // 1:30 AM PST
		EndAt:         time.Date(2024, 3, 10, 10, 30, 0, 0, time.UTC), // 3:30 AM PDT
		Quality:       6,
		Type:          SleepTypeCore,
		LocalTimezone: "America/Los_Angeles",
	}

	resp := sleepLog.ToResponse()

	// Elapsed duration (what actually matters for sleep tracking) is 1 hour
	elapsedDuration := resp.EndAt.Sub(resp.StartAt)
	if elapsedDuration != 1*time.Hour {
		t.Errorf("Elapsed duration = %v, want 1h", elapsedDuration)
	}

	// Wall-clock duration appears to be 2 hours (1:30 AM → 3:30 AM)
	// but Sub() on time.Time measures elapsed time, not wall-clock difference
	localDuration := resp.LocalEndAt.Sub(resp.LocalStartAt)
	if localDuration != 1*time.Hour {
		t.Errorf("Local duration = %v, want 1h (elapsed, not wall-clock)", localDuration)
	}

	// Verify the local wall-clock times
	if resp.LocalStartAt.Hour() != 1 || resp.LocalStartAt.Minute() != 30 {
		t.Errorf("LocalStartAt = %02d:%02d, want 01:30", resp.LocalStartAt.Hour(), resp.LocalStartAt.Minute())
	}
	if resp.LocalEndAt.Hour() != 3 || resp.LocalEndAt.Minute() != 30 {
		t.Errorf("LocalEndAt = %02d:%02d, want 03:30", resp.LocalEndAt.Hour(), resp.LocalEndAt.Minute())
	}

	// Verify timezone abbreviations change (PST → PDT)
	startZone, _ := resp.LocalStartAt.Zone()
	endZone, _ := resp.LocalEndAt.Zone()
	if startZone != "PST" {
		t.Errorf("Start zone = %s, want PST", startZone)
	}
	if endZone != "PDT" {
		t.Errorf("End zone = %s, want PDT", endZone)
	}
}

// TestSleepLog_ToResponse_DSTFallBack tests behavior during DST "fall back"
// In America/Los_Angeles, 2024-11-03 at 2:00 AM clocks fall back to 1:00 AM
// The hour from 1:00-2:00 AM occurs twice (ambiguous local time)
func TestSleepLog_ToResponse_DSTFallBack(t *testing.T) {
	// Sleep from 12:30 AM to 2:30 AM local time on DST transition day
	// Wall-clock shows 2 hours, but elapsed time is 3 hours
	//
	// 2024-11-03 00:30 PDT (UTC-7) = 2024-11-03 07:30 UTC
	// 2024-11-03 02:30 PST (UTC-8) = 2024-11-03 10:30 UTC
	// Elapsed: 3 hours (not 2!)
	sleepLog := SleepLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		StartAt:       time.Date(2024, 11, 3, 7, 30, 0, 0, time.UTC),  // 12:30 AM PDT
		EndAt:         time.Date(2024, 11, 3, 10, 30, 0, 0, time.UTC), // 2:30 AM PST
		Quality:       7,
		Type:          SleepTypeCore,
		LocalTimezone: "America/Los_Angeles",
	}

	resp := sleepLog.ToResponse()

	// Elapsed duration is 3 hours
	elapsedDuration := resp.EndAt.Sub(resp.StartAt)
	if elapsedDuration != 3*time.Hour {
		t.Errorf("Elapsed duration = %v, want 3h", elapsedDuration)
	}

	// Local duration also measures elapsed time
	localDuration := resp.LocalEndAt.Sub(resp.LocalStartAt)
	if localDuration != 3*time.Hour {
		t.Errorf("Local duration = %v, want 3h", localDuration)
	}

	// Verify the local wall-clock times
	if resp.LocalStartAt.Hour() != 0 || resp.LocalStartAt.Minute() != 30 {
		t.Errorf("LocalStartAt = %02d:%02d, want 00:30", resp.LocalStartAt.Hour(), resp.LocalStartAt.Minute())
	}
	if resp.LocalEndAt.Hour() != 2 || resp.LocalEndAt.Minute() != 30 {
		t.Errorf("LocalEndAt = %02d:%02d, want 02:30", resp.LocalEndAt.Hour(), resp.LocalEndAt.Minute())
	}

	// Verify timezone abbreviations change (PDT → PST)
	startZone, _ := resp.LocalStartAt.Zone()
	endZone, _ := resp.LocalEndAt.Zone()
	if startZone != "PDT" {
		t.Errorf("Start zone = %s, want PDT", startZone)
	}
	if endZone != "PST" {
		t.Errorf("End zone = %s, want PST", endZone)
	}
}

// TestSleepLog_RoundTrip_TravelScenario simulates creating a sleep log and then
// retrieving it, verifying the response is consistent (as would happen via List endpoint)
func TestSleepLog_RoundTrip_TravelScenario(t *testing.T) {
	tests := []struct {
		name              string
		sleepLog          SleepLog
		wantLocalStartHr  int
		wantLocalEndHr    int
		wantLocalStartDay int
		wantLocalEndDay   int
		wantDuration      time.Duration
	}{
		{
			name: "Poznan to SF - 11h sleep - create and get",
			// User creates log after sleeping in SF
			// When retrieved via List, should show same local times
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),  // 10 PM Jan 15 SF
				EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC), // 9 AM Jan 16 SF
				Quality:       8,
				Type:          SleepTypeCore,
				LocalTimezone: "America/Los_Angeles",
				CreatedAt:     time.Now(),
			},
			wantLocalStartHr:  22,
			wantLocalEndHr:    9,
			wantLocalStartDay: 15,
			wantLocalEndDay:   16,
			wantDuration:      11 * time.Hour,
		},
		{
			name: "Poznan sleep before flight - create and get",
			sleepLog: SleepLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				StartAt:       time.Date(2024, 1, 14, 22, 0, 0, 0, time.UTC), // 11 PM Jan 14 Warsaw
				EndAt:         time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),  // 7 AM Jan 15 Warsaw
				Quality:       7,
				Type:          SleepTypeCore,
				LocalTimezone: "Europe/Warsaw",
				CreatedAt:     time.Now(),
			},
			wantLocalStartHr:  23,
			wantLocalEndHr:    7,
			wantLocalStartDay: 14,
			wantLocalEndDay:   15,
			wantDuration:      8 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate "Create" - store the log (in real code this goes to DB)
			storedLog := tt.sleepLog

			// Simulate "Get/List" - retrieve and convert to response
			// This is what happens when List endpoint returns data
			response := storedLog.ToResponse()

			// Verify ID is preserved
			if response.ID != storedLog.ID {
				t.Errorf("ID mismatch: got %v, want %v", response.ID, storedLog.ID)
			}

			// Verify UTC times are preserved exactly
			if !response.StartAt.Equal(storedLog.StartAt) {
				t.Errorf("StartAt mismatch: got %v, want %v", response.StartAt, storedLog.StartAt)
			}
			if !response.EndAt.Equal(storedLog.EndAt) {
				t.Errorf("EndAt mismatch: got %v, want %v", response.EndAt, storedLog.EndAt)
			}

			// Verify local times are correct
			if response.LocalStartAt.Hour() != tt.wantLocalStartHr {
				t.Errorf("LocalStartAt hour = %d, want %d", response.LocalStartAt.Hour(), tt.wantLocalStartHr)
			}
			if response.LocalEndAt.Hour() != tt.wantLocalEndHr {
				t.Errorf("LocalEndAt hour = %d, want %d", response.LocalEndAt.Hour(), tt.wantLocalEndHr)
			}
			if response.LocalStartAt.Day() != tt.wantLocalStartDay {
				t.Errorf("LocalStartAt day = %d, want %d", response.LocalStartAt.Day(), tt.wantLocalStartDay)
			}
			if response.LocalEndAt.Day() != tt.wantLocalEndDay {
				t.Errorf("LocalEndAt day = %d, want %d", response.LocalEndAt.Day(), tt.wantLocalEndDay)
			}

			// Verify timezone is preserved
			if response.LocalTimezone != storedLog.LocalTimezone {
				t.Errorf("LocalTimezone = %s, want %s", response.LocalTimezone, storedLog.LocalTimezone)
			}

			// Verify duration is consistent in both UTC and local
			utcDuration := response.EndAt.Sub(response.StartAt)
			localDuration := response.LocalEndAt.Sub(response.LocalStartAt)

			if utcDuration != tt.wantDuration {
				t.Errorf("UTC duration = %v, want %v", utcDuration, tt.wantDuration)
			}
			if localDuration != tt.wantDuration {
				t.Errorf("Local duration = %v, want %v", localDuration, tt.wantDuration)
			}
			if utcDuration != localDuration {
				t.Errorf("Duration mismatch between UTC (%v) and local (%v)", utcDuration, localDuration)
			}

			// Verify quality and type are preserved
			if response.Quality != storedLog.Quality {
				t.Errorf("Quality = %d, want %d", response.Quality, storedLog.Quality)
			}
			if response.Type != storedLog.Type {
				t.Errorf("Type = %s, want %s", response.Type, storedLog.Type)
			}
		})
	}
}

// TestSleepLog_MultipleRetrievals verifies that retrieving the same log multiple times
// produces identical results (idempotent reads)
func TestSleepLog_MultipleRetrievals(t *testing.T) {
	sleepLog := SleepLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		StartAt:       time.Date(2024, 1, 16, 6, 0, 0, 0, time.UTC),
		EndAt:         time.Date(2024, 1, 16, 17, 0, 0, 0, time.UTC),
		Quality:       8,
		Type:          SleepTypeCore,
		LocalTimezone: "America/Los_Angeles",
		CreatedAt:     time.Now(),
	}

	// Get response multiple times
	resp1 := sleepLog.ToResponse()
	resp2 := sleepLog.ToResponse()
	resp3 := sleepLog.ToResponse()

	// All should be identical
	if resp1.LocalStartAt.Hour() != resp2.LocalStartAt.Hour() || resp2.LocalStartAt.Hour() != resp3.LocalStartAt.Hour() {
		t.Error("LocalStartAt hour differs between retrievals")
	}
	if resp1.LocalEndAt.Hour() != resp2.LocalEndAt.Hour() || resp2.LocalEndAt.Hour() != resp3.LocalEndAt.Hour() {
		t.Error("LocalEndAt hour differs between retrievals")
	}
	if !resp1.StartAt.Equal(resp2.StartAt) || !resp2.StartAt.Equal(resp3.StartAt) {
		t.Error("StartAt differs between retrievals")
	}
	if !resp1.EndAt.Equal(resp2.EndAt) || !resp2.EndAt.Equal(resp3.EndAt) {
		t.Error("EndAt differs between retrievals")
	}
}
