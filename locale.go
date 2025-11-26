package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Locale struct {
	translations map[string]string
	locale       string
}

var globalLocale *Locale

// InitLocale initializes the global locale system
func InitLocale() error {
	locale := DetectSystemLocale()

	l, err := LoadLocale(locale)
	if err != nil {
		// Fallback to English
		fmt.Printf("Warning: Failed to load locale '%s', falling back to en_US: %v\n", locale, err)
		l, err = LoadLocale("en_US")
		if err != nil {
			return fmt.Errorf("failed to load fallback locale en_US: %w", err)
		}
	}

	globalLocale = l
	return nil
}

// DetectSystemLocale detects the user's system locale
func DetectSystemLocale() string {
	// Try environment variables first (works on both Mac and Linux)
	if locale := os.Getenv("LANG"); locale != "" {
		// LANG is typically like "en_US.UTF-8" or "ru_RU.UTF-8"
		parts := strings.Split(locale, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	if locale := os.Getenv("LC_ALL"); locale != "" {
		parts := strings.Split(locale, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	if locale := os.Getenv("LC_MESSAGES"); locale != "" {
		parts := strings.Split(locale, ".")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Windows-specific environment variable
	if runtime.GOOS == "windows" {
		// Try to get Windows locale from environment
		// Windows uses different locale codes, we'll map common ones
		if locale := os.Getenv("LANG"); locale != "" {
			return locale
		}
	}

	// Default fallback
	return "en_US"
}

// LoadLocale loads a locale file from the lang/ directory
func LoadLocale(locale string) (*Locale, error) {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Try loading from lang/ directory next to executable
	localeFile := filepath.Join(exeDir, "lang", locale+".yaml")

	data, err := os.ReadFile(localeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read locale file %s: %w", localeFile, err)
	}

	var translations map[string]string
	if err := yaml.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("failed to parse locale file %s: %w", localeFile, err)
	}

	return &Locale{
		translations: translations,
		locale:       locale,
	}, nil
}

// T translates a key with optional parameters
// Usage: T("greeting", "name", "John") => "Hello, John!"
func T(key string, params ...interface{}) string {
	if globalLocale == nil {
		return key
	}

	translation, ok := globalLocale.translations[key]
	if !ok {
		// Return key if translation not found
		return key
	}

	// Simple parameter substitution
	if len(params) > 0 {
		// Build args for fmt.Sprintf style
		return fmt.Sprintf(translation, params...)
	}

	return translation
}

// GetLocale returns the current locale code (e.g., "en_US", "ru_RU")
func GetLocale() string {
	if globalLocale == nil {
		return "en_US"
	}
	return globalLocale.locale
}
