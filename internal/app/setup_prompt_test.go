package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPromptAndCreateConfig_SuccessDefaultToken(t *testing.T) {
	// Redirect HOME so config.Save writes to temp dir
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Clear SHELFCTL_CONFIG so DefaultPath() is used
	t.Setenv("SHELFCTL_CONFIG", "")

	input := strings.NewReader("testowner\n\n")
	cfg, err := promptAndCreateConfigFromReader(input)
	if err != nil {
		t.Fatalf("promptAndCreateConfigFromReader() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("promptAndCreateConfigFromReader() returned nil config")
	}
	if cfg.GitHub.Owner != "testowner" {
		t.Errorf("owner = %q, want %q", cfg.GitHub.Owner, "testowner")
	}
	if cfg.GitHub.TokenEnv != "SHELFCTL_GITHUB_TOKEN" {
		t.Errorf("token_env = %q, want %q", cfg.GitHub.TokenEnv, "SHELFCTL_GITHUB_TOKEN")
	}

	// Verify the file was written to disk
	configPath := filepath.Join(tmpHome, ".config", "shelfctl", "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file not written at %s", configPath)
	}
}

func TestPromptAndCreateConfig_CustomTokenEnv(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELFCTL_CONFIG", "")

	input := strings.NewReader("testowner\nMY_CUSTOM_TOKEN\n")
	cfg, err := promptAndCreateConfigFromReader(input)
	if err != nil {
		t.Fatalf("promptAndCreateConfigFromReader() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("promptAndCreateConfigFromReader() returned nil config")
	}
	if cfg.GitHub.Owner != "testowner" {
		t.Errorf("owner = %q, want %q", cfg.GitHub.Owner, "testowner")
	}
	if cfg.GitHub.TokenEnv != "MY_CUSTOM_TOKEN" {
		t.Errorf("token_env = %q, want %q", cfg.GitHub.TokenEnv, "MY_CUSTOM_TOKEN")
	}
}

func TestPromptAndCreateConfig_EmptyOwnerRepromptsUntilProvided(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELFCTL_CONFIG", "")

	// First line is empty (just enter), second line provides the owner
	input := strings.NewReader("\nactualowner\n\n")
	cfg, err := promptAndCreateConfigFromReader(input)
	if err != nil {
		t.Fatalf("promptAndCreateConfigFromReader() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("promptAndCreateConfigFromReader() returned nil config")
	}
	if cfg.GitHub.Owner != "actualowner" {
		t.Errorf("owner = %q, want %q", cfg.GitHub.Owner, "actualowner")
	}
}

func TestPromptAndCreateConfig_EmptyOwnerEOF(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("SHELFCTL_CONFIG", "")

	// Only empty lines then EOF — should error because owner is never provided
	input := strings.NewReader("\n")
	_, err := promptAndCreateConfigFromReader(input)
	if err == nil {
		t.Fatal("expected error when owner is empty and input ends, got nil")
	}
	if !strings.Contains(err.Error(), "reading input") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "reading input")
	}
}
