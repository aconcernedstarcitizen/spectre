package main

import (
	"fmt"
	"strings"
	"time"
)

// ParseSaleTime parses user-friendly time formats into time.Time
// Supports the following formats (all assumed to be UTC):
//   - "2025-01-15 16:00"          (YYYY-MM-DD HH:MM)
//   - "2025-01-15T16:00:00Z"      (RFC3339)
//   - "2025-01-15 16:00 UTC"      (YYYY-MM-DD HH:MM UTC)
//   - "2025-01-15 16:00:00"       (YYYY-MM-DD HH:MM:SS)
func ParseSaleTime(timeStr string) (time.Time, error) {
	// Trim whitespace
	timeStr = strings.TrimSpace(timeStr)

	// Remove trailing "UTC" if present
	timeStr = strings.TrimSuffix(timeStr, " UTC")
	timeStr = strings.TrimSuffix(timeStr, "UTC")
	timeStr = strings.TrimSpace(timeStr)

	// Try RFC3339 format first (backward compatibility)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try friendly format: "2025-01-15 16:00"
	if t, err := time.Parse("2006-01-02 15:04", timeStr); err == nil {
		// Return time in UTC
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC), nil
	}

	// Try with seconds: "2025-01-15 16:00:00"
	if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	}

	// If none of the formats work, return helpful error
	return time.Time{}, fmt.Errorf("invalid time format '%s'. Use format: YYYY-MM-DD HH:MM (e.g., 2025-01-15 16:00). Time is assumed to be UTC", timeStr)
}
