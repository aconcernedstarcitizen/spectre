package main

import (
	"testing"
	"time"
)

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"WORLD", "world"},
		{"MiXeD CaSe", "mixed case"},
		{"123ABC", "123abc"},
		{"", ""},
		{"already lowercase", "already lowercase"},
	}

	for _, test := range tests {
		result := toLower(test.input)
		if result != test.expected {
			t.Errorf("toLower(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"Hello World", []string{"world"}, true},
		{"Hello World", []string{"WORLD"}, true},
		{"Hello World", []string{"foo"}, false},
		{"Hello World", []string{"foo", "world"}, true},
		{"Hello World", []string{"foo", "bar"}, false},
		{"", []string{"test"}, false},
		{"test", []string{""}, true},
		{"GoLang", []string{"go"}, true},
		{"GoLang", []string{"GO"}, true},
	}

	for _, test := range tests {
		result := contains(test.s, test.substrs...)
		if result != test.expected {
			t.Errorf("contains(%q, %v) = %v, expected %v", test.s, test.substrs, result, test.expected)
		}
	}
}

func TestNewAutomation(t *testing.T) {
	config := DefaultConfig()
	automation := NewAutomation(config)

	if automation == nil {
		t.Fatal("NewAutomation returned nil")
	}

	if automation.config != config {
		t.Error("Automation config does not match provided config")
	}

	if automation.rand == nil {
		t.Error("Random number generator not initialized")
	}

	if automation.stopChan == nil {
		t.Error("Stop channel not initialized")
	}

	if automation.itemInCart != false {
		t.Error("itemInCart should be false initially")
	}
}

func TestGetTimeout(t *testing.T) {
	config := DefaultConfig()
	automation := NewAutomation(config)

	// Test that timeout is within expected range (200-380ms)
	for i := 0; i < 100; i++ {
		timeout := automation.getTimeout()
		if timeout < 200*time.Millisecond || timeout > 380*time.Millisecond {
			t.Errorf("getTimeout() returned %v, expected between 200ms and 380ms", timeout)
		}
	}
}

func TestGetClickTimeout(t *testing.T) {
	config := DefaultConfig()
	automation := NewAutomation(config)

	// Test that click timeout is within expected range (700-1100ms)
	for i := 0; i < 100; i++ {
		timeout := automation.getClickTimeout()
		if timeout < 700*time.Millisecond || timeout > 1100*time.Millisecond {
			t.Errorf("getClickTimeout() returned %v, expected between 700ms and 1100ms", timeout)
		}
	}
}

func TestCheckIfTotalIsZero(t *testing.T) {
	// This test would require a browser instance, so we'll skip it
	// In a real-world scenario, you'd use a mock or test page
	t.Skip("Skipping browser-dependent test")
}

func TestRandomDelay(t *testing.T) {
	config := DefaultConfig()
	config.MinDelayBetween = 0.1
	config.MaxDelayBetween = 0.2
	automation := NewAutomation(config)

	start := time.Now()
	automation.randomDelay()
	elapsed := time.Since(start)

	// Check that delay is within expected range (100-200ms)
	if elapsed < 100*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Errorf("randomDelay() took %v, expected between 100ms and 300ms", elapsed)
	}
}

func TestDebugLog(t *testing.T) {
	config := DefaultConfig()
	automation := NewAutomation(config)

	// This should not panic
	automation.debugLog("Test message: %s", "test")

	// Enable debug mode
	config.DebugMode = true
	automation.debugLog("Debug enabled: %d", 42)
}

func TestIsBrowserAlive(t *testing.T) {
	config := DefaultConfig()
	automation := NewAutomation(config)

	// Without a browser, should return false
	if automation.isBrowserAlive() {
		t.Error("isBrowserAlive() should return false when browser is nil")
	}
}
