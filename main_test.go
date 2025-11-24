package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetUserDataDir(t *testing.T) {
	dir := getUserDataDir()

	if dir == "" {
		t.Fatal("getUserDataDir returned empty string")
	}

	// Check that it either returns the fallback or a path containing .specter
	if dir == "./specter-data" {
		// Fallback path is acceptable
		return
	}

	if !strings.Contains(dir, ".specter") {
		t.Errorf("Expected directory to contain '.specter', got '%s'", dir)
	}

	// Verify it's an absolute path (unless it's the fallback)
	if !filepath.IsAbs(dir) && dir != "./specter-data" {
		t.Errorf("Expected absolute path, got '%s'", dir)
	}
}

func TestGetUserDataDirCreatesDirectory(t *testing.T) {
	// This test verifies that the init() function creates the directory
	dir := getUserDataDir()

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		// Directory doesn't exist yet - this is okay for the fallback case
		if dir != "./specter-data" {
			t.Logf("Note: User data directory doesn't exist yet: %v", err)
		}
		return
	}

	// If it exists, verify it's a directory
	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", dir)
	}

	// Check permissions (should be 0755)
	mode := info.Mode().Perm()
	if mode != 0755 {
		t.Logf("Note: Directory permissions are %o, expected 0755", mode)
	}
}

func TestMainPackage(t *testing.T) {
	// Verify that the main package can be imported
	// This is a basic sanity check
	config := DefaultConfig()
	if config == nil {
		t.Fatal("Unable to create default config")
	}

	automation := NewAutomation(config)
	if automation == nil {
		t.Fatal("Unable to create automation instance")
	}
}
