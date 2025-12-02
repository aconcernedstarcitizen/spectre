package main

import (
	"testing"
	"time"
)

func TestTimeSync(t *testing.T) {
	ts := NewTimeSync(false)

	if ts.IsSynced() {
		t.Error("TimeSync should not be synced initially")
	}

	err := ts.Sync()
	if err != nil {
		t.Fatalf("Failed to sync time: %v", err)
	}

	if !ts.IsSynced() {
		t.Error("TimeSync should be synced after calling Sync()")
	}

	// Check that offset is reasonable (within 10 seconds)
	offset := ts.GetOffset()
	if offset > 10*time.Second || offset < -10*time.Second {
		t.Errorf("Time offset seems unreasonable: %v", offset)
	}

	// Check that Now() returns a time close to system time
	syncedTime := ts.Now()
	systemTime := time.Now()
	diff := syncedTime.Sub(systemTime)

	// Allow up to 2 seconds difference (accounting for offset + test execution time)
	if diff > 2*time.Second || diff < -2*time.Second {
		t.Errorf("Synced time differs too much from system time: %v", diff)
	}
}

func TestTimeSyncResync(t *testing.T) {
	ts := NewTimeSync(false)

	if !ts.ShouldResync() {
		t.Error("Should need to resync when not yet synced")
	}

	err := ts.Sync()
	if err != nil {
		t.Fatalf("Failed to sync time: %v", err)
	}

	if ts.ShouldResync() {
		t.Error("Should not need to resync immediately after syncing")
	}

	// Simulate time passing by directly modifying lastSyncTime
	ts.lastSyncTime = time.Now().Add(-2 * time.Hour)

	if !ts.ShouldResync() {
		t.Error("Should need to resync after 2 hours")
	}
}

func TestTimeSyncDebugMode(t *testing.T) {
	ts := NewTimeSync(true) // Enable debug mode

	err := ts.Sync()
	if err != nil {
		t.Fatalf("Failed to sync time in debug mode: %v", err)
	}

	if !ts.IsSynced() {
		t.Error("TimeSync should be synced after calling Sync()")
	}
}

func TestTimeSyncBeforeSync(t *testing.T) {
	ts := NewTimeSync(false)

	// Before syncing, Now() should return approximately system time
	syncedTime := ts.Now()
	systemTime := time.Now()
	diff := syncedTime.Sub(systemTime)

	// Should be very close (within 100ms) since it's just returning time.Now()
	if diff > 100*time.Millisecond || diff < -100*time.Millisecond {
		t.Errorf("Unsynced time differs from system time: %v", diff)
	}
}
