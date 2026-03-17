package operations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
)

// TestAddShelfToConfig_LoadError tests BUG 13 fix:
// if config.Load returns an error, confirm addShelfToConfig propagates it.
func TestAddShelfToConfig_LoadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shelfctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write an invalid YAML file to a temp path
	configPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(configPath, []byte("this is not: valid: yaml: [unclosed"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Point shelfctl at the temp config so the real config is never touched
	oldEnv := os.Getenv("SHELFCTL_CONFIG")
	_ = os.Setenv("SHELFCTL_CONFIG", configPath)
	defer func() {
		if oldEnv == "" {
			_ = os.Unsetenv("SHELFCTL_CONFIG")
		} else {
			_ = os.Setenv("SHELFCTL_CONFIG", oldEnv)
		}
	}()

	cfg := &config.Config{
		GitHub: config.GitHubConfig{Owner: "testowner"},
	}

	err = addShelfToConfig(cfg, "testshelf", "testowner", "testrepo")
	if err == nil {
		t.Error("Expected error when config.Load() fails, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "loading config") {
		t.Errorf("Expected error to mention 'loading config', got: %v", err)
	}
}
