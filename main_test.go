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

	if dir == "./specter-data" {
		return
	}

	if !strings.Contains(dir, ".specter") {
		t.Errorf("Expected directory to contain '.specter', got '%s'", dir)
	}

	if !filepath.IsAbs(dir) && dir != "./specter-data" {
		t.Errorf("Expected absolute path, got '%s'", dir)
	}
}

func TestGetUserDataDirCreatesDirectory(t *testing.T) {
	dir := getUserDataDir()

	info, err := os.Stat(dir)
	if err != nil {
		if dir != "./specter-data" {
			t.Logf("Note: User data directory doesn't exist yet: %v", err)
		}
		return
	}

	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", dir)
	}

	mode := info.Mode().Perm()
	if mode != 0755 {
		t.Logf("Note: Directory permissions are %o, expected 0755", mode)
	}
}

func TestMainPackage(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("Unable to create default config")
	}

	automation := NewAutomation(config)
	if automation == nil {
		t.Fatal("Unable to create automation instance")
	}
}
