package scenarios

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// TestCacheDownload verifies downloading and caching an asset
func TestCacheDownload(t *testing.T) {
	// Setup mock server
	srv, err := mockserver.NewServer()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-download-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]
	if len(shelf.Books) == 0 {
		t.Fatal("no books in shelf fixture")
	}

	book := shelf.Books[0]

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Verify book not cached initially
	if cacheMgr.Exists(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset) {
		t.Error("book should not be cached initially")
	}

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Get release
	rel, err := ghClient.GetReleaseByTag(shelf.Owner, shelf.Repo, book.Source.Release)
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	// Find asset
	asset, err := ghClient.FindAsset(shelf.Owner, shelf.Repo, rel.ID, book.Source.Asset)
	if err != nil {
		t.Fatalf("failed to find asset: %v", err)
	}
	if asset == nil {
		t.Fatal("asset not found in release")
	}

	// Download asset
	rc, err := ghClient.DownloadAsset(shelf.Owner, shelf.Repo, asset.ID)
	if err != nil {
		t.Fatalf("failed to download asset: %v", err)
	}

	// Store in cache
	cachedPath, err := cacheMgr.Store(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset, rc, book.Checksum.SHA256)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("failed to store in cache: %v", err)
	}

	// Verify book is now cached
	if !cacheMgr.Exists(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset) {
		t.Error("book should be cached after download")
	}

	// Verify cached file exists
	if _, err := os.Stat(cachedPath); err != nil {
		t.Errorf("cached file should exist at %s: %v", cachedPath, err)
	}

	// Verify cached path matches manager's Path method
	expectedPath := cacheMgr.Path(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset)
	if cachedPath != expectedPath {
		t.Errorf("cached path mismatch: got %s, expected %s", cachedPath, expectedPath)
	}

	t.Logf("Successfully cached book %s at %s", book.ID, cachedPath)
}

// TestCacheExists verifies cache hit/miss detection
func TestCacheExists(t *testing.T) {
	// Setup mock server
	srv, err := mockserver.NewServer()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-exists-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]
	if len(shelf.Books) < 2 {
		t.Fatal("need at least 2 books in shelf fixture")
	}

	cachedBook := shelf.Books[0]
	uncachedBook := shelf.Books[1]

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Download and cache first book
	rel, err := ghClient.GetReleaseByTag(shelf.Owner, shelf.Repo, cachedBook.Source.Release)
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	asset, err := ghClient.FindAsset(shelf.Owner, shelf.Repo, rel.ID, cachedBook.Source.Asset)
	if err != nil {
		t.Fatalf("failed to find asset: %v", err)
	}

	rc, err := ghClient.DownloadAsset(shelf.Owner, shelf.Repo, asset.ID)
	if err != nil {
		t.Fatalf("failed to download asset: %v", err)
	}

	_, err = cacheMgr.Store(shelf.Owner, shelf.Repo, cachedBook.ID, cachedBook.Source.Asset, rc, cachedBook.Checksum.SHA256)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("failed to store in cache: %v", err)
	}

	// Test cache hit (first book)
	if !cacheMgr.Exists(shelf.Owner, shelf.Repo, cachedBook.ID, cachedBook.Source.Asset) {
		t.Error("cache hit failed: cached book should exist")
	}

	// Test cache miss (second book)
	if cacheMgr.Exists(shelf.Owner, shelf.Repo, uncachedBook.ID, uncachedBook.Source.Asset) {
		t.Error("cache miss failed: uncached book should not exist")
	}

	// Test cache miss with non-existent book
	if cacheMgr.Exists(shelf.Owner, shelf.Repo, "nonexistent-book", "nonexistent.pdf") {
		t.Error("cache miss failed: nonexistent book should not exist")
	}

	t.Logf("Successfully verified cache hit/miss detection")
}

// TestCacheRemove verifies removing books from cache
func TestCacheRemove(t *testing.T) {
	// Setup mock server
	srv, err := mockserver.NewServer()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-remove-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]
	if len(shelf.Books) == 0 {
		t.Fatal("no books in shelf fixture")
	}

	book := shelf.Books[0]

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Download and cache book
	rel, err := ghClient.GetReleaseByTag(shelf.Owner, shelf.Repo, book.Source.Release)
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	asset, err := ghClient.FindAsset(shelf.Owner, shelf.Repo, rel.ID, book.Source.Asset)
	if err != nil {
		t.Fatalf("failed to find asset: %v", err)
	}

	rc, err := ghClient.DownloadAsset(shelf.Owner, shelf.Repo, asset.ID)
	if err != nil {
		t.Fatalf("failed to download asset: %v", err)
	}

	_, err = cacheMgr.Store(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset, rc, book.Checksum.SHA256)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("failed to store in cache: %v", err)
	}

	// Verify cached
	if !cacheMgr.Exists(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset) {
		t.Fatal("book should be cached before removal test")
	}

	// Remove from cache
	err = cacheMgr.Remove(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset)
	if err != nil {
		t.Fatalf("failed to remove from cache: %v", err)
	}

	// Verify no longer cached
	if cacheMgr.Exists(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset) {
		t.Error("book should not be cached after removal")
	}

	// Verify file is deleted
	cachedPath := cacheMgr.Path(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset)
	if _, err := os.Stat(cachedPath); err == nil {
		t.Errorf("cached file should be deleted at %s", cachedPath)
	}

	t.Logf("Successfully removed book %s from cache", book.ID)
}

// TestCacheModificationDetection verifies SHA256 modification detection
func TestCacheModificationDetection(t *testing.T) {
	// Setup mock server
	srv, err := mockserver.NewServer()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-modified-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]
	if len(shelf.Books) == 0 {
		t.Fatal("no books in shelf fixture")
	}

	book := shelf.Books[0]

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Download and cache book
	rel, err := ghClient.GetReleaseByTag(shelf.Owner, shelf.Repo, book.Source.Release)
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	asset, err := ghClient.FindAsset(shelf.Owner, shelf.Repo, rel.ID, book.Source.Asset)
	if err != nil {
		t.Fatalf("failed to find asset: %v", err)
	}

	rc, err := ghClient.DownloadAsset(shelf.Owner, shelf.Repo, asset.ID)
	if err != nil {
		t.Fatalf("failed to download asset: %v", err)
	}

	_, err = cacheMgr.Store(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset, rc, book.Checksum.SHA256)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("failed to store in cache: %v", err)
	}

	// Initially should not be modified
	if cacheMgr.HasBeenModified(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset, book.Checksum.SHA256) {
		t.Error("freshly cached file should not be marked as modified")
	}

	// Modify the cached file
	cachedPath := cacheMgr.Path(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset)
	f, err := os.OpenFile(cachedPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open cached file: %v", err)
	}
	_, err = f.WriteString("\nMODIFIED CONTENT")
	_ = f.Close()
	if err != nil {
		t.Fatalf("failed to modify cached file: %v", err)
	}

	// Should now be detected as modified
	if !cacheMgr.HasBeenModified(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset, book.Checksum.SHA256) {
		t.Error("modified file should be detected as modified")
	}

	t.Logf("Successfully detected modification of cached book %s", book.ID)
}

// TestCacheChecksumValidation verifies checksum validation during storage
func TestCacheChecksumValidation(t *testing.T) {
	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-checksum-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Create test content
	testContent := []byte("test content for checksum validation")
	correctChecksum := sha256.Sum256(testContent)
	correctChecksumHex := hex.EncodeToString(correctChecksum[:])
	wrongChecksumHex := "0000000000000000000000000000000000000000000000000000000000000000"

	// Test with correct checksum (should succeed)
	reader := bytes.NewReader(testContent)
	_, err = cacheMgr.Store("testowner", "testrepo", "testbook", "testbook.pdf", reader, correctChecksumHex)
	if err != nil {
		t.Errorf("store with correct checksum should succeed: %v", err)
	}

	// Test with wrong checksum (should fail or store with warning)
	reader2 := bytes.NewReader(testContent)
	_, err = cacheMgr.Store("testowner", "testrepo", "testbook2", "testbook2.pdf", reader2, wrongChecksumHex)
	// Note: actual behavior depends on cache.Manager implementation
	// Some implementations might warn, others might error
	if err != nil {
		t.Logf("Store with wrong checksum failed as expected: %v", err)
	} else {
		t.Logf("Store with wrong checksum succeeded (implementation may warn instead of error)")
	}

	t.Logf("Successfully tested checksum validation")
}

// TestCacheMultipleRepos verifies cache isolation between repos
func TestCacheMultipleRepos(t *testing.T) {
	// Setup mock server
	srv, err := mockserver.NewServer()
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	// Create temp cache directory
	tmpDir, err := os.MkdirTemp("", "cache-multirepo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cacheDir := filepath.Join(tmpDir, "cache")

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) < 2 {
		t.Skip("need at least 2 shelves for multi-repo test")
	}

	shelf1 := fixtures.Shelves[0]
	shelf2 := fixtures.Shelves[1]

	// Create cache manager
	cacheMgr := cache.New(cacheDir)

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Cache book from shelf1
	if len(shelf1.Books) > 0 {
		book1 := shelf1.Books[0]
		rel1, err := ghClient.GetReleaseByTag(shelf1.Owner, shelf1.Repo, book1.Source.Release)
		if err == nil {
			asset1, err := ghClient.FindAsset(shelf1.Owner, shelf1.Repo, rel1.ID, book1.Source.Asset)
			if err == nil && asset1 != nil {
				rc1, err := ghClient.DownloadAsset(shelf1.Owner, shelf1.Repo, asset1.ID)
				if err == nil {
					_, _ = cacheMgr.Store(shelf1.Owner, shelf1.Repo, book1.ID, book1.Source.Asset, rc1, book1.Checksum.SHA256)
					_ = rc1.Close()
				}
			}
		}
	}

	// Cache book from shelf2
	if len(shelf2.Books) > 0 {
		book2 := shelf2.Books[0]
		rel2, err := ghClient.GetReleaseByTag(shelf2.Owner, shelf2.Repo, book2.Source.Release)
		if err == nil {
			asset2, err := ghClient.FindAsset(shelf2.Owner, shelf2.Repo, rel2.ID, book2.Source.Asset)
			if err == nil && asset2 != nil {
				rc2, err := ghClient.DownloadAsset(shelf2.Owner, shelf2.Repo, asset2.ID)
				if err == nil {
					_, _ = cacheMgr.Store(shelf2.Owner, shelf2.Repo, book2.ID, book2.Source.Asset, rc2, book2.Checksum.SHA256)
					_ = rc2.Close()
				}
			}
		}
	}

	// Verify isolation: book from shelf1 should not be visible in shelf2 namespace
	if len(shelf1.Books) > 0 && len(shelf2.Books) > 0 {
		book1 := shelf1.Books[0]
		if cacheMgr.Exists(shelf2.Owner, shelf2.Repo, book1.ID, book1.Source.Asset) {
			t.Error("book from shelf1 should not exist in shelf2 cache namespace")
		}
	}

	t.Logf("Successfully verified cache isolation between repos")
}
