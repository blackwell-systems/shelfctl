package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp config file
	tmpDir, err := os.MkdirTemp("", "test-config-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yml")
	validYAML := `github:
  owner: testowner
  token_env: TEST_TOKEN
  api_base: https://api.example.com
  backend: api
defaults:
  release: library
  cache_dir: /tmp/cache
  asset_naming: id
shelves:
  - name: testshelf
    repo: testrepo
    release: library
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set SHELFCTL_CONFIG to point to our test file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded values
	if cfg.GitHub.Owner != "testowner" {
		t.Errorf("GitHub.Owner = %q, want %q", cfg.GitHub.Owner, "testowner")
	}
	if cfg.GitHub.TokenEnv != "TEST_TOKEN" {
		t.Errorf("GitHub.TokenEnv = %q, want %q", cfg.GitHub.TokenEnv, "TEST_TOKEN")
	}
	if cfg.GitHub.APIBase != "https://api.example.com" {
		t.Errorf("GitHub.APIBase = %q, want %q", cfg.GitHub.APIBase, "https://api.example.com")
	}
	if cfg.GitHub.Backend != "api" {
		t.Errorf("GitHub.Backend = %q, want %q", cfg.GitHub.Backend, "api")
	}
	if cfg.Defaults.Release != "library" {
		t.Errorf("Defaults.Release = %q, want %q", cfg.Defaults.Release, "library")
	}
	if cfg.Defaults.AssetNaming != "id" {
		t.Errorf("Defaults.AssetNaming = %q, want %q", cfg.Defaults.AssetNaming, "id")
	}
	if len(cfg.Shelves) != 1 {
		t.Errorf("len(Shelves) = %d, want 1", len(cfg.Shelves))
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create temp config file with minimal fields
	tmpDir, err := os.MkdirTemp("", "test-config-defaults-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yml")
	minimalYAML := `github:
  owner: testowner
`
	if err := os.WriteFile(configPath, []byte(minimalYAML), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set SHELFCTL_CONFIG to point to our test file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults
	if cfg.GitHub.APIBase != "https://api.github.com" {
		t.Errorf("GitHub.APIBase = %q, want default %q", cfg.GitHub.APIBase, "https://api.github.com")
	}
	if cfg.GitHub.TokenEnv != "GITHUB_TOKEN" {
		t.Errorf("GitHub.TokenEnv = %q, want default %q", cfg.GitHub.TokenEnv, "GITHUB_TOKEN")
	}
	if cfg.GitHub.Backend != "api" {
		t.Errorf("GitHub.Backend = %q, want default %q", cfg.GitHub.Backend, "api")
	}
	if cfg.Defaults.Release != "library" {
		t.Errorf("Defaults.Release = %q, want default %q", cfg.Defaults.Release, "library")
	}
	if cfg.Defaults.AssetNaming != "id" {
		t.Errorf("Defaults.AssetNaming = %q, want default %q", cfg.Defaults.AssetNaming, "id")
	}
	// CacheDir default is dynamic (depends on home dir), just check it's not empty
	if cfg.Defaults.CacheDir == "" {
		t.Error("Defaults.CacheDir is empty, expected default value")
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	// Create temp config file with malformed YAML
	tmpDir, err := os.MkdirTemp("", "test-config-invalid-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yml")
	invalidYAML := `github:
  owner: testowner
  token_env: [unclosed bracket
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set SHELFCTL_CONFIG to point to our test file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Load config - expect error
	_, err = Load()
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML, got nil")
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	// Point to non-existent config file
	tmpDir, err := os.MkdirTemp("", "test-config-notfound-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "nonexistent.yml")

	// Set SHELFCTL_CONFIG to point to non-existent file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Load config - should NOT error (returns empty config)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error for missing config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Should have default values
	if cfg.GitHub.APIBase != "https://api.github.com" {
		t.Errorf("GitHub.APIBase = %q, want default %q", cfg.GitHub.APIBase, "https://api.github.com")
	}
}

func TestLoadConfigValidation(t *testing.T) {
	// Test schema validation - this is a basic test since the Load function
	// currently doesn't enforce required fields, but we can test unmarshal behavior
	tmpDir, err := os.MkdirTemp("", "test-config-validation-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yml")
	// Empty YAML - should load with all defaults
	emptyYAML := ``
	if err := os.WriteFile(configPath, []byte(emptyYAML), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set SHELFCTL_CONFIG to point to our test file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// All defaults should be set
	if cfg.GitHub.APIBase != "https://api.github.com" {
		t.Errorf("GitHub.APIBase = %q, want default", cfg.GitHub.APIBase)
	}
	if cfg.GitHub.TokenEnv != "GITHUB_TOKEN" {
		t.Errorf("GitHub.TokenEnv = %q, want default", cfg.GitHub.TokenEnv)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	// Create temp config file
	tmpDir, err := os.MkdirTemp("", "test-config-env-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yml")
	configYAML := `github:
  owner: fileowner
  api_base: https://api.file.com
defaults:
  release: library
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set SHELFCTL_CONFIG to point to our test file
	oldConfig := os.Getenv("SHELFCTL_CONFIG")
	if err := os.Setenv("SHELFCTL_CONFIG", configPath); err != nil {
		t.Fatalf("failed to set SHELFCTL_CONFIG: %v", err)
	}
	defer func() {
		if err := os.Setenv("SHELFCTL_CONFIG", oldConfig); err != nil {
			t.Errorf("failed to restore SHELFCTL_CONFIG: %v", err)
		}
	}()

	// Set environment variables to override config file values
	oldOwner := os.Getenv("SHELFCTL_GITHUB_OWNER")
	oldAPIBase := os.Getenv("SHELFCTL_GITHUB_API_BASE")
	oldRelease := os.Getenv("SHELFCTL_DEFAULTS_RELEASE")

	if err := os.Setenv("SHELFCTL_GITHUB_OWNER", "envowner"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	if err := os.Setenv("SHELFCTL_GITHUB_API_BASE", "https://api.env.com"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	if err := os.Setenv("SHELFCTL_DEFAULTS_RELEASE", "snapshot"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}

	defer func() {
		if err := os.Setenv("SHELFCTL_GITHUB_OWNER", oldOwner); err != nil {
			t.Errorf("failed to restore env: %v", err)
		}
		if err := os.Setenv("SHELFCTL_GITHUB_API_BASE", oldAPIBase); err != nil {
			t.Errorf("failed to restore env: %v", err)
		}
		if err := os.Setenv("SHELFCTL_DEFAULTS_RELEASE", oldRelease); err != nil {
			t.Errorf("failed to restore env: %v", err)
		}
	}()

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify env variables override file values
	if cfg.GitHub.Owner != "envowner" {
		t.Errorf("GitHub.Owner = %q, want env override %q", cfg.GitHub.Owner, "envowner")
	}
	if cfg.GitHub.APIBase != "https://api.env.com" {
		t.Errorf("GitHub.APIBase = %q, want env override %q", cfg.GitHub.APIBase, "https://api.env.com")
	}
	if cfg.Defaults.Release != "snapshot" {
		t.Errorf("Defaults.Release = %q, want env override %q", cfg.Defaults.Release, "snapshot")
	}
}
