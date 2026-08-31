package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config failed validation: %v", err)
	}
}

func TestSaveAndLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "engagement.json")

	cfg := DefaultConfig()
	cfg.Name = "test-engagement"

	if err := cfg.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("config file should exist after save")
	}

	loaded, err := LoadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "test-engagement" {
		t.Fatalf("expected test-engagement, got %s", loaded.Name)
	}
	if len(loaded.Listeners) != 2 {
		t.Fatalf("expected 2 listeners, got %d", len(loaded.Listeners))
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
}

func TestValidationMissingName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Name = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestValidationMissingListenerAddr(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Listeners[0].Addr = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for missing listener addr")
	}
}

func TestValidationBadJitter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Profiles[0].JitterPercent = 150
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for jitter > 100")
	}
}

func TestLoadNonExistent(t *testing.T) {
	if _, err := LoadJSON("/no/such/path/config.json"); err == nil {
		t.Fatal("expected error loading non-existent config")
	}
}
