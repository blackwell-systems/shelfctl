package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigFileExists_NoFile(t *testing.T) {
	// Point SHELFCTL_CONFIG at a non-existent file in a temp dir
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "nonexistent.yml")

	t.Setenv("SHELFCTL_CONFIG", fakePath)

	if configFileExists() {
		t.Error("configFileExists() = true, want false for missing file")
	}
}

func TestConfigFileExists_WithFile(t *testing.T) {
	// Create a real file and point SHELFCTL_CONFIG at it
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")

	if err := os.WriteFile(cfgPath, []byte("github:\n  owner: test\n"), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	t.Setenv("SHELFCTL_CONFIG", cfgPath)

	if !configFileExists() {
		t.Error("configFileExists() = false, want true for existing file")
	}
}

func TestPromptAndCreateConfig_NonTTY(t *testing.T) {
	// In test environments, stdin is not a TTY, so this should return (nil, nil)
	cfg, err := PromptAndCreateConfig()
	if err != nil {
		t.Fatalf("PromptAndCreateConfig() error = %v, want nil", err)
	}
	if cfg != nil {
		t.Errorf("PromptAndCreateConfig() config = %v, want nil for non-TTY", cfg)
	}
}
