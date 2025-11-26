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

	if config.RetryDurationSeconds != 300 {
		t.Errorf("Expected RetryDurationSeconds to be 300, got %d", config.RetryDurationSeconds)
	}

	if config.RetryDelayMinMs != 5 {
		t.Errorf("Expected RetryDelayMinMs to be 5, got %d", config.RetryDelayMinMs)
	}

	if config.RetryDelayMaxMs != 20 {
		t.Errorf("Expected RetryDelayMaxMs to be 20, got %d", config.RetryDelayMaxMs)
	}

	if config.StartBeforeSaleMinutes != 10 {
		t.Errorf("Expected StartBeforeSaleMinutes to be 10, got %d", config.StartBeforeSaleMinutes)
	}

	if config.ContinueAfterSaleMinutes != 20 {
		t.Errorf("Expected ContinueAfterSaleMinutes to be 20, got %d", config.ContinueAfterSaleMinutes)
	}

	if config.EnableSaleTiming != false {
		t.Error("Expected EnableSaleTiming to be false by default")
	}

	if config.SaleStartTime != "" {
		t.Errorf("Expected SaleStartTime to be empty by default, got '%s'", config.SaleStartTime)
	}

	if config.RecaptchaSiteKey == "" {
		t.Error("Expected RecaptchaSiteKey to be set")
	}

	if config.RecaptchaAction != "store/cart/add" {
		t.Errorf("Expected RecaptchaAction to be 'store/cart/add', got '%s'", config.RecaptchaAction)
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

	if config.Selectors.AddToCartButton == "" {
		t.Error("Expected AddToCartButton selector to be set")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")

	config := DefaultConfig()
	config.ItemURL = "https://example.com/item"
	config.PageLoadTimeout = 60
	config.Headless = true
	config.RetryDurationSeconds = 600
	config.RetryDelayMinMs = 50
	config.RetryDelayMaxMs = 150

	if err := config.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedConfig.ItemURL != config.ItemURL {
		t.Errorf("Expected ItemURL to be '%s', got '%s'", config.ItemURL, loadedConfig.ItemURL)
	}

	if loadedConfig.PageLoadTimeout != config.PageLoadTimeout {
		t.Errorf("Expected PageLoadTimeout to be %d, got %d", config.PageLoadTimeout, loadedConfig.PageLoadTimeout)
	}

	if loadedConfig.Headless != config.Headless {
		t.Errorf("Expected Headless to be %v, got %v", config.Headless, loadedConfig.Headless)
	}

	if loadedConfig.RetryDurationSeconds != config.RetryDurationSeconds {
		t.Errorf("Expected RetryDurationSeconds to be %d, got %d", config.RetryDurationSeconds, loadedConfig.RetryDurationSeconds)
	}

	if loadedConfig.RetryDelayMinMs != config.RetryDelayMinMs {
		t.Errorf("Expected RetryDelayMinMs to be %d, got %d", config.RetryDelayMinMs, loadedConfig.RetryDelayMinMs)
	}

	if loadedConfig.RetryDelayMaxMs != config.RetryDelayMaxMs {
		t.Errorf("Expected RetryDelayMaxMs to be %d, got %d", config.RetryDelayMaxMs, loadedConfig.RetryDelayMaxMs)
	}
}

func TestLoadConfigCreatesDefaultIfMissing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "new-config.yaml")

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig returned nil")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created automatically")
	}

	if config.BrowserType != "chrome" {
		t.Errorf("Expected default BrowserType to be 'chrome', got '%s'", config.BrowserType)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "specter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	invalidYAML := "invalid: yaml: content: [unclosed"
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}
