package app

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
)

// TestMoveBook tests destination determination logic for cross-shelf move
func TestMoveBook(t *testing.T) {
	// Setup config
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v1.0",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:           "shelf-a",
				Repo:           "repo-a",
				Owner:          "test-owner",
				DefaultRelease: "v1.0",
			},
			{
				Name:           "shelf-b",
				Repo:           "repo-b",
				Owner:          "test-owner",
				DefaultRelease: "v2.0",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	srcShelf := &config.ShelfConfig{
		Name:           "shelf-a",
		Repo:           "repo-a",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	// Test cross-shelf move
	params := &moveParams{
		toShelfName: "shelf-b",
	}

	dst, err := determineDestination(srcShelf, "test-owner", params)
	if err != nil {
		t.Fatalf("determineDestination failed: %v", err)
	}

	if dst.owner != "test-owner" {
		t.Errorf("expected owner test-owner, got %s", dst.owner)
	}
	if dst.repo != "repo-b" {
		t.Errorf("expected repo repo-b, got %s", dst.repo)
	}
	if dst.release != "v2.0" {
		t.Errorf("expected release v2.0 (shelf-b default), got %s", dst.release)
	}
	if dst.shelf == nil {
		t.Fatal("expected shelf to be set")
	}
	if dst.shelf.Name != "shelf-b" {
		t.Errorf("expected shelf name shelf-b, got %s", dst.shelf.Name)
	}
}

// TestMoveBookNotFound tests error handling when destination shelf doesn't exist
func TestMoveBookNotFound(t *testing.T) {
	// Setup config with only one shelf
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v1.0",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:           "shelf-a",
				Repo:           "repo-a",
				Owner:          "test-owner",
				DefaultRelease: "v1.0",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	srcShelf := &config.ShelfConfig{
		Name:           "shelf-a",
		Repo:           "repo-a",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	// Try to move to nonexistent shelf
	params := &moveParams{
		toShelfName: "nonexistent-shelf",
	}

	_, err := determineDestination(srcShelf, "test-owner", params)
	if err == nil {
		t.Fatal("expected error for nonexistent shelf, got nil")
	}
	// Error should mention the shelf not being found
	expectedMsg := `shelf "nonexistent-shelf" not found in config (use --create-shelf to auto-create)`
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

// TestMoveToSameShelf tests destination determination for same-shelf move (different release)
func TestMoveToSameShelf(t *testing.T) {
	// Setup config
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v1.0",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:           "shelf-a",
				Repo:           "repo-a",
				Owner:          "test-owner",
				DefaultRelease: "v1.0",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	srcShelf := &config.ShelfConfig{
		Name:           "shelf-a",
		Repo:           "repo-a",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	// Test same-shelf move to different release
	params := &moveParams{
		toRelease: "v2.0",
	}

	dst, err := determineDestination(srcShelf, "test-owner", params)
	if err != nil {
		t.Fatalf("determineDestination failed: %v", err)
	}

	// Should stay in same repo but different release
	if dst.owner != "test-owner" {
		t.Errorf("expected owner test-owner, got %s", dst.owner)
	}
	if dst.repo != "repo-a" {
		t.Errorf("expected repo repo-a (same as source), got %s", dst.repo)
	}
	if dst.release != "v2.0" {
		t.Errorf("expected release v2.0, got %s", dst.release)
	}
	if dst.shelf != nil {
		t.Error("expected shelf to be nil for same-shelf move")
	}
}

// TestMovePreserveMetadata tests destination determination with explicit release override
func TestMovePreserveMetadata(t *testing.T) {
	// Setup config
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v1.0",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:           "shelf-a",
				Repo:           "repo-a",
				Owner:          "test-owner",
				DefaultRelease: "v1.0",
			},
			{
				Name:           "shelf-b",
				Repo:           "repo-b",
				Owner:          "test-owner",
				DefaultRelease: "v2.0",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	srcShelf := &config.ShelfConfig{
		Name:           "shelf-a",
		Repo:           "repo-a",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	// Test cross-shelf move with explicit release override
	params := &moveParams{
		toShelfName: "shelf-b",
		toRelease:   "v3.0", // Override shelf-b's default v2.0
	}

	dst, err := determineDestination(srcShelf, "test-owner", params)
	if err != nil {
		t.Fatalf("determineDestination failed: %v", err)
	}

	if dst.owner != "test-owner" {
		t.Errorf("expected owner test-owner, got %s", dst.owner)
	}
	if dst.repo != "repo-b" {
		t.Errorf("expected repo repo-b, got %s", dst.repo)
	}
	if dst.release != "v3.0" {
		t.Errorf("expected release v3.0 (explicit override), got %s", dst.release)
	}
	if dst.shelf == nil {
		t.Fatal("expected shelf to be set")
	}
	if dst.shelf.Name != "shelf-b" {
		t.Errorf("expected shelf name shelf-b, got %s", dst.shelf.Name)
	}
}

// TestMoveWithCache tests destination determination with missing release specification
func TestMoveWithCache(t *testing.T) {
	// Setup config
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v1.0",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:           "shelf-a",
				Repo:           "repo-a",
				Owner:          "test-owner",
				DefaultRelease: "v1.0",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	srcShelf := &config.ShelfConfig{
		Name:           "shelf-a",
		Repo:           "repo-a",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	// Test same-shelf move without specifying release - should error
	params := &moveParams{
		// No toRelease and no toShelfName
	}

	_, err := determineDestination(srcShelf, "test-owner", params)
	if err == nil {
		t.Fatal("expected error when no release specified for same-shelf move, got nil")
	}
	if err.Error() != "--to-release is required when not using --to-shelf" {
		t.Errorf("unexpected error message: %v", err)
	}
}
