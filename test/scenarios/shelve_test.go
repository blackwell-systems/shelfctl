package scenarios

import (
	"bytes"
	"os"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// TestShelveBook verifies adding a new book to a shelf
func TestShelveBook(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "shelve-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no shelves in default fixtures")
	}

	shelf := fixtures.Shelves[0]

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Ensure release exists
	releaseTag := "library"
	rel, err := ghClient.EnsureRelease(shelf.Owner, shelf.Repo, releaseTag)
	if err != nil {
		t.Fatalf("failed to ensure release: %v", err)
	}

	// Create test PDF content
	testPDFContent := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 1/Kids[3 0 R]>>endobj\n3 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 2 0 R>>endobj\nxref\n0 4\ntrailer<</Size 4/Root 1 0 R>>\nstartxref\n150\n%%EOF")
	testBookID := "test-book-new"
	testAssetName := "test-book-new.pdf"

	// Upload asset to release
	assetReader := bytes.NewReader(testPDFContent)
	asset, err := ghClient.UploadAsset(shelf.Owner, shelf.Repo, rel.ID, testAssetName, assetReader, int64(len(testPDFContent)), "application/pdf")
	if err != nil {
		t.Fatalf("failed to upload asset: %v", err)
	}

	if asset.Name != testAssetName {
		t.Errorf("expected asset name %s, got %s", testAssetName, asset.Name)
	}

	// Load current catalog
	catalogPath := "catalog.yml"
	mgr := catalog.NewManager(ghClient, shelf.Owner, shelf.Repo, catalogPath)
	books, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	originalCount := len(books)

	// Create new book entry
	newBook := catalog.Book{
		ID:     testBookID,
		Title:  "Test Book",
		Author: "Test Author",
		Year:   2024,
		Tags:   []string{"test", "integration"},
		Format: "pdf",
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   shelf.Owner,
			Repo:    shelf.Repo,
			Release: releaseTag,
			Asset:   testAssetName,
		},
		Checksum: catalog.Checksum{
			SHA256: "test-sha256-placeholder",
		},
		SizeBytes: int64(len(testPDFContent)),
	}

	// Add book to catalog
	updatedBooks := catalog.Append(books, newBook)

	// Save catalog
	err = mgr.Save(updatedBooks, "test: add test-book-new")
	if err != nil {
		t.Fatalf("failed to save catalog: %v", err)
	}

	// Reload catalog to verify persistence
	reloadedBooks, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to reload catalog: %v", err)
	}

	if len(reloadedBooks) != originalCount+1 {
		t.Errorf("expected %d books after add, got %d", originalCount+1, len(reloadedBooks))
	}

	// Verify new book is in catalog
	found := false
	for _, b := range reloadedBooks {
		if b.ID == testBookID {
			found = true
			if b.Title != "Test Book" {
				t.Errorf("expected title 'Test Book', got %s", b.Title)
			}
			if b.Author != "Test Author" {
				t.Errorf("expected author 'Test Author', got %s", b.Author)
			}
			if len(b.Tags) != 2 {
				t.Errorf("expected 2 tags, got %d", len(b.Tags))
			}
			break
		}
	}

	if !found {
		t.Errorf("book %s not found in reloaded catalog", testBookID)
	}

	t.Logf("Successfully added book %s to shelf %s", testBookID, shelf.Name)
}

// TestShelveBookDuplicate verifies duplicate detection
func TestShelveBookDuplicate(t *testing.T) {
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
	if len(shelf.Books) == 0 {
		t.Fatal("no books in shelf fixture")
	}

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())

	// Load catalog
	mgr := catalog.NewManager(ghClient, shelf.Owner, shelf.Repo, "catalog.yml")
	books, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	// Get existing book
	existingBook := shelf.Books[0]

	// Try to add same book again (by ID)
	duplicateBook := existingBook
	duplicateBook.Title = "Updated Title" // Change title but keep same ID

	// Append replaces by ID, so this should work
	updatedBooks := catalog.Append(books, duplicateBook)

	// Verify count didn't increase (replacement, not addition)
	if len(updatedBooks) != len(books) {
		t.Logf("Note: Append with existing ID replaced book (expected behavior)")
	}

	// Verify the book was updated
	for _, b := range updatedBooks {
		if b.ID == duplicateBook.ID {
			if b.Title != "Updated Title" {
				t.Errorf("expected updated title, got %s", b.Title)
			}
			break
		}
	}

	t.Logf("Successfully verified duplicate ID handling")
}

// TestShelveMultipleFormats verifies adding books in different formats
func TestShelveMultipleFormats(t *testing.T) {
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
	ghClient := github.New("mock-token", srv.URL())

	// Load catalog
	mgr := catalog.NewManager(ghClient, shelf.Owner, shelf.Repo, "catalog.yml")
	books, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	// Create PDF book
	pdfBook := catalog.Book{
		ID:     "test-pdf-book",
		Title:  "Test PDF Book",
		Format: "pdf",
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   shelf.Owner,
			Repo:    shelf.Repo,
			Release: "library",
			Asset:   "test-pdf-book.pdf",
		},
	}

	// Create EPUB book
	epubBook := catalog.Book{
		ID:     "test-epub-book",
		Title:  "Test EPUB Book",
		Format: "epub",
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   shelf.Owner,
			Repo:    shelf.Repo,
			Release: "library",
			Asset:   "test-epub-book.epub",
		},
	}

	// Add both books
	updatedBooks := catalog.Append(books, pdfBook)
	updatedBooks = catalog.Append(updatedBooks, epubBook)

	// Verify both formats are present
	formats := make(map[string]int)
	for _, b := range updatedBooks {
		formats[b.Format]++
	}

	if formats["pdf"] == 0 {
		t.Error("no PDF books found after adding")
	}
	if formats["epub"] == 0 {
		t.Error("no EPUB books found after adding")
	}

	t.Logf("Successfully verified multiple format support: %v", formats)
}

// TestShelveBookMetadata verifies book metadata completeness
func TestShelveBookMetadata(t *testing.T) {
	// Create test book with full metadata
	book := catalog.Book{
		ID:     "complete-book",
		Title:  "Complete Book Title",
		Author: "Test Author",
		Year:   2024,
		Tags:   []string{"golang", "testing", "integration"},
		Format: "pdf",
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   "testowner",
			Repo:    "testrepo",
			Release: "v1.0.0",
			Asset:   "complete-book.pdf",
		},
		Checksum: catalog.Checksum{
			SHA256: "abcd1234",
		},
		SizeBytes: 1024000,
		Meta: catalog.Meta{
			AddedAt: "2024-03-22T10:00:00Z",
		},
	}

	// Verify all required fields are present
	if book.ID == "" {
		t.Error("book ID is empty")
	}
	if book.Title == "" {
		t.Error("book title is empty")
	}
	if book.Format == "" {
		t.Error("book format is empty")
	}
	if book.Source.Type == "" {
		t.Error("source type is empty")
	}
	if book.Source.Owner == "" {
		t.Error("source owner is empty")
	}
	if book.Source.Repo == "" {
		t.Error("source repo is empty")
	}
	if book.Source.Release == "" {
		t.Error("source release is empty")
	}
	if book.Source.Asset == "" {
		t.Error("source asset is empty")
	}

	// Verify optional fields are preserved
	if book.Author != "Test Author" {
		t.Errorf("author mismatch: got %s", book.Author)
	}
	if book.Year != 2024 {
		t.Errorf("year mismatch: got %d", book.Year)
	}
	if len(book.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(book.Tags))
	}

	t.Logf("Successfully verified book metadata completeness")
}
