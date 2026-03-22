package scenarios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// TestBrowseList verifies non-interactive browse command lists books from catalog
func TestBrowseList(t *testing.T) {
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

	// Create temp directory for config and cache
	tmpDir, err := os.MkdirTemp("", "browse-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	// Pick first shelf for testing
	shelf := fixtures.Shelves[0]

	// Create GitHub client pointing to mock server
	ghClient := github.NewClient(srv.URL(), "mock-token")

	// Create catalog manager
	catalogPath := "catalog.yml"
	mgr := catalog.NewManager(ghClient, shelf.Owner, shelf.Repo, catalogPath)

	// Load catalog from mock server
	books, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	// Verify books were loaded
	if len(books) == 0 {
		t.Error("expected books in catalog, got none")
	}

	// Verify books match fixture data
	if len(books) != len(shelf.Books) {
		t.Errorf("expected %d books, got %d", len(shelf.Books), len(books))
	}

	// Check first book has required fields
	if len(books) > 0 {
		b := books[0]
		if b.ID == "" {
			t.Error("book ID is empty")
		}
		if b.Title == "" {
			t.Error("book title is empty")
		}
		if b.Format == "" {
			t.Error("book format is empty")
		}
		if b.Source.Type != "github_release" {
			t.Errorf("expected source type github_release, got %s", b.Source.Type)
		}
	}

	// Verify catalog content matches expected structure
	data, _, err := ghClient.GetFileContent(shelf.Owner, shelf.Repo, catalogPath, "")
	if err != nil {
		t.Fatalf("failed to fetch catalog content: %v", err)
	}

	catalogText := string(data)
	if !strings.Contains(catalogText, shelf.Books[0].ID) {
		t.Errorf("catalog should contain book ID %s", shelf.Books[0].ID)
	}

	t.Logf("Successfully browsed %d books from shelf %s", len(books), shelf.Name)
}

// TestBrowseWithCache verifies browse shows cached status correctly
func TestBrowseWithCache(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "browse-cache-test-*")
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

	// Create cache manager
	cacheMgr := cache.NewManager(cacheDir)

	// Create GitHub client
	ghClient := github.NewClient(srv.URL(), "mock-token")

	// Download and cache first book
	book := shelf.Books[0]
	rel, err := ghClient.GetReleaseByTag(shelf.Owner, shelf.Repo, book.Source.Release)
	if err != nil {
		t.Fatalf("failed to get release: %v", err)
	}

	asset, err := ghClient.FindAsset(shelf.Owner, shelf.Repo, rel.ID, book.Source.Asset)
	if err != nil {
		t.Fatalf("failed to find asset: %v", err)
	}
	if asset == nil {
		t.Fatal("asset not found")
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

	// Verify book is now cached
	if !cacheMgr.Exists(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset) {
		t.Error("book should be cached after download")
	}

	// Verify cached file path is valid
	cachedPath := cacheMgr.Path(shelf.Owner, shelf.Repo, book.ID, book.Source.Asset)
	if _, err := os.Stat(cachedPath); err != nil {
		t.Errorf("cached file should exist at %s: %v", cachedPath, err)
	}

	t.Logf("Successfully verified cache status for book %s", book.ID)
}

// TestBrowseFilter verifies catalog filtering by tag and format
func TestBrowseFilter(t *testing.T) {
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

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]

	// Create GitHub client
	ghClient := github.NewClient(srv.URL(), "mock-token")

	// Load catalog
	mgr := catalog.NewManager(ghClient, shelf.Owner, shelf.Repo, "catalog.yml")
	books, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	// Test format filter
	pdfBooks := catalog.Filter{Format: "pdf"}.Apply(books)
	for _, b := range pdfBooks {
		if b.Format != "pdf" {
			t.Errorf("format filter failed: expected pdf, got %s", b.Format)
		}
	}

	// Test tag filter (if any books have tags)
	if len(books) > 0 && len(books[0].Tags) > 0 {
		firstTag := books[0].Tags[0]
		taggedBooks := catalog.Filter{Tag: firstTag}.Apply(books)
		for _, b := range taggedBooks {
			found := false
			for _, tag := range b.Tags {
				if tag == firstTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("tag filter failed: book %s missing tag %s", b.ID, firstTag)
			}
		}
	}

	t.Logf("Successfully tested catalog filtering")
}
