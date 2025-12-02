package main

import (
	"fmt"
	"net/http"
	"time"
)

// TimeSync handles time synchronization with reliable time servers
type TimeSync struct {
	offset        time.Duration
	lastSyncTime  time.Time
	synced        bool
	debugMode     bool
}

// NewTimeSync creates a new TimeSync instance
func NewTimeSync(debugMode bool) *TimeSync {
	return &TimeSync{
		debugMode: debugMode,
	}
}

// Sync synchronizes time with multiple reliable time servers
// It makes HTTP HEAD requests and parses the Date header
func (ts *TimeSync) Sync() error {
	// Try multiple time servers for reliability
	servers := []string{
		"https://www.google.com",
		"https://www.cloudflare.com",
		"https://www.amazon.com",
	}

	var totalOffset time.Duration
	successCount := 0

	for _, server := range servers {
		offset, err := ts.getTimeOffset(server)
		if err != nil {
			if ts.debugMode {
				fmt.Printf("⚠️  Time sync failed for %s: %v\n", server, err)
			}
			continue
		}

		totalOffset += offset
		successCount++

		if ts.debugMode {
			fmt.Printf("✓ Time offset from %s: %v\n", server, offset)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to sync time with any server")
	}

	// Calculate average offset
	ts.offset = totalOffset / time.Duration(successCount)
	ts.lastSyncTime = time.Now()
	ts.synced = true

	if ts.debugMode {
		fmt.Printf("✓ Time synchronized (average offset: %v)\n", ts.offset)
	}

	return nil
}

// getTimeOffset makes an HTTP HEAD request and calculates time offset
func (ts *TimeSync) getTimeOffset(url string) (time.Duration, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Record time before request
	beforeRequest := time.Now()

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Record time after request
	afterRequest := time.Now()

	// Parse Date header
	dateHeader := resp.Header.Get("Date")
	if dateHeader == "" {
		return 0, fmt.Errorf("no Date header in response")
	}

	serverTime, err := http.ParseTime(dateHeader)
	if err != nil {
		return 0, fmt.Errorf("failed to parse Date header: %w", err)
	}

	// Estimate network latency (round trip time / 2)
	latency := afterRequest.Sub(beforeRequest) / 2

	// Calculate offset: server time - local time (adjusted for latency)
	localTime := beforeRequest.Add(latency)
	offset := serverTime.Sub(localTime)

	return offset, nil
}

// Now returns the current synchronized time
func (ts *TimeSync) Now() time.Time {
	if !ts.synced {
		// If not synced, return local time
		return time.Now()
	}

	// Return local time adjusted by offset
	return time.Now().Add(ts.offset)
}

// IsSynced returns whether time has been synchronized
func (ts *TimeSync) IsSynced() bool {
	return ts.synced
}

// GetOffset returns the calculated time offset
func (ts *TimeSync) GetOffset() time.Duration {
	return ts.offset
}

// ShouldResync checks if we should resync (e.g., every hour)
func (ts *TimeSync) ShouldResync() bool {
	if !ts.synced {
		return true
	}

	// Resync if it's been more than 1 hour since last sync
	return time.Since(ts.lastSyncTime) > 1*time.Hour
}
