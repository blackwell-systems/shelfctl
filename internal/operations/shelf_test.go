package operations

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
)

// TestAddShelfToConfig_LoadError tests BUG 13 fix:
// if config.Load returns an error, confirm addShelfToConfig propagates it.
func TestAddShelfToConfig_LoadError(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "shelfctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an invalid config file (not valid YAML)
	configPath := filepath.Join(tmpDir, "config.yml")
	invalidYAML := "this is not: valid: yaml: content: [unclosed"
	if err := ioutil.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Set the config path environment variable
	oldEnv := os.Getenv("SHELFCTL_CONFIG")
	os.Setenv("SHELFCTL_CONFIG", configPath)
	defer func() {
		if oldEnv == "" {
			os.Unsetenv("SHELFCTL_CONFIG")
		} else {
			os.Setenv("SHELFCTL_CONFIG", oldEnv)
		}
	}()

	// Create a valid config object to pass to addShelfToConfig
	// (this is separate from the invalid file we created)
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "testowner",
		},
	}

	// Attempt to add a shelf - this should fail because config.Load() will fail
	err = addShelfToConfig(cfg, "testshelf", "testowner", "testrepo")

	// Verify that an error is returned (not silently swallowed)
	if err == nil {
		t.Error("Expected an error from addShelfToConfig when config.Load() fails, but got nil")
	}

	// Verify the error message mentions "loading config"
	if err != nil && !contains(err.Error(), "loading config") {
		t.Errorf("Expected error to mention 'loading config', got: %v", err)
	}
}

// TestAddShelfToConfig_Success tests that addShelfToConfig works correctly
// when config.Load() succeeds and preserves existing shelves.
// Note: config.Save() saves to DefaultPath(), not the test file path.
// This test verifies that the BUG 13 fix (returning error instead of silently
// dropping shelves) works correctly when config can be loaded.
func TestAddShelfToConfig_Success(t *testing.T) {
	// For this test, we need to work with the default config path
	// We'll temporarily move any existing config and restore it afterward
	
	defaultPath := filepath.Join(os.Getenv("HOME"), ".config", "shelfctl", "config.yml")
	backupPath := defaultPath + ".test-backup"
	
	// Backup existing config if it exists
	if _, err := os.Stat(defaultPath); err == nil {
		data, _ := ioutil.ReadFile(defaultPath)
		ioutil.WriteFile(backupPath, data, 0644)
		defer func() {
			// Restore original config
			data, _ := ioutil.ReadFile(backupPath)
			ioutil.WriteFile(defaultPath, data, 0644)
			os.Remove(backupPath)
		}()
	} else {
		// No existing config, clean up after test
		defer os.Remove(defaultPath)
	}

	// Create a test config file at the default location
	initialConfig := `github:
  owner: existingowner
  token_env: GITHUB_TOKEN
  api_base: https://api.github.com
defaults:
  release: library
  asset_naming: id
shelves:
  - name: existingshelf
    owner: existingowner
    repo: existingrepo
`
	if err := os.MkdirAll(filepath.Dir(defaultPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := ioutil.WriteFile(defaultPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Clear any environment override for config path
	oldEnv := os.Getenv("SHELFCTL_CONFIG")
	os.Unsetenv("SHELFCTL_CONFIG")
	defer func() {
		if oldEnv != "" {
			os.Setenv("SHELFCTL_CONFIG", oldEnv)
		}
	}()

	// Create a config object to pass to addShelfToConfig
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "testowner",
		},
	}

	// Load initial config to verify it has one shelf
	initialCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load initial config: %v", err)
	}
	if len(initialCfg.Shelves) != 1 {
		t.Fatalf("Expected initial config to have 1 shelf, got %d", len(initialCfg.Shelves))
	}

	// Add a new shelf
	err = addShelfToConfig(cfg, "newshelf", "testowner", "newrepo")
	if err != nil {
		t.Errorf("Expected addShelfToConfig to succeed, got error: %v", err)
	}

	// Verify the config was updated correctly
	updatedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}

	// Check that both shelves exist
	if len(updatedCfg.Shelves) != 2 {
		t.Errorf("Expected 2 shelves, got %d", len(updatedCfg.Shelves))
		for i, s := range updatedCfg.Shelves {
			t.Logf("Shelf %d: name=%s owner=%s repo=%s", i, s.Name, s.Owner, s.Repo)
		}
	}

	// Verify the existing shelf is still there
	foundExisting := false
	foundNew := false
	for _, s := range updatedCfg.Shelves {
		if s.Name == "existingshelf" && s.Owner == "existingowner" && s.Repo == "existingrepo" {
			foundExisting = true
		}
		if s.Name == "newshelf" && s.Owner == "testowner" && s.Repo == "newrepo" {
			foundNew = true
		}
	}

	if !foundExisting {
		t.Error("Existing shelf was lost after adding new shelf")
	}

	if !foundNew {
		t.Error("New shelf was not added")
	}
}

// TestAddShelfToConfig_NoDuplicates tests that adding a shelf with the same name
// doesn't create duplicates.
func TestAddShelfToConfig_NoDuplicates(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := ioutil.TempDir("", "shelfctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid initial config file
	configPath := filepath.Join(tmpDir, "config.yml")
	initialConfig := `github:
  owner: testowner
  token_env: GITHUB_TOKEN
  api_base: https://api.github.com
defaults:
  release: library
  asset_naming: id
shelves:
  - name: testshelf
    owner: testowner
    repo: testrepo
`
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := ioutil.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Set the config path environment variable
	oldEnv := os.Getenv("SHELFCTL_CONFIG")
	os.Setenv("SHELFCTL_CONFIG", configPath)
	defer func() {
		if oldEnv == "" {
			os.Unsetenv("SHELFCTL_CONFIG")
		} else {
			os.Setenv("SHELFCTL_CONFIG", oldEnv)
		}
	}()

	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "testowner",
		},
	}

	// Try to add the same shelf again
	err = addShelfToConfig(cfg, "testshelf", "testowner", "testrepo")
	if err != nil {
		t.Errorf("Expected addShelfToConfig to succeed (skip duplicate), got error: %v", err)
	}

	// Verify only one shelf exists
	updatedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}

	if len(updatedCfg.Shelves) != 1 {
		t.Errorf("Expected 1 shelf (no duplicate), got %d", len(updatedCfg.Shelves))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
