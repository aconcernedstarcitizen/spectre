package main

import (
	"testing"
	"time"
)

func TestParseSaleTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantYear    int
		wantMonth   time.Month
		wantDay     int
		wantHour    int
		wantMinute  int
		shouldError bool
	}{
		{
			name:       "Friendly format YYYY-MM-DD HH:MM",
			input:      "2025-01-15 16:00",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   16,
			wantMinute: 0,
		},
		{
			name:       "Friendly format with seconds",
			input:      "2025-01-15 20:30:45",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   20,
			wantMinute: 30,
		},
		{
			name:       "Friendly format with UTC suffix",
			input:      "2025-01-15 16:00 UTC",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   16,
			wantMinute: 0,
		},
		{
			name:       "RFC3339 format (backward compatibility)",
			input:      "2025-01-15T16:00:00Z",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   16,
			wantMinute: 0,
		},
		{
			name:       "Midnight wave",
			input:      "2025-01-16 00:00",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    16,
			wantHour:   0,
			wantMinute: 0,
		},
		{
			name:       "Early morning wave",
			input:      "2025-01-16 04:00",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    16,
			wantHour:   4,
			wantMinute: 0,
		},
		{
			name:        "Invalid format",
			input:       "not a date",
			shouldError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			shouldError: true,
		},
		{
			name:       "With extra whitespace",
			input:      "  2025-01-15 16:00  ",
			wantYear:   2025,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   16,
			wantMinute: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSaleTime(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			// Verify the parsed time
			if got.Year() != tt.wantYear {
				t.Errorf("Year mismatch: got %d, want %d", got.Year(), tt.wantYear)
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("Month mismatch: got %v, want %v", got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("Day mismatch: got %d, want %d", got.Day(), tt.wantDay)
			}
			if got.Hour() != tt.wantHour {
				t.Errorf("Hour mismatch: got %d, want %d", got.Hour(), tt.wantHour)
			}
			if got.Minute() != tt.wantMinute {
				t.Errorf("Minute mismatch: got %d, want %d", got.Minute(), tt.wantMinute)
			}

			// Verify timezone is UTC
			if got.Location() != time.UTC {
				t.Errorf("Timezone mismatch: got %v, want UTC", got.Location())
			}
		})
	}
}

func TestParseSaleTimeAllStandardWaves(t *testing.T) {
	// Test parsing all 6 standard CIG wave times
	waves := []string{
		"2025-01-15 16:00",
		"2025-01-15 20:00",
		"2025-01-16 00:00",
		"2025-01-16 04:00",
		"2025-01-16 08:00",
		"2025-01-16 12:00",
	}

	for i, waveStr := range waves {
		t.Run("Wave "+string(rune('1'+i)), func(t *testing.T) {
			parsed, err := ParseSaleTime(waveStr)
			if err != nil {
				t.Errorf("Failed to parse wave %d (%s): %v", i+1, waveStr, err)
				return
			}

			// Verify it's in UTC
			if parsed.Location() != time.UTC {
				t.Errorf("Wave %d not in UTC: %v", i+1, parsed.Location())
			}

			// Verify we can format it back to RFC3339
			rfc3339 := parsed.Format(time.RFC3339)
			if rfc3339 == "" {
				t.Errorf("Failed to format wave %d to RFC3339", i+1)
			}
		})
	}
}
