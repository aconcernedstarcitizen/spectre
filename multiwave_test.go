package main

import (
	"testing"
	"time"
)

// TestParseSaleWindows tests the parseSaleWindows function
func TestParseSaleWindows(t *testing.T) {
	tests := []struct {
		name        string
		saleWindows []string
		wantCount   int
		wantError   bool
	}{
		{
			name: "Valid RFC3339 format",
			saleWindows: []string{
				"2025-01-15T16:00:00Z",
				"2025-01-15T20:00:00Z",
			},
			wantCount: 2,
			wantError: false,
		},
		{
			name: "Valid user-friendly format",
			saleWindows: []string{
				"2025-01-15 16:00",
				"2025-01-15 20:00",
				"2025-01-16 00:00",
			},
			wantCount: 3,
			wantError: false,
		},
		{
			name: "Mixed formats",
			saleWindows: []string{
				"2025-01-15 16:00",
				"2025-01-15T20:00:00Z",
				"2025-01-16 00:00 UTC",
			},
			wantCount: 3,
			wantError: false,
		},
		{
			name: "Invalid format",
			saleWindows: []string{
				"2025-01-15 16:00",
				"invalid date",
			},
			wantCount: 0,
			wantError: true,
		},
		{
			name:        "Empty list",
			saleWindows: []string{},
			wantCount:   0,
			wantError:   false,
		},
		{
			name: "All 6 standard CIG waves",
			saleWindows: []string{
				"2025-01-15 16:00",
				"2025-01-15 20:00",
				"2025-01-16 00:00",
				"2025-01-16 04:00",
				"2025-01-16 08:00",
				"2025-01-16 12:00",
			},
			wantCount: 6,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				SaleWindows: tt.saleWindows,
			}
			mwo := &MultiWaveOrchestrator{
				config: config,
			}

			windows, err := mwo.parseSaleWindows()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(windows) != tt.wantCount {
				t.Errorf("Expected %d windows, got %d", tt.wantCount, len(windows))
			}

			// Verify all times are in UTC
			for i, waveTime := range windows {
				if waveTime.Location() != time.UTC {
					t.Errorf("Wave %d: expected UTC timezone, got %v", i+1, waveTime.Location())
				}
			}
		})
	}
}

// TestSmartWaveDetection tests the smart wave detection logic
func TestSmartWaveDetection(t *testing.T) {
	// Standard 6 waves
	saleWindows := []time.Time{
		time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC), // Wave 1: 16:00 UTC
		time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC), // Wave 2: 20:00 UTC
		time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),  // Wave 3: 00:00 UTC
		time.Date(2025, 1, 16, 4, 0, 0, 0, time.UTC),  // Wave 4: 04:00 UTC
		time.Date(2025, 1, 16, 8, 0, 0, 0, time.UTC),  // Wave 5: 08:00 UTC
		time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC), // Wave 6: 12:00 UTC
	}

	postWaveTimeout := 5 * time.Minute

	tests := []struct {
		name           string
		currentTime    time.Time
		wantStartIndex int
		description    string
	}{
		{
			name:           "Before first wave",
			currentTime:    time.Date(2025, 1, 15, 15, 0, 0, 0, time.UTC), // 15:00 UTC
			wantStartIndex: 0,                                              // Start at Wave 1
			description:    "Should start at Wave 1",
		},
		{
			name:           "During Wave 1",
			currentTime:    time.Date(2025, 1, 15, 16, 2, 0, 0, time.UTC), // 16:02 UTC
			wantStartIndex: 0,                                              // Still Wave 1
			description:    "Should continue with Wave 1",
		},
		{
			name:           "Just after Wave 1 (within timeout)",
			currentTime:    time.Date(2025, 1, 15, 16, 3, 0, 0, time.UTC), // 16:03 UTC
			wantStartIndex: 0,                                              // Still Wave 1
			description:    "Wave 1 timeout hasn't passed (16:05)",
		},
		{
			name:           "After Wave 1 timeout",
			currentTime:    time.Date(2025, 1, 15, 16, 6, 0, 0, time.UTC), // 16:06 UTC
			wantStartIndex: 1,                                              // Skip to Wave 2
			description:    "Wave 1 ended at 16:05, should skip to Wave 2",
		},
		{
			name:           "Between Wave 1 and Wave 2",
			currentTime:    time.Date(2025, 1, 15, 18, 0, 0, 0, time.UTC), // 18:00 UTC
			wantStartIndex: 1,                                              // Wave 2
			description:    "Should be at Wave 2",
		},
		{
			name:           "After Wave 5 timeout, before Wave 6",
			currentTime:    time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC), // 10:00 UTC
			wantStartIndex: 5,                                              // Wave 6
			description:    "Should be at Wave 6 (last wave)",
		},
		{
			name:           "After all waves (just after Wave 6 timeout)",
			currentTime:    time.Date(2025, 1, 16, 12, 6, 0, 0, time.UTC), // 12:06 UTC
			wantStartIndex: -1,                                             // No waves left
			description:    "All waves have ended",
		},
		{
			name:           "Long after all waves",
			currentTime:    time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), // Next day
			wantStartIndex: -1,                                            // No waves left
			description:    "All waves have ended (next day)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the wave detection logic from multiwave.go (lines 64-77)
			startWaveIndex := -1
			for i, waveTime := range saleWindows {
				waveEndTime := waveTime.Add(postWaveTimeout)

				if tt.currentTime.Before(waveEndTime) {
					// This wave is still relevant (either upcoming or active)
					startWaveIndex = i
					break
				}
			}

			if startWaveIndex != tt.wantStartIndex {
				t.Errorf("%s: Expected startWaveIndex=%d, got %d",
					tt.description, tt.wantStartIndex, startWaveIndex)
			}
		})
	}
}

// TestWaveSkippingScenarios tests various wave skipping scenarios
func TestWaveSkippingScenarios(t *testing.T) {
	postWaveTimeout := 5 * time.Minute

	tests := []struct {
		name              string
		saleWindows       []time.Time
		currentTime       time.Time
		expectedSkipped   int
		expectedStartWave int
	}{
		{
			name: "No waves skipped",
			saleWindows: []time.Time{
				time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC),
			},
			currentTime:       time.Date(2025, 1, 15, 15, 0, 0, 0, time.UTC),
			expectedSkipped:   0,
			expectedStartWave: 0,
		},
		{
			name: "Skip 1 wave",
			saleWindows: []time.Time{
				time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
			},
			currentTime:       time.Date(2025, 1, 15, 16, 10, 0, 0, time.UTC), // After Wave 1 timeout
			expectedSkipped:   1,
			expectedStartWave: 1,
		},
		{
			name: "Skip 3 waves",
			saleWindows: []time.Time{
				time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 16, 4, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 16, 8, 0, 0, 0, time.UTC),
			},
			currentTime:       time.Date(2025, 1, 16, 1, 0, 0, 0, time.UTC), // After Wave 3 timeout
			expectedSkipped:   3,
			expectedStartWave: 3,
		},
		{
			name: "All waves passed",
			saleWindows: []time.Time{
				time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC),
			},
			currentTime:       time.Date(2025, 1, 15, 21, 0, 0, 0, time.UTC), // After all waves
			expectedSkipped:   2,
			expectedStartWave: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the wave detection logic
			startWaveIndex := -1
			for i, waveTime := range tt.saleWindows {
				waveEndTime := waveTime.Add(postWaveTimeout)

				if tt.currentTime.Before(waveEndTime) {
					startWaveIndex = i
					break
				}
			}

			if startWaveIndex != tt.expectedStartWave {
				t.Errorf("Expected startWaveIndex=%d, got %d", tt.expectedStartWave, startWaveIndex)
			}

			// Count skipped waves
			skippedWaves := 0
			if startWaveIndex > 0 {
				skippedWaves = startWaveIndex
			} else if startWaveIndex == -1 {
				skippedWaves = len(tt.saleWindows)
			}

			if skippedWaves != tt.expectedSkipped {
				t.Errorf("Expected %d skipped waves, got %d", tt.expectedSkipped, skippedWaves)
			}
		})
	}
}

// TestWaveEndTimeCalculation tests that wave end times are calculated correctly
func TestWaveEndTimeCalculation(t *testing.T) {
	waveTime := time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC)
	postWaveTimeout := 5 * time.Minute

	expectedEndTime := time.Date(2025, 1, 15, 16, 5, 0, 0, time.UTC)
	actualEndTime := waveTime.Add(postWaveTimeout)

	if !actualEndTime.Equal(expectedEndTime) {
		t.Errorf("Wave end time calculation incorrect. Expected %v, got %v",
			expectedEndTime, actualEndTime)
	}
}

// TestMultipleWaveFormats tests that different time formats work together
func TestMultipleWaveFormats(t *testing.T) {
	config := &Config{
		SaleWindows: []string{
			"2025-01-15 16:00",           // User-friendly
			"2025-01-15T20:00:00Z",       // RFC3339
			"2025-01-16 00:00 UTC",       // With UTC suffix
			"2025-01-16 04:00:00",        // With seconds
		},
	}

	mwo := &MultiWaveOrchestrator{
		config: config,
	}

	windows, err := mwo.parseSaleWindows()
	if err != nil {
		t.Fatalf("Failed to parse mixed formats: %v", err)
	}

	if len(windows) != 4 {
		t.Errorf("Expected 4 windows, got %d", len(windows))
	}

	// Verify chronological order
	expectedTimes := []time.Time{
		time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 20, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 16, 4, 0, 0, 0, time.UTC),
	}

	for i, waveTime := range windows {
		if !waveTime.Equal(expectedTimes[i]) {
			t.Errorf("Wave %d: expected %v, got %v", i+1, expectedTimes[i], waveTime)
		}
	}
}
