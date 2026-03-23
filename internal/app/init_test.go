package app

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
	ghpkg "github.com/blackwell-systems/shelfctl/internal/github"
)

// fakeGitHubClientForInit is a fake client for init tests
type fakeGitHubClientForInit struct {
	getRepoFn           func(owner, repo string) (*ghpkg.Repository, error)
	createRepoFn        func(name string, private bool) (*ghpkg.Repository, error)
	ensureReleaseFn     func(owner, repo, tag string) (*ghpkg.Release, error)
	getFileContentFn    func(owner, repo, path, ref string) ([]byte, string, error)
	commitFileFn        func(owner, repo, filePath string, content []byte, message string) error
	getReleaseByTagFn   func(owner, repo, tag string) (*ghpkg.Release, error)
	listReleaseAssetsFn func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error)
	findAssetFn         func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error)
	downloadAssetFn     func(owner, repo string, assetID int64) (io.ReadCloser, error)
	uploadAssetFn       func(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*ghpkg.Asset, error)
	deleteAssetFn       func(owner, repo string, assetID int64) error
	createRepoCalls     int
	ensureReleaseCalls  int
	commitFileCalls     int
}

func (f *fakeGitHubClientForInit) GetRepo(owner, repo string) (*ghpkg.Repository, error) {
	if f.getRepoFn != nil {
		return f.getRepoFn(owner, repo)
	}
	return nil, ghpkg.ErrNotFound
}

func (f *fakeGitHubClientForInit) CreateRepo(name string, private bool) (*ghpkg.Repository, error) {
	f.createRepoCalls++
	if f.createRepoFn != nil {
		return f.createRepoFn(name, private)
	}
	return &ghpkg.Repository{
		Name:    name,
		HTMLURL: fmt.Sprintf("https://github.com/test-owner/%s", name),
		Private: private,
	}, nil
}

func (f *fakeGitHubClientForInit) EnsureRelease(owner, repo, tag string) (*ghpkg.Release, error) {
	f.ensureReleaseCalls++
	if f.ensureReleaseFn != nil {
		return f.ensureReleaseFn(owner, repo, tag)
	}
	return &ghpkg.Release{
		ID:      1,
		TagName: tag,
		HTMLURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, tag),
	}, nil
}

func (f *fakeGitHubClientForInit) GetReleaseByTag(owner, repo, tag string) (*ghpkg.Release, error) {
	if f.getReleaseByTagFn != nil {
		return f.getReleaseByTagFn(owner, repo, tag)
	}
	return &ghpkg.Release{ID: 1, TagName: tag}, nil
}

func (f *fakeGitHubClientForInit) FindAsset(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
	if f.findAssetFn != nil {
		return f.findAssetFn(owner, repo, releaseID, name)
	}
	return nil, nil
}

func (f *fakeGitHubClientForInit) ListReleaseAssets(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
	if f.listReleaseAssetsFn != nil {
		return f.listReleaseAssetsFn(owner, repo, releaseID)
	}
	return nil, nil
}

func (f *fakeGitHubClientForInit) DownloadAsset(owner, repo string, assetID int64) (io.ReadCloser, error) {
	if f.downloadAssetFn != nil {
		return f.downloadAssetFn(owner, repo, assetID)
	}
	return nil, nil
}

func (f *fakeGitHubClientForInit) UploadAsset(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*ghpkg.Asset, error) {
	if f.uploadAssetFn != nil {
		return f.uploadAssetFn(owner, repo, releaseID, name, r, size, contentType)
	}
	return nil, nil
}

func (f *fakeGitHubClientForInit) DeleteAsset(owner, repo string, assetID int64) error {
	if f.deleteAssetFn != nil {
		return f.deleteAssetFn(owner, repo, assetID)
	}
	return nil
}

func (f *fakeGitHubClientForInit) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	if f.getFileContentFn != nil {
		return f.getFileContentFn(owner, repo, path, ref)
	}
	return nil, "", ghpkg.ErrNotFound
}

func (f *fakeGitHubClientForInit) CommitFile(owner, repo, filePath string, content []byte, message string) error {
	f.commitFileCalls++
	if f.commitFileFn != nil {
		return f.commitFileFn(owner, repo, filePath, content, message)
	}
	return nil
}

// TestInitWizard tests full init flow (create shelf, generate catalog, commit)
func TestInitWizard(t *testing.T) {
	// Redirect stdout to suppress init output
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	t.Cleanup(func() {
		os.Stdout = origStdout
	})

	// Save and restore cfg
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner:    "test-owner",
			TokenEnv: "GITHUB_TOKEN",
			APIBase:  "https://api.github.com",
		},
		Defaults: config.DefaultsConfig{
			Release:     "library",
			AssetNaming: "id",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	t.Setenv("SHELFCTL_CONFIG", configPath)

	fake := &fakeGitHubClientForInit{}

	// Save and restore gh
	origGh := gh
	gh = fake
	t.Cleanup(func() {
		gh = origGh
	})

	// Run init with --create-repo flag
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--owner", "test-owner",
		"--repo", "shelf-programming",
		"--name", "programming",
		"--create-repo",
		"--private=false",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify repo creation was called
	if fake.createRepoCalls != 1 {
		t.Errorf("expected CreateRepo to be called once, got %d calls", fake.createRepoCalls)
	}

	// Verify release creation was called
	if fake.ensureReleaseCalls != 1 {
		t.Errorf("expected EnsureRelease to be called once, got %d calls", fake.ensureReleaseCalls)
	}

	// Verify README commit was attempted
	if fake.commitFileCalls != 1 {
		t.Errorf("expected CommitFile to be called once for README, got %d calls", fake.commitFileCalls)
	}

	// Verify config was updated
	savedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(savedCfg.Shelves) != 1 {
		t.Fatalf("expected 1 shelf in config, got %d", len(savedCfg.Shelves))
	}

	shelf := savedCfg.Shelves[0]
	if shelf.Name != "programming" {
		t.Errorf("expected shelf name 'programming', got %q", shelf.Name)
	}
	if shelf.Repo != "shelf-programming" {
		t.Errorf("expected repo 'shelf-programming', got %q", shelf.Repo)
	}
	if shelf.Owner != "test-owner" {
		t.Errorf("expected owner 'test-owner', got %q", shelf.Owner)
	}
}

// TestInitExistingShelf tests error handling for duplicate shelf
func TestInitExistingShelf(t *testing.T) {
	// Redirect stdout to suppress init output
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	t.Cleanup(func() {
		os.Stdout = origStdout
	})

	// Save and restore cfg
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "library",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:  "programming",
				Repo:  "shelf-programming",
				Owner: "test-owner",
			},
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create temp config file with existing shelf
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	t.Setenv("SHELFCTL_CONFIG", configPath)
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save initial config: %v", err)
	}

	fake := &fakeGitHubClientForInit{
		getRepoFn: func(owner, repo string) (*ghpkg.Repository, error) {
			return &ghpkg.Repository{
				Name:    repo,
				HTMLURL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
			}, nil
		},
	}

	// Save and restore gh
	origGh := gh
	gh = fake
	t.Cleanup(func() {
		gh = origGh
	})

	// Try to init the same shelf again
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--owner", "test-owner",
		"--repo", "shelf-programming",
		"--name", "programming",
		"--create-repo",
	})

	// This should succeed but skip adding duplicate shelf
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init command should succeed for existing shelf: %v", err)
	}

	// Verify config still has only 1 shelf
	savedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(savedCfg.Shelves) != 1 {
		t.Errorf("expected 1 shelf in config (no duplicate), got %d", len(savedCfg.Shelves))
	}
}

// TestInitInvalidRepo tests validation for malformed owner/repo
func TestInitInvalidRepo(t *testing.T) {
	// Redirect stdout to suppress init output
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	t.Cleanup(func() {
		os.Stdout = origStdout
	})

	// Save and restore cfg
	origCfg := cfg
	cfg = nil // No config loaded
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Test missing owner
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--repo", "shelf-test",
	})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing owner, got nil")
	}
	if err != nil && err.Error() != "--owner is required (or set github.owner in config)" {
		t.Errorf("expected owner error, got: %v", err)
	}

	// Test missing repo
	cmd2 := newInitCmd()
	cmd2.SetArgs([]string{
		"--owner", "test-owner",
	})

	err2 := cmd2.Execute()
	if err2 == nil {
		t.Error("expected error for missing repo, got nil")
	}
	if err2 != nil && err2.Error() != "--repo is required (run 'shelfctl init --help' for examples)" {
		t.Errorf("expected repo error, got: %v", err2)
	}
}

// TestInitWithCustomCatalogPath tests non-default catalog path
func TestInitWithCustomCatalogPath(t *testing.T) {
	// Note: Current implementation doesn't support custom catalog path via flags.
	// This test documents the default behavior and can be expanded when custom paths are supported.

	// Redirect stdout to suppress init output
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	t.Cleanup(func() {
		os.Stdout = origStdout
	})

	// Save and restore cfg
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "library",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	t.Setenv("SHELFCTL_CONFIG", configPath)

	fake := &fakeGitHubClientForInit{}

	// Save and restore gh
	origGh := gh
	gh = fake
	t.Cleanup(func() {
		gh = origGh
	})

	// Run init - catalog path defaults to catalog.yml
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--owner", "test-owner",
		"--repo", "shelf-test",
		"--name", "test",
		"--create-repo",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify shelf was added with default catalog path
	savedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(savedCfg.Shelves) != 1 {
		t.Fatalf("expected 1 shelf in config, got %d", len(savedCfg.Shelves))
	}

	// Current implementation uses default catalog.yml
	// When custom catalog paths are supported, verify shelf.CatalogPath here
}

// TestInitDryRun tests preview mode without committing
func TestInitDryRun(t *testing.T) {
	// Note: Current implementation doesn't have a --dry-run flag.
	// This test documents the behavior when --create-repo is omitted (manual setup mode).

	// Redirect stdout to suppress init output
	origStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	t.Cleanup(func() {
		os.Stdout = origStdout
	})

	// Save and restore cfg
	origCfg := cfg
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "library",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create temp config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yml"
	t.Setenv("SHELFCTL_CONFIG", configPath)

	fake := &fakeGitHubClientForInit{}

	// Save and restore gh
	origGh := gh
	gh = fake
	t.Cleanup(func() {
		gh = origGh
	})

	// Run init WITHOUT --create-repo flag (manual mode)
	cmd := newInitCmd()
	cmd.SetArgs([]string{
		"--owner", "test-owner",
		"--repo", "shelf-manual",
		"--name", "manual",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify repo creation was NOT called
	if fake.createRepoCalls != 0 {
		t.Errorf("expected CreateRepo NOT to be called, got %d calls", fake.createRepoCalls)
	}

	// Verify release creation was NOT called
	if fake.ensureReleaseCalls != 0 {
		t.Errorf("expected EnsureRelease NOT to be called, got %d calls", fake.ensureReleaseCalls)
	}

	// Verify README commit was NOT attempted
	if fake.commitFileCalls != 0 {
		t.Errorf("expected CommitFile NOT to be called, got %d calls", fake.commitFileCalls)
	}

	// Verify shelf was still added to config
	savedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(savedCfg.Shelves) != 1 {
		t.Fatalf("expected 1 shelf in config, got %d", len(savedCfg.Shelves))
	}

	shelf := savedCfg.Shelves[0]
	if shelf.Name != "manual" {
		t.Errorf("expected shelf name 'manual', got %q", shelf.Name)
	}
}
