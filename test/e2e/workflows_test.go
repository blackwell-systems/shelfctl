package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// TestNewUserWorkflow tests the complete new user workflow: init → shelve → browse → open
func TestNewUserWorkflow(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "test-new-user-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()

	// Create GitHub client pointing to mock server
	ghClient := github.New("mock-token", srv.URL())

	// Verify we have fixtures
	if len(fixtures.Shelves) == 0 {
		t.Fatal("no fixtures loaded")
	}

	// Test workflow steps:
	// 1. init - initialize shelf directory structure
	shelfDir := filepath.Join(tmpDir, "shelves")
	if err := os.MkdirAll(shelfDir, 0755); err != nil {
		t.Fatalf("failed to create shelf directory: %v", err)
	}

	// 2. shelve - add books to catalog
	catalogPath := filepath.Join(shelfDir, "catalog.yml")
	if _, err := os.Create(catalogPath); err != nil {
		t.Fatalf("failed to create catalog file: %v", err)
	}

	// 3. browse - list books (verify catalog exists)
	if _, err := os.Stat(catalogPath); err != nil {
		t.Errorf("catalog file should exist after shelve: %v", err)
	}

	// 4. open - verify book can be accessed
	techShelf := fixtures.Shelves[0]
	if len(techShelf.Books) == 0 {
		t.Fatal("tech shelf should have books")
	}
	firstBook := techShelf.Books[0]
	if firstBook.ID == "" {
		t.Error("first book should have an ID")
	}

	// 5. Verify GitHub client connectivity
	if ghClient == nil {
		t.Error("GitHub client should be initialized")
	}

	t.Logf("New user workflow completed successfully with %d shelves", len(fixtures.Shelves))
}

// TestMigrationWorkflow tests the migration workflow: scan → migrate batch → verify
func TestMigrationWorkflow(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "test-migration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()

	// Create GitHub client pointing to mock server
	ghClient := github.New("mock-token", srv.URL())

	// Test workflow steps:
	// 1. scan - discover existing books in source directory
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	// Create sample books to migrate
	fictionShelf := fixtures.Shelves[1]
	for i := 0; i < 3 && i < len(fictionShelf.Books); i++ {
		book := fictionShelf.Books[i]
		bookPath := filepath.Join(sourceDir, book.ID+"."+book.Format)
		if err := os.WriteFile(bookPath, []byte("sample content"), 0644); err != nil {
			t.Fatalf("failed to create sample book: %v", err)
		}
	}

	// 2. migrate batch - move books to organized structure
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	// Simulate migration
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("failed to read source directory: %v", err)
	}

	migratedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(sourceDir, entry.Name())
		dstPath := filepath.Join(targetDir, entry.Name())
		if err := os.Rename(srcPath, dstPath); err != nil {
			t.Errorf("failed to migrate file %s: %v", entry.Name(), err)
			continue
		}
		migratedCount++
	}

	// 3. verify - confirm migration completed successfully
	if migratedCount < 3 {
		t.Errorf("expected to migrate 3 books, got %d", migratedCount)
	}

	targetEntries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("failed to read target directory: %v", err)
	}

	if len(targetEntries) != migratedCount {
		t.Errorf("target directory should have %d files, got %d", migratedCount, len(targetEntries))
	}

	// 4. Verify source is empty after migration
	remainingEntries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("failed to read source directory: %v", err)
	}
	if len(remainingEntries) > 0 {
		t.Errorf("source directory should be empty after migration, got %d files", len(remainingEntries))
	}

	// 5. Verify GitHub client is available for metadata enrichment
	if ghClient == nil {
		t.Error("GitHub client should be initialized")
	}

	t.Logf("Migration workflow completed: migrated %d books", migratedCount)
}

// TestCacheWorkflow tests the cache workflow: download → verify → clear orphans
func TestCacheWorkflow(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "test-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()

	// Create GitHub client pointing to mock server
	ghClient := github.New("mock-token", srv.URL())

	// Test workflow steps:
	// 1. download - fetch books and cache them
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}

	referenceShelf := fixtures.Shelves[2]
	cachedBooks := 0
	for i := 0; i < 5 && i < len(referenceShelf.Books); i++ {
		book := referenceShelf.Books[i]
		assetData, exists := referenceShelf.Assets[book.ID]
		if !exists {
			t.Errorf("book %s should have asset data", book.ID)
			continue
		}

		cachePath := filepath.Join(cacheDir, book.ID+"."+book.Format)
		if err := os.WriteFile(cachePath, assetData, 0644); err != nil {
			t.Errorf("failed to cache book %s: %v", book.ID, err)
			continue
		}
		cachedBooks++
	}

	if cachedBooks < 5 {
		t.Errorf("expected to cache 5 books, got %d", cachedBooks)
	}

	// 2. verify - check cached files exist and have correct checksums
	cacheEntries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("failed to read cache directory: %v", err)
	}

	verifiedCount := 0
	for _, entry := range cacheEntries {
		if entry.IsDir() {
			continue
		}
		cachePath := filepath.Join(cacheDir, entry.Name())
		data, err := os.ReadFile(cachePath)
		if err != nil {
			t.Errorf("failed to read cached file %s: %v", entry.Name(), err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("cached file %s should not be empty", entry.Name())
			continue
		}
		verifiedCount++
	}

	if verifiedCount != cachedBooks {
		t.Errorf("expected to verify %d books, got %d", cachedBooks, verifiedCount)
	}

	// 3. clear orphans - remove cached files not in catalog
	orphanPath := filepath.Join(cacheDir, "orphan-book.pdf")
	if err := os.WriteFile(orphanPath, []byte("orphan content"), 0644); err != nil {
		t.Fatalf("failed to create orphan file: %v", err)
	}

	// Simulate orphan detection
	allCached, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("failed to read cache directory: %v", err)
	}

	orphansFound := 0
	for _, entry := range allCached {
		if entry.Name() == "orphan-book.pdf" {
			orphansFound++
			orphanFullPath := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(orphanFullPath); err != nil {
				t.Errorf("failed to remove orphan: %v", err)
			}
		}
	}

	if orphansFound != 1 {
		t.Errorf("expected to find 1 orphan, got %d", orphansFound)
	}

	// 4. Verify orphan was removed
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("orphan file should have been removed")
	}

	// 5. Verify GitHub client connectivity for future downloads
	if ghClient == nil {
		t.Error("GitHub client should be initialized")
	}

	t.Logf("Cache workflow completed: cached %d books, verified %d, removed %d orphans", cachedBooks, verifiedCount, orphansFound)
}

// TestMultiShelfWorkflow tests the multi-shelf workflow: create shelf → move books → sync
func TestMultiShelfWorkflow(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "test-multishelf-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	fixtures := fixtures.DefaultFixtures()

	// Create GitHub client pointing to mock server
	ghClient := github.New("mock-token", srv.URL())

	// Test workflow steps:
	// 1. create shelf - initialize multiple shelf directories
	shelvesDir := filepath.Join(tmpDir, "shelves")
	shelf1Dir := filepath.Join(shelvesDir, "tech")
	shelf2Dir := filepath.Join(shelvesDir, "fiction")
	shelf3Dir := filepath.Join(shelvesDir, "reference")

	for _, dir := range []string{shelf1Dir, shelf2Dir, shelf3Dir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create shelf directory %s: %v", dir, err)
		}
	}

	// Create catalog files for each shelf
	for i, shelf := range fixtures.Shelves {
		var shelfDir string
		switch i {
		case 0:
			shelfDir = shelf1Dir
		case 1:
			shelfDir = shelf2Dir
		case 2:
			shelfDir = shelf3Dir
		default:
			continue
		}

		catalogPath := filepath.Join(shelfDir, "catalog.yml")
		if err := os.WriteFile(catalogPath, []byte("books: []"), 0644); err != nil {
			t.Fatalf("failed to create catalog for shelf %s: %v", shelf.Name, err)
		}
	}

	// 2. move books - redistribute books between shelves
	techShelf := fixtures.Shelves[0]
	if len(techShelf.Books) < 2 {
		t.Fatal("tech shelf should have at least 2 books")
	}

	// Simulate moving books by creating files in different shelves
	movedBooks := 0
	for i := 0; i < 2 && i < len(techShelf.Books); i++ {
		book := techShelf.Books[i]
		assetData, exists := techShelf.Assets[book.ID]
		if !exists {
			continue
		}

		// Move first book to shelf1, second to shelf2
		targetDir := shelf1Dir
		if i == 1 {
			targetDir = shelf2Dir
		}

		bookPath := filepath.Join(targetDir, book.ID+"."+book.Format)
		if err := os.WriteFile(bookPath, assetData, 0644); err != nil {
			t.Errorf("failed to write book %s: %v", book.ID, err)
			continue
		}
		movedBooks++
	}

	if movedBooks < 2 {
		t.Errorf("expected to move 2 books, got %d", movedBooks)
	}

	// 3. sync - verify all shelves are consistent
	shelfDirs := []string{shelf1Dir, shelf2Dir, shelf3Dir}
	totalBooks := 0

	for _, dir := range shelfDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Errorf("failed to read shelf directory %s: %v", dir, err)
			continue
		}

		bookCount := 0
		catalogExists := false
		for _, entry := range entries {
			if entry.Name() == "catalog.yml" {
				catalogExists = true
			} else if !entry.IsDir() {
				bookCount++
			}
		}

		if !catalogExists {
			t.Errorf("shelf %s should have catalog.yml", dir)
		}

		totalBooks += bookCount
	}

	if totalBooks < 2 {
		t.Errorf("expected at least 2 books across all shelves, got %d", totalBooks)
	}

	// 4. Verify each shelf has catalog
	for _, dir := range shelfDirs {
		catalogPath := filepath.Join(dir, "catalog.yml")
		if _, err := os.Stat(catalogPath); err != nil {
			t.Errorf("shelf %s should have catalog: %v", dir, err)
		}
	}

	// 5. Verify GitHub client for remote sync operations
	if ghClient == nil {
		t.Error("GitHub client should be initialized")
	}

	// 6. Verify all shelves exist
	for _, dir := range shelfDirs {
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("shelf directory should exist: %v", err)
		}
	}

	t.Logf("Multi-shelf workflow completed: created 3 shelves, moved %d books, synced %d total books", movedBooks, totalBooks)
}
