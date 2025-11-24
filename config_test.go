package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check default values
	if config.BrowserType != "chrome" {
		t.Errorf("Expected BrowserType to be 'chrome', got '%s'", config.BrowserType)
	}

	if config.PageLoadTimeout != 30 {
		t.Errorf("Expected PageLoadTimeout to be 30, got %d", config.PageLoadTimeout)
	}

	if config.ViewportWidth != 1920 {
		t.Errorf("Expected ViewportWidth to be 1920, got %d", config.ViewportWidth)
	}

	if config.ViewportHeight != 1080 {
		t.Errorf("Expected ViewportHeight to be 1080, got %d", config.ViewportHeight)
	}

	if config.MaxRetries != 30 {
		t.Errorf("Expected MaxRetries to be 30, got %d", config.MaxRetries)
	}

	if config.AutoApplyCredit != true {
		t.Error("Expected AutoApplyCredit to be true")
	}

	if config.Headless != false {
		t.Error("Expected Headless to be false")
	}

	if config.KeepBrowserOpen != true {
		t.Error("Expected KeepBrowserOpen to be true")
	}

	// Check selectors are set
	if config.Selectors.AddToCartButton == "" {
		t.Error("Expected AddToCartButton selector to be set")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create a config with custom values
	config := DefaultConfig()
	config.ItemURL = "https://example.com/item"
	config.PageLoadTimeout = 60
	config.Headless = true
	config.MaxRetries = 10

	// Save the config
	if err := config.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check that the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the config back
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values match saved values
	if loadedConfig.ItemURL != config.ItemURL {
		t.Errorf("Expected ItemURL to be '%s', got '%s'", config.ItemURL, loadedConfig.ItemURL)
	}

	if loadedConfig.PageLoadTimeout != config.PageLoadTimeout {
		t.Errorf("Expected PageLoadTimeout to be %d, got %d", config.PageLoadTimeout, loadedConfig.PageLoadTimeout)
	}

	if loadedConfig.Headless != config.Headless {
		t.Errorf("Expected Headless to be %v, got %v", config.Headless, loadedConfig.Headless)
	}

	if loadedConfig.MaxRetries != config.MaxRetries {
		t.Errorf("Expected MaxRetries to be %d, got %d", config.MaxRetries, loadedConfig.MaxRetries)
	}
}

func TestLoadConfigCreatesDefaultIfMissing(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "new-config.yaml")

	// Load config from non-existent path
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig returned nil")
	}

	// Check that the file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created automatically")
	}

	// Verify it has default values
	if config.BrowserType != "chrome" {
		t.Errorf("Expected default BrowserType to be 'chrome', got '%s'", config.BrowserType)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	// Write invalid YAML
	invalidYAML := "invalid: yaml: content: [unclosed"
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	// Try to load the invalid config
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}
