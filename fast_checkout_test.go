package main

import (
	"fmt"
	"testing"
	"time"
)

// Test error detection functions
func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Rate limit error 4227",
			err:      fmt.Errorf("GraphQL error: code 4227 - rate limited"),
			expected: true,
		},
		{
			name:     "Rate limit text",
			err:      fmt.Errorf("rate limited"),
			expected: true,
		},
		{
			name:     "Rate-limit text",
			err:      fmt.Errorf("rate-limit exceeded"),
			expected: true,
		},
		{
			name:     "Other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("isRateLimitError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

func TestIsOutOfStockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Out of stock error 4226",
			err:      fmt.Errorf("GraphQL error: code 4226 - out of stock"),
			expected: true,
		},
		{
			name:     "Out of stock text",
			err:      fmt.Errorf("out of stock"),
			expected: true,
		},
		{
			name:     "Not available text",
			err:      fmt.Errorf("item not available"),
			expected: true,
		},
		{
			name:     "Unavailable text",
			err:      fmt.Errorf("currently unavailable"),
			expected: true,
		},
		{
			name:     "Other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOutOfStockError(tt.err)
			if result != tt.expected {
				t.Errorf("isOutOfStockError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

func TestIsCaptchaError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "CAPTCHA lowercase",
			err:      fmt.Errorf("captcha verification failed"),
			expected: true,
		},
		{
			name:     "reCAPTCHA text",
			err:      fmt.Errorf("reCAPTCHA challenge failed"),
			expected: true,
		},
		{
			name:     "CAPTCHA uppercase",
			err:      fmt.Errorf("CAPTCHA required"),
			expected: true,
		},
		{
			name:     "Other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCaptchaError(tt.err)
			if result != tt.expected {
				t.Errorf("isCaptchaError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

// Test GraphQL request structure
func TestGraphQLRequestStructure(t *testing.T) {
	request := GraphQLRequest{
		OperationName: "TestOperation",
		Variables: map[string]interface{}{
			"test": "value",
		},
		Query: "query TestQuery { test }",
	}

	if request.OperationName != "TestOperation" {
		t.Errorf("Expected OperationName 'TestOperation', got '%s'", request.OperationName)
	}

	if request.Query != "query TestQuery { test }" {
		t.Errorf("Expected Query 'query TestQuery { test }', got '%s'", request.Query)
	}

	if request.Variables["test"] != "value" {
		t.Errorf("Expected Variables['test'] to be 'value', got '%v'", request.Variables["test"])
	}
}

// Test timed sale mode timing calculations
func TestTimedSaleModeTimingCalculations(t *testing.T) {
	// Test sale time parsing
	saleTimeStr := "2025-01-15T18:00:00Z"
	saleTime, err := time.Parse(time.RFC3339, saleTimeStr)
	if err != nil {
		t.Fatalf("Failed to parse sale time: %v", err)
	}

	// Test timing calculations
	startBeforeMinutes := 10
	continueAfterMinutes := 20

	startRetryTime := saleTime.Add(-time.Duration(startBeforeMinutes) * time.Minute)
	endRetryTime := saleTime.Add(time.Duration(continueAfterMinutes) * time.Minute)

	expectedStartTime := time.Date(2025, 1, 15, 17, 50, 0, 0, time.UTC)
	expectedEndTime := time.Date(2025, 1, 15, 18, 20, 0, 0, time.UTC)

	if !startRetryTime.Equal(expectedStartTime) {
		t.Errorf("Expected start retry time %v, got %v", expectedStartTime, startRetryTime)
	}

	if !endRetryTime.Equal(expectedEndTime) {
		t.Errorf("Expected end retry time %v, got %v", expectedEndTime, endRetryTime)
	}

	// Test total window duration
	totalWindow := endRetryTime.Sub(startRetryTime)
	expectedWindow := 30 * time.Minute

	if totalWindow != expectedWindow {
		t.Errorf("Expected total window %v, got %v", expectedWindow, totalWindow)
	}
}

// Test retry delay calculations
func TestRetryDelayCalculations(t *testing.T) {
	config := DefaultConfig()

	// Test that default delays are within expected ultra-fast range
	if config.RetryDelayMinMs < 1 || config.RetryDelayMinMs > 10 {
		t.Errorf("Expected RetryDelayMinMs to be 1-10ms for ultra-fast retries, got %d", config.RetryDelayMinMs)
	}

	if config.RetryDelayMaxMs < 10 || config.RetryDelayMaxMs > 50 {
		t.Errorf("Expected RetryDelayMaxMs to be 10-50ms for ultra-fast retries, got %d", config.RetryDelayMaxMs)
	}

	// Verify min is less than max
	if config.RetryDelayMinMs >= config.RetryDelayMaxMs {
		t.Errorf("RetryDelayMinMs (%d) should be less than RetryDelayMaxMs (%d)",
			config.RetryDelayMinMs, config.RetryDelayMaxMs)
	}
}

// Test FastCheckout initialization
func TestNewFastCheckout(t *testing.T) {
	config := DefaultConfig()
	config.RecaptchaSiteKey = "test-site-key"

	fc, err := NewFastCheckout(config)
	if err != nil {
		t.Fatalf("NewFastCheckout failed: %v", err)
	}

	if fc == nil {
		t.Fatal("NewFastCheckout returned nil")
	}

	if fc.config != config {
		t.Error("FastCheckout config not set correctly")
	}

	if fc.client == nil {
		t.Error("FastCheckout HTTP client not initialized")
	}

	if fc.baseURL != "https://robertsspaceindustries.com" {
		t.Errorf("Expected baseURL 'https://robertsspaceindustries.com', got '%s'", fc.baseURL)
	}

	if fc.graphqlURL != "https://robertsspaceindustries.com/graphql" {
		t.Errorf("Expected graphqlURL 'https://robertsspaceindustries.com/graphql', got '%s'", fc.graphqlURL)
	}
}

// Test sale timing configuration validation
func TestSaleTimingConfiguration(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	if config.EnableSaleTiming != false {
		t.Error("Expected EnableSaleTiming to be false by default")
	}

	if config.SaleStartTime != "" {
		t.Errorf("Expected SaleStartTime to be empty by default, got '%s'", config.SaleStartTime)
	}

	if config.StartBeforeSaleMinutes != 10 {
		t.Errorf("Expected StartBeforeSaleMinutes to be 10, got %d", config.StartBeforeSaleMinutes)
	}

	if config.ContinueAfterSaleMinutes != 20 {
		t.Errorf("Expected ContinueAfterSaleMinutes to be 20, got %d", config.ContinueAfterSaleMinutes)
	}

	// Test that we can set sale timing
	config.EnableSaleTiming = true
	config.SaleStartTime = "2025-01-15T18:00:00Z"
	config.StartBeforeSaleMinutes = 15
	config.ContinueAfterSaleMinutes = 30

	if !config.EnableSaleTiming {
		t.Error("Failed to enable sale timing")
	}

	if config.SaleStartTime != "2025-01-15T18:00:00Z" {
		t.Errorf("Expected SaleStartTime '2025-01-15T18:00:00Z', got '%s'", config.SaleStartTime)
	}

	if config.StartBeforeSaleMinutes != 15 {
		t.Errorf("Expected StartBeforeSaleMinutes 15, got %d", config.StartBeforeSaleMinutes)
	}

	if config.ContinueAfterSaleMinutes != 30 {
		t.Errorf("Expected ContinueAfterSaleMinutes 30, got %d", config.ContinueAfterSaleMinutes)
	}
}

// Test address caching
func TestAddressCaching(t *testing.T) {
	config := DefaultConfig()
	fc, err := NewFastCheckout(config)
	if err != nil {
		t.Fatalf("NewFastCheckout failed: %v", err)
	}

	// Initially, cached address should be empty
	if fc.cachedAddressID != "" {
		t.Errorf("Expected cachedAddressID to be empty initially, got '%s'", fc.cachedAddressID)
	}

	// Simulate caching an address
	testAddressID := "test-address-123"
	fc.cachedAddressID = testAddressID

	if fc.cachedAddressID != testAddressID {
		t.Errorf("Expected cachedAddressID '%s', got '%s'", testAddressID, fc.cachedAddressID)
	}
}

// Test that optimized retry delays are significantly faster than old values
func TestOptimizedRetryDelays(t *testing.T) {
	config := DefaultConfig()

	// Old values were: 29-107ms
	// New values are: 5-20ms
	// Verify new values are at least 80% faster at minimum
	oldMin := 29
	oldMax := 107

	if config.RetryDelayMinMs >= oldMin {
		t.Errorf("RetryDelayMinMs (%d) should be less than old value (%d)",
			config.RetryDelayMinMs, oldMin)
	}

	if config.RetryDelayMaxMs >= oldMax {
		t.Errorf("RetryDelayMaxMs (%d) should be less than old value (%d)",
			config.RetryDelayMaxMs, oldMax)
	}

	// Calculate speed improvement
	avgOld := float64(oldMin+oldMax) / 2.0
	avgNew := float64(config.RetryDelayMinMs+config.RetryDelayMaxMs) / 2.0
	improvement := ((avgOld - avgNew) / avgOld) * 100

	if improvement < 80 {
		t.Errorf("Expected at least 80%% improvement in retry delays, got %.1f%%", improvement)
	}

	t.Logf("Retry delay improvement: %.1f%% (old avg: %.1fms, new avg: %.1fms)",
		improvement, avgOld, avgNew)
}

// Test reCAPTCHA configuration
func TestRecaptchaConfiguration(t *testing.T) {
	config := DefaultConfig()

	if config.RecaptchaSiteKey == "" {
		t.Error("Expected RecaptchaSiteKey to be set by default")
	}

	expectedSiteKey := "6LcZ-cUpAAAAABTy47-ryVJAsZFocXguqi_FgLlJ"
	if config.RecaptchaSiteKey != expectedSiteKey {
		t.Errorf("Expected RecaptchaSiteKey '%s', got '%s'",
			expectedSiteKey, config.RecaptchaSiteKey)
	}

	if config.RecaptchaAction != "store/cart/add" {
		t.Errorf("Expected RecaptchaAction 'store/cart/add', got '%s'",
			config.RecaptchaAction)
	}
}

// Test viewport configuration for desktop
func TestViewportConfiguration(t *testing.T) {
	config := DefaultConfig()

	// Should be desktop resolution by default
	if config.ViewportWidth != 1920 {
		t.Errorf("Expected ViewportWidth 1920, got %d", config.ViewportWidth)
	}

	if config.ViewportHeight != 1080 {
		t.Errorf("Expected ViewportHeight 1080, got %d", config.ViewportHeight)
	}

	// Verify it's a reasonable desktop resolution
	if config.ViewportWidth < 1024 || config.ViewportHeight < 768 {
		t.Error("Viewport dimensions should be at least 1024x768 for desktop")
	}
}
