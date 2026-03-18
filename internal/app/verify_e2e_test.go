package app

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghpkg "github.com/blackwell-systems/shelfctl/internal/github"
)

// fakeGitHubClientForVerify is a more complete fake for verify tests
type fakeGitHubClientForVerify struct {
	getFileContentFn     func(owner, repo, path, ref string) ([]byte, string, error)
	getReleaseByTagFn    func(owner, repo, tag string) (*ghpkg.Release, error)
	listReleaseAssetsFn  func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error)
	commitFileFn         func(owner, repo, filePath string, content []byte, message string) error
	deleteAssetFn        func(owner, repo string, assetID int64) error
	commitFileCalls      []commitFileCall
	deleteAssetCalls     []deleteAssetCall
}

type commitFileCall struct {
	owner    string
	repo     string
	filePath string
	content  []byte
	message  string
}

type deleteAssetCall struct {
	owner   string
	repo    string
	assetID int64
}

func (f *fakeGitHubClientForVerify) GetReleaseByTag(owner, repo, tag string) (*ghpkg.Release, error) {
	if f.getReleaseByTagFn != nil {
		return f.getReleaseByTagFn(owner, repo, tag)
	}
	return &ghpkg.Release{ID: 1, TagName: tag}, nil
}

func (f *fakeGitHubClientForVerify) EnsureRelease(owner, repo, tag string) (*ghpkg.Release, error) {
	return &ghpkg.Release{ID: 1, TagName: tag}, nil
}

func (f *fakeGitHubClientForVerify) FindAsset(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
	return nil, nil
}

func (f *fakeGitHubClientForVerify) ListReleaseAssets(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
	if f.listReleaseAssetsFn != nil {
		return f.listReleaseAssetsFn(owner, repo, releaseID)
	}
	return nil, nil
}

func (f *fakeGitHubClientForVerify) DownloadAsset(owner, repo string, assetID int64) (io.ReadCloser, error) {
	return nil, nil
}

func (f *fakeGitHubClientForVerify) UploadAsset(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*ghpkg.Asset, error) {
	return nil, nil
}

func (f *fakeGitHubClientForVerify) DeleteAsset(owner, repo string, assetID int64) error {
	if f.deleteAssetFn != nil {
		f.deleteAssetCalls = append(f.deleteAssetCalls, deleteAssetCall{
			owner:   owner,
			repo:    repo,
			assetID: assetID,
		})
		return f.deleteAssetFn(owner, repo, assetID)
	}
	f.deleteAssetCalls = append(f.deleteAssetCalls, deleteAssetCall{
		owner:   owner,
		repo:    repo,
		assetID: assetID,
	})
	return nil
}

func (f *fakeGitHubClientForVerify) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	if f.getFileContentFn != nil {
		return f.getFileContentFn(owner, repo, path, ref)
	}
	return nil, "", nil
}

func (f *fakeGitHubClientForVerify) CommitFile(owner, repo, filePath string, content []byte, message string) error {
	if f.commitFileFn != nil {
		f.commitFileCalls = append(f.commitFileCalls, commitFileCall{
			owner:    owner,
			repo:     repo,
			filePath: filePath,
			content:  content,
			message:  message,
		})
		return f.commitFileFn(owner, repo, filePath, content, message)
	}
	f.commitFileCalls = append(f.commitFileCalls, commitFileCall{
		owner:    owner,
		repo:     repo,
		filePath: filePath,
		content:  content,
		message:  message,
	})
	return nil
}

// TestVerifySingleShelf_OrphanedCatalogEntry_NoFix verifies detect mode
func TestVerifySingleShelf_OrphanedCatalogEntry_NoFix(t *testing.T) {
	// Redirect stdout to suppress verify output
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
			Release: "v1.0",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create catalog with one book whose asset is NOT in the release
	catalogYAML := `- id: orphaned-book
  title: "Orphaned Book"
  format: pdf
  source:
    release: v1.0
    asset: orphaned.pdf
  checksum:
    sha256: abc123
`

	fake := &fakeGitHubClientForVerify{
		getFileContentFn: func(owner, repo, path, ref string) ([]byte, string, error) {
			if path == "catalog.yml" {
				return []byte(catalogYAML), "", nil
			}
			return nil, "", fmt.Errorf("file not found")
		},
		getReleaseByTagFn: func(owner, repo, tag string) (*ghpkg.Release, error) {
			return &ghpkg.Release{ID: 1, TagName: "v1.0"}, nil
		},
		listReleaseAssetsFn: func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
			// Return empty asset list - so orphaned.pdf is NOT in the release
			return []ghpkg.Asset{}, nil
		},
	}

	// Save and restore gh
	origGh := gh
	gh = nil // Set to nil to ensure test doesn't use package var
	t.Cleanup(func() {
		gh = origGh
	})

	// Save and restore cacheMgr
	origCacheMgr := cacheMgr
	testCacheMgr := cache.New(t.TempDir())
	cacheMgr = nil
	t.Cleanup(func() {
		cacheMgr = origCacheMgr
	})

	shelf := &config.ShelfConfig{
		Name:           "test-shelf",
		Repo:           "test-repo",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	issues := verifySingleShelfWithClient(shelf, false, fake, testCacheMgr)

	// Verify one issue of type orphaned_catalog
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].Type != "orphaned_catalog" {
		t.Errorf("expected issue type 'orphaned_catalog', got %q", issues[0].Type)
	}

	if issues[0].BookID != "orphaned-book" {
		t.Errorf("expected BookID 'orphaned-book', got %q", issues[0].BookID)
	}

	// Verify catalog was NOT committed (detect mode only)
	if len(fake.commitFileCalls) > 0 {
		t.Errorf("catalog should not be committed in detect mode, got %d calls", len(fake.commitFileCalls))
	}
}

// TestVerifySingleShelf_OrphanedCatalogEntry_WithFix verifies fix mode
func TestVerifySingleShelf_OrphanedCatalogEntry_WithFix(t *testing.T) {
	// Redirect stdout to suppress verify output
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
			Release: "v1.0",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create catalog with one orphaned book and one valid book
	catalogYAML := `- id: orphaned-book
  title: "Orphaned Book"
  format: pdf
  source:
    release: v1.0
    asset: orphaned.pdf
  checksum:
    sha256: abc123
- id: valid-book
  title: "Valid Book"
  format: pdf
  source:
    release: v1.0
    asset: valid.pdf
  checksum:
    sha256: def456
`

	fake := &fakeGitHubClientForVerify{
		getFileContentFn: func(owner, repo, path, ref string) ([]byte, string, error) {
			if path == "catalog.yml" {
				return []byte(catalogYAML), "", nil
			}
			return nil, "", fmt.Errorf("file not found")
		},
		getReleaseByTagFn: func(owner, repo, tag string) (*ghpkg.Release, error) {
			return &ghpkg.Release{ID: 1, TagName: "v1.0"}, nil
		},
		listReleaseAssetsFn: func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
			// Only valid.pdf exists in release
			return []ghpkg.Asset{
				{ID: 101, Name: "valid.pdf"},
			}, nil
		},
	}

	// Save and restore gh
	origGh := gh
	gh = nil // Set to nil to ensure test doesn't use package var
	t.Cleanup(func() {
		gh = origGh
	})

	// Save and restore cacheMgr
	origCacheMgr := cacheMgr
	testCacheMgr := cache.New(t.TempDir())
	cacheMgr = nil
	t.Cleanup(func() {
		cacheMgr = origCacheMgr
	})

	shelf := &config.ShelfConfig{
		Name:           "test-shelf",
		Repo:           "test-repo",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	issues := verifySingleShelfWithClient(shelf, true, fake, testCacheMgr)

	// Verify one issue detected
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].Type != "orphaned_catalog" {
		t.Errorf("expected issue type 'orphaned_catalog', got %q", issues[0].Type)
	}

	// Verify catalog WAS committed
	if len(fake.commitFileCalls) == 0 {
		t.Fatal("catalog should be committed in fix mode")
	}

	// Verify the orphaned book is absent from committed data
	committedData := fake.commitFileCalls[0].content
	books, err := catalog.Parse(committedData)
	if err != nil {
		t.Fatalf("failed to parse committed catalog: %v", err)
	}

	// Should have only 1 book (valid-book)
	if len(books) != 1 {
		t.Errorf("expected 1 book in committed catalog, got %d", len(books))
	}

	if len(books) > 0 && books[0].ID != "valid-book" {
		t.Errorf("expected remaining book to be 'valid-book', got %q", books[0].ID)
	}

	// Verify orphaned-book is absent
	for _, b := range books {
		if b.ID == "orphaned-book" {
			t.Error("orphaned-book should have been removed from catalog")
		}
	}
}

// TestVerifySingleShelf_OrphanedAsset_WithFix verifies orphaned asset deletion
func TestVerifySingleShelf_OrphanedAsset_WithFix(t *testing.T) {
	// Redirect stdout to suppress verify output
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
			Release: "v1.0",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create catalog with one book
	catalogYAML := `- id: book1
  title: "Book One"
  format: pdf
  source:
    release: v1.0
    asset: book1.pdf
  checksum:
    sha256: abc123
`

	fake := &fakeGitHubClientForVerify{
		getFileContentFn: func(owner, repo, path, ref string) ([]byte, string, error) {
			if path == "catalog.yml" {
				return []byte(catalogYAML), "", nil
			}
			return nil, "", fmt.Errorf("file not found")
		},
		getReleaseByTagFn: func(owner, repo, tag string) (*ghpkg.Release, error) {
			return &ghpkg.Release{ID: 1, TagName: "v1.0"}, nil
		},
		listReleaseAssetsFn: func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
			// Release has both book1.pdf and orphaned-asset.pdf
			return []ghpkg.Asset{
				{ID: 101, Name: "book1.pdf"},
				{ID: 102, Name: "orphaned-asset.pdf"}, // Not in catalog
			}, nil
		},
	}

	// Save and restore gh
	origGh := gh
	gh = nil // Set to nil to ensure test doesn't use package var
	t.Cleanup(func() {
		gh = origGh
	})

	// Save and restore cacheMgr
	origCacheMgr := cacheMgr
	testCacheMgr := cache.New(t.TempDir())
	cacheMgr = nil
	t.Cleanup(func() {
		cacheMgr = origCacheMgr
	})

	shelf := &config.ShelfConfig{
		Name:           "test-shelf",
		Repo:           "test-repo",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	issues := verifySingleShelfWithClient(shelf, true, fake, testCacheMgr)

	// Verify one issue of type orphaned_asset
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].Type != "orphaned_asset" {
		t.Errorf("expected issue type 'orphaned_asset', got %q", issues[0].Type)
	}

	if issues[0].AssetName != "orphaned-asset.pdf" {
		t.Errorf("expected AssetName 'orphaned-asset.pdf', got %q", issues[0].AssetName)
	}

	// Verify DeleteAsset was called
	if len(fake.deleteAssetCalls) == 0 {
		t.Fatal("DeleteAsset should have been called in fix mode")
	}

	if fake.deleteAssetCalls[0].assetID != 102 {
		t.Errorf("expected DeleteAsset to be called with asset ID 102, got %d", fake.deleteAssetCalls[0].assetID)
	}
}

// TestVerifySingleShelf_Clean verifies no issues when catalog and assets match
func TestVerifySingleShelf_Clean(t *testing.T) {
	// Redirect stdout to suppress verify output
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
			Release: "v1.0",
		},
	}
	t.Cleanup(func() {
		cfg = origCfg
	})

	// Create catalog with two books
	catalogYAML := `- id: book1
  title: "Book One"
  format: pdf
  source:
    release: v1.0
    asset: book1.pdf
  checksum:
    sha256: abc123
- id: book2
  title: "Book Two"
  format: epub
  source:
    release: v1.0
    asset: book2.epub
  checksum:
    sha256: def456
`

	fake := &fakeGitHubClientForVerify{
		getFileContentFn: func(owner, repo, path, ref string) ([]byte, string, error) {
			if path == "catalog.yml" {
				return []byte(catalogYAML), "", nil
			}
			return nil, "", fmt.Errorf("file not found")
		},
		getReleaseByTagFn: func(owner, repo, tag string) (*ghpkg.Release, error) {
			return &ghpkg.Release{ID: 1, TagName: "v1.0"}, nil
		},
		listReleaseAssetsFn: func(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
			// Release has exactly the assets in catalog (perfect match)
			return []ghpkg.Asset{
				{ID: 101, Name: "book1.pdf"},
				{ID: 102, Name: "book2.epub"},
			}, nil
		},
	}

	// Save and restore gh
	origGh := gh
	gh = nil // Set to nil to ensure test doesn't use package var
	t.Cleanup(func() {
		gh = origGh
	})

	// Save and restore cacheMgr
	origCacheMgr := cacheMgr
	testCacheMgr := cache.New(t.TempDir())
	cacheMgr = nil
	t.Cleanup(func() {
		cacheMgr = origCacheMgr
	})

	shelf := &config.ShelfConfig{
		Name:           "test-shelf",
		Repo:           "test-repo",
		Owner:          "test-owner",
		DefaultRelease: "v1.0",
	}

	issues := verifySingleShelfWithClient(shelf, false, fake, testCacheMgr)

	// Verify zero issues returned
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for clean catalog, got %d", len(issues))
	}

	// Verify no commits or deletes
	if len(fake.commitFileCalls) > 0 {
		t.Errorf("should not commit when clean, got %d calls", len(fake.commitFileCalls))
	}

	if len(fake.deleteAssetCalls) > 0 {
		t.Errorf("should not delete assets when clean, got %d calls", len(fake.deleteAssetCalls))
	}
}

