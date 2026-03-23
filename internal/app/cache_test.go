package app

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// computeFileSHA256 computes the SHA256 checksum of a file
func computeFileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file for checksum: %v", err)
	}
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// TestCacheClear verifies that clearShelfCache removes all cached books for a shelf
func TestCacheClear(t *testing.T) {
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

	// Create temp directory for cache
	tmpDir, err := os.MkdirTemp("", "test-cache-clear-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fxs := fixtures.DefaultFixtures()
	techShelf := fxs.Shelves[0] // tech shelf

	// Create cache manager
	mgr := cache.New(tmpDir)

	// Store some books in cache
	for bookID, content := range techShelf.Assets {
		// Find book to get asset name
		var assetName string
		for _, book := range techShelf.Books {
			if book.ID == bookID {
				assetName = book.Source.Asset
				break
			}
		}
		if assetName == "" {
			continue
		}

		_, err := mgr.Store(techShelf.Owner, techShelf.Repo, bookID, assetName, strings.NewReader(string(content)), "")
		if err != nil {
			t.Fatalf("failed to store book %s: %v", bookID, err)
		}
	}

	// Verify books are cached
	cachedCount := 0
	for _, book := range techShelf.Books {
		if mgr.Exists(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset) {
			cachedCount++
		}
	}
	if cachedCount == 0 {
		t.Fatal("no books were cached")
	}

	// Clear cache for each book
	for _, book := range techShelf.Books {
		if mgr.Exists(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset) {
			err := mgr.Remove(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset)
			if err != nil {
				t.Errorf("failed to remove book %s: %v", book.ID, err)
			}
		}
	}

	// Verify all books are removed
	for _, book := range techShelf.Books {
		if mgr.Exists(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset) {
			t.Errorf("book %s still exists after clear", book.ID)
		}
	}
}

// TestCacheOrphanDetection verifies detection of cached books not in catalog
func TestCacheOrphanDetection(t *testing.T) {
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

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-orphan-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fxs := fixtures.DefaultFixtures()
	techShelf := fxs.Shelves[0]

	// Create cache manager
	mgr := cache.New(tmpDir)

	// Store some valid books
	validBooks := techShelf.Books[:2]
	for _, book := range validBooks {
		content := []byte("valid book content")
		_, err := mgr.Store(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset, strings.NewReader(string(content)), "")
		if err != nil {
			t.Fatalf("failed to store valid book: %v", err)
		}
	}

	// Store an orphaned book (not in catalog)
	orphanedAsset := "orphaned-book.pdf"
	orphanedContent := []byte("orphaned content")
	repoDir := filepath.Join(tmpDir, techShelf.Repo)
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}
	orphanedPath := filepath.Join(repoDir, orphanedAsset)
	if err := os.WriteFile(orphanedPath, orphanedContent, 0644); err != nil {
		t.Fatalf("failed to write orphaned file: %v", err)
	}

	// Create shelf catalog list with only valid books
	shelves := []cache.ShelfCatalog{
		{
			Owner: techShelf.Owner,
			Repo:  techShelf.Repo,
			Books: validBooks,
		},
	}

	// Detect orphans
	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("DetectOrphans failed: %v", err)
	}

	// Should find the orphaned book
	if report.TotalCount != 1 {
		t.Errorf("expected 1 orphan, got %d", report.TotalCount)
	}

	if len(report.Entries) != 1 {
		t.Fatalf("expected 1 orphan entry, got %d", len(report.Entries))
	}

	orphan := report.Entries[0]
	if orphan.Filename != orphanedAsset {
		t.Errorf("orphan filename = %q, want %q", orphan.Filename, orphanedAsset)
	}
	if orphan.Repo != techShelf.Repo {
		t.Errorf("orphan repo = %q, want %q", orphan.Repo, techShelf.Repo)
	}
	if orphan.Size != int64(len(orphanedContent)) {
		t.Errorf("orphan size = %d, want %d", orphan.Size, len(orphanedContent))
	}
}

// TestCacheOrphanCleanup verifies removal of orphaned cache entries
func TestCacheOrphanCleanup(t *testing.T) {
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

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-orphan-cleanup-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fxs := fixtures.DefaultFixtures()
	techShelf := fxs.Shelves[0]

	// Create cache manager
	mgr := cache.New(tmpDir)

	// Store orphaned files
	repoDir := filepath.Join(tmpDir, techShelf.Repo)
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	orphanFiles := []string{"orphan1.pdf", "orphan2.pdf", "orphan3.pdf"}
	for _, filename := range orphanFiles {
		path := filepath.Join(repoDir, filename)
		if err := os.WriteFile(path, []byte("orphan content"), 0644); err != nil {
			t.Fatalf("failed to write orphan file: %v", err)
		}
	}

	// Detect orphans (no valid books in catalog)
	shelves := []cache.ShelfCatalog{
		{
			Owner: techShelf.Owner,
			Repo:  techShelf.Repo,
			Books: []catalog.Book{}, // Empty catalog
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("DetectOrphans failed: %v", err)
	}

	if report.TotalCount != len(orphanFiles) {
		t.Errorf("expected %d orphans, got %d", len(orphanFiles), report.TotalCount)
	}

	// Clear orphans
	deleted, err := mgr.ClearOrphans(report)
	if err != nil {
		t.Fatalf("ClearOrphans failed: %v", err)
	}

	if deleted != len(orphanFiles) {
		t.Errorf("expected %d deleted, got %d", len(orphanFiles), deleted)
	}

	// Verify files are gone
	for _, filename := range orphanFiles {
		path := filepath.Join(repoDir, filename)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("orphan file %s still exists", filename)
		}
	}
}

// TestCacheStatus verifies cache statistics reporting
func TestCacheStatus(t *testing.T) {
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

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-cache-status-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fxs := fixtures.DefaultFixtures()
	techShelf := fxs.Shelves[0]

	// Create cache manager
	mgr := cache.New(tmpDir)

	// Store some books
	storedCount := 0
	var totalSize int64
	for i, book := range techShelf.Books {
		if i >= 3 {
			break // Store only 3 books
		}
		content := []byte("test content for " + book.ID)
		_, err := mgr.Store(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset, strings.NewReader(string(content)), "")
		if err != nil {
			t.Fatalf("failed to store book: %v", err)
		}
		storedCount++
		totalSize += int64(len(content))
	}

	// Calculate cache statistics
	cachedCount := 0
	var calculatedSize int64
	for _, book := range techShelf.Books {
		if mgr.Exists(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset) {
			cachedCount++
			path := mgr.Path(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				calculatedSize += info.Size()
			}
		}
	}

	// Verify statistics
	if cachedCount != storedCount {
		t.Errorf("cached count = %d, want %d", cachedCount, storedCount)
	}

	if calculatedSize != totalSize {
		t.Errorf("cached size = %d, want %d", calculatedSize, totalSize)
	}

	// Verify total book count
	totalBooks := len(techShelf.Books)
	uncachedCount := totalBooks - cachedCount
	if uncachedCount != totalBooks-storedCount {
		t.Errorf("uncached count = %d, want %d", uncachedCount, totalBooks-storedCount)
	}
}

// TestCacheInvalidChecksum verifies handling of corrupted cached files
func TestCacheInvalidChecksum(t *testing.T) {
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

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-cache-checksum-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fxs := fixtures.DefaultFixtures()
	techShelf := fxs.Shelves[0]
	book := techShelf.Books[0]

	// Create cache manager
	mgr := cache.New(tmpDir)

	// Store book without checksum validation first, then verify it
	originalContent := []byte("original book content")
	_, err = mgr.Store(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset, strings.NewReader(string(originalContent)), "")
	if err != nil {
		t.Fatalf("failed to store book: %v", err)
	}

	// Compute actual checksum of stored content
	path := mgr.Path(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset)
	actualChecksum := computeFileSHA256(t, path)

	// Verify file is not modified with correct checksum
	if mgr.HasBeenModified(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset, actualChecksum) {
		t.Error("HasBeenModified should return false for unmodified file")
	}

	// Corrupt the file by writing different content
	corruptedContent := []byte("corrupted content - different from original")
	if err := os.WriteFile(path, corruptedContent, 0644); err != nil {
		t.Fatalf("failed to corrupt file: %v", err)
	}

	// Verify file is detected as modified (using the original checksum)
	if !mgr.HasBeenModified(techShelf.Owner, techShelf.Repo, book.ID, book.Source.Asset, actualChecksum) {
		t.Error("HasBeenModified should return true for corrupted file")
	}

	// Try to store with wrong checksum - should fail
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = mgr.Store(techShelf.Owner, techShelf.Repo, "test-book", "test.pdf", strings.NewReader("test data"), wrongChecksum)
	if err == nil {
		t.Error("Store with invalid checksum should fail")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}
}
