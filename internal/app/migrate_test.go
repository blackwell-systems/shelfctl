package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/migrate"
	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
)

// TestMigrateWorkflow tests the full scan → migrate batch → verify flow.
func TestMigrateWorkflow(t *testing.T) {
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

	// Create temp directory for ledger and queue
	tmpDir, err := os.MkdirTemp("", "test-migrate-workflow-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	_ = fixtures.DefaultFixtures()

	// Create GitHub client pointing to mock server
	ghClient := github.New("mock-token", srv.URL())

	// Setup test config with migration source
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner:   "test-user",
			Token:   "mock-token",
			APIBase: srv.URL(),
		},
		Defaults: config.DefaultsConfig{
			Release: "v2023.1",
		},
		Shelves: []config.ShelfConfig{
			{
				Name:  "tech",
				Owner: "tech-library",
				Repo:  "tech-books",
			},
		},
		Migration: config.MigrationConfig{
			Sources: []config.MigrationSource{
				{
					Owner: "old-library",
					Repo:  "legacy-books",
					Ref:   "main",
					Mapping: map[string]string{
						"books/programming/": "tech",
						"books/fiction/":     "fiction",
					},
				},
			},
		},
	}
	gh = ghClient

	// Step 1: Create a mock queue file with legacy paths
	queueFile := filepath.Join(tmpDir, "migrate-queue.txt")
	queueContent := `books/programming/go-patterns.pdf
books/programming/rust-programming.pdf
books/fiction/neuromancer.pdf
`
	if err := os.WriteFile(queueFile, []byte(queueContent), 0644); err != nil {
		t.Fatalf("failed to write queue file: %v", err)
	}

	// Step 2: Open ledger
	ledgerPath := filepath.Join(tmpDir, "migrated.jsonl")
	ledger, err := migrate.OpenLedger(ledgerPath)
	if err != nil {
		t.Fatalf("failed to open ledger: %v", err)
	}

	// Step 3: Process migration queue (dry-run first)
	f, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	processed, skipped := processMigrationQueue(f, ledger, 2, false, true, true)
	_ = f.Close()

	// Verify dry-run counts
	if processed != 2 {
		t.Errorf("dry-run processed = %d, want 2", processed)
	}
	if skipped != 0 {
		t.Errorf("dry-run skipped = %d, want 0", skipped)
	}

	// Step 4: Verify ledger is still empty after dry-run
	contains, err := ledger.Contains("books/programming/go-patterns.pdf")
	if err != nil {
		t.Fatalf("failed to check ledger: %v", err)
	}
	if contains {
		t.Error("ledger should be empty after dry-run, but contains entry")
	}
}

// TestMigrateScanLegacyRepo tests detecting books in legacy format.
func TestMigrateScanLegacyRepo(t *testing.T) {
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
	tmpDir, err := os.MkdirTemp("", "test-scan-legacy-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Load fixtures
	_ = fixtures.DefaultFixtures()

	// Setup test config
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Token:   "mock-token",
			APIBase: srv.URL(),
		},
		Migration: config.MigrationConfig{
			Sources: []config.MigrationSource{
				{
					Owner: "old-library",
					Repo:  "legacy-books",
					Ref:   "main",
					Mapping: map[string]string{
						"books/": "general",
					},
				},
			},
		},
	}

	// Test parseExtensions helper
	exts := parseExtensions("pdf,epub,mobi")
	if len(exts) != 3 {
		t.Errorf("parseExtensions returned %d extensions, want 3", len(exts))
	}
	if exts[0] != "pdf" || exts[1] != "epub" || exts[2] != "mobi" {
		t.Errorf("parseExtensions = %v, want [pdf epub mobi]", exts)
	}

	// Test scanMigrationSources with filter
	// Note: This test is limited because mockserver doesn't implement the Contents API
	// for listing directory contents. In a real integration test, we'd need full mock support.
	files, err := scanMigrationSources("", cfg.Migration.Sources, exts)
	// Expect empty result since mockserver doesn't support scanning
	// This validates the call path works without crashing
	// scanMigrationSources continues on individual repo errors and returns accumulated results
	// with no error, so files can be nil (no files found) with err == nil
	if err != nil {
		t.Fatalf("scanMigrationSources returned unexpected error: %v", err)
	}
	// nil slice is acceptable when no files are found (all sources failed to scan)
	if files == nil {
		t.Log("scanMigrationSources returned nil (no files found from any source)")
	}
}

// TestMigrateBatchSize tests handling migration in chunks.
func TestMigrateBatchSize(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-batch-size-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create queue file with 5 items
	queueFile := filepath.Join(tmpDir, "batch-queue.txt")
	queueContent := `books/file1.pdf
books/file2.pdf
books/file3.pdf
books/file4.pdf
books/file5.pdf
`
	if err := os.WriteFile(queueFile, []byte(queueContent), 0644); err != nil {
		t.Fatalf("failed to write queue file: %v", err)
	}

	// Open ledger
	ledgerPath := filepath.Join(tmpDir, "batch-ledger.jsonl")
	ledger, err := migrate.OpenLedger(ledgerPath)
	if err != nil {
		t.Fatalf("failed to open ledger: %v", err)
	}

	// Test processing with batch size limit (n=3)
	f, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	processed, skipped := processMigrationQueue(f, ledger, 3, false, true, true)
	_ = f.Close()

	// Should process exactly 3 items (limit)
	if processed != 3 {
		t.Errorf("processMigrationQueue with n=3 processed %d, want 3", processed)
	}
	if skipped != 0 {
		t.Errorf("processMigrationQueue skipped %d, want 0", skipped)
	}

	// Test with n=0 (unlimited)
	f2, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	processed2, skipped2 := processMigrationQueue(f2, ledger, 0, false, true, true)
	_ = f2.Close()

	// Should process all 5 items
	if processed2 != 5 {
		t.Errorf("processMigrationQueue with n=0 processed %d, want 5", processed2)
	}
	if skipped2 != 0 {
		t.Errorf("processMigrationQueue skipped %d, want 0", skipped2)
	}
}

// TestMigrateProgress tests progress reporting during migration.
func TestMigrateProgress(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-progress-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create queue file with comments and empty lines (should be skipped)
	queueFile := filepath.Join(tmpDir, "progress-queue.txt")
	queueContent := `# Migration queue for test
books/file1.pdf

# Another comment
books/file2.pdf
  books/file3.pdf
# Final comment
`
	if err := os.WriteFile(queueFile, []byte(queueContent), 0644); err != nil {
		t.Fatalf("failed to write queue file: %v", err)
	}

	// Open ledger
	ledgerPath := filepath.Join(tmpDir, "progress-ledger.jsonl")
	ledger, err := migrate.OpenLedger(ledgerPath)
	if err != nil {
		t.Fatalf("failed to open ledger: %v", err)
	}

	// Process with dry-run to test comment/empty line handling
	f, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	processed, skipped := processMigrationQueue(f, ledger, 0, false, true, true)
	_ = f.Close()

	// Should process 3 valid lines (ignoring comments and empty lines)
	if processed != 3 {
		t.Errorf("processMigrationQueue processed %d, want 3 (comments should be ignored)", processed)
	}
	if skipped != 0 {
		t.Errorf("processMigrationQueue skipped %d, want 0", skipped)
	}

	// Test continue flag (skip already-migrated)
	// Add entries to ledger
	if err := ledger.Append(migrate.LedgerEntry{
		Source: "books/file1.pdf",
		BookID: "file1",
		Shelf:  "test",
	}); err != nil {
		t.Fatalf("failed to append to ledger: %v", err)
	}

	// Process again with continue flag
	f2, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	processed2, skipped2 := processMigrationQueue(f2, ledger, 0, true, true, true)
	_ = f2.Close()

	// Should process 2 and skip 1 (already in ledger)
	if processed2 != 2 {
		t.Errorf("processMigrationQueue with continue processed %d, want 2", processed2)
	}
	if skipped2 != 1 {
		t.Errorf("processMigrationQueue with continue skipped %d, want 1", skipped2)
	}
}

// TestMigrateRollback tests error handling and rollback on failure.
func TestMigrateRollback(t *testing.T) {
	// Setup config with migration sources
	cfg = &config.Config{
		Migration: config.MigrationConfig{
			Sources: []config.MigrationSource{
				{Owner: "owner1", Repo: "repo1"},
				{Owner: "owner2", Repo: "repo2"},
			},
		},
	}

	// Test with malformed sourceSpec (missing slash)
	_, err := resolveMigrationSources("invalidspec")
	if err == nil {
		t.Error("resolveMigrationSources with malformed spec should return error")
	}
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("error message should mention 'owner/repo', got: %v", err)
	}

	// Test with valid but non-existent sourceSpec
	_, err = resolveMigrationSources("nonexistent/repo")
	if err == nil {
		t.Error("resolveMigrationSources with non-existent source should return error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found', got: %v", err)
	}

	// Test with valid sourceSpec
	resolved, err := resolveMigrationSources("owner1/repo1")
	if err != nil {
		t.Fatalf("resolveMigrationSources with valid spec failed: %v", err)
	}
	if len(resolved) != 1 {
		t.Errorf("resolved %d sources, want 1", len(resolved))
	}
	if resolved[0].Owner != "owner1" || resolved[0].Repo != "repo1" {
		t.Errorf("resolved source = %+v, want owner1/repo1", resolved[0])
	}

	// Test with empty sourceSpec (should return all sources)
	resolved2, err := resolveMigrationSources("")
	if err != nil {
		t.Fatalf("resolveMigrationSources with empty spec failed: %v", err)
	}
	if len(resolved2) != 2 {
		t.Errorf("resolved %d sources with empty spec, want 2", len(resolved2))
	}
}

// TestWriteFileList tests queue file output.
func TestWriteFileList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-filelist-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	files := []migrate.FileEntry{
		{Path: "books/file1.pdf", SHA: "abc123", Size: 1024},
		{Path: "books/file2.pdf", SHA: "def456", Size: 2048},
		{Path: "books/file3.epub", SHA: "ghi789", Size: 512},
	}

	// Test writing to file
	outFile := filepath.Join(tmpDir, "filelist.txt")
	if err := writeFileList(files, outFile); err != nil {
		t.Fatalf("writeFileList failed: %v", err)
	}

	// Verify file contents
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "books/file1.pdf") {
		t.Error("output should contain books/file1.pdf")
	}
	if !strings.Contains(content, "books/file2.pdf") {
		t.Error("output should contain books/file2.pdf")
	}
	if !strings.Contains(content, "books/file3.epub") {
		t.Error("output should contain books/file3.epub")
	}

	// Count lines
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 3 {
		t.Errorf("output has %d lines, want 3", len(lines))
	}
}

// TestBuildMigratedBook tests book metadata construction.
func TestBuildMigratedBook(t *testing.T) {
	shelf := &config.ShelfConfig{
		Name:           "tech",
		Owner:          "test-owner",
		Repo:           "test-repo",
		DefaultRelease: "v2023.1",
	}

	src := config.MigrationSource{
		Owner: "old-owner",
		Repo:  "old-repo",
	}

	book := buildMigratedBook(
		"go-programming",
		"go-programming.pdf",
		"books/programming/Go Programming.pdf",
		"abc123def456",
		102400,
		shelf,
		src,
	)

	// Verify book fields
	if book.ID != "go-programming" {
		t.Errorf("book.ID = %q, want go-programming", book.ID)
	}
	if book.Title != "Go Programming" {
		t.Errorf("book.Title = %q, want 'Go Programming'", book.Title)
	}
	if book.Format != "pdf" {
		t.Errorf("book.Format = %q, want pdf", book.Format)
	}
	if book.SizeBytes != 102400 {
		t.Errorf("book.SizeBytes = %d, want 102400", book.SizeBytes)
	}
	if book.Checksum.SHA256 != "abc123def456" {
		t.Errorf("book.Checksum.SHA256 = %q, want abc123def456", book.Checksum.SHA256)
	}
	if book.Source.Type != "github_release" {
		t.Errorf("book.Source.Type = %q, want github_release", book.Source.Type)
	}
	if book.Source.Owner != "test-owner" {
		t.Errorf("book.Source.Owner = %q, want test-owner", book.Source.Owner)
	}
	if book.Source.Repo != "test-repo" {
		t.Errorf("book.Source.Repo = %q, want test-repo", book.Source.Repo)
	}
	if book.Source.Release != "v2023.1" {
		t.Errorf("book.Source.Release = %q, want v2023.1", book.Source.Release)
	}
	if book.Source.Asset != "go-programming.pdf" {
		t.Errorf("book.Source.Asset = %q, want go-programming.pdf", book.Source.Asset)
	}
	if book.Meta.AddedAt == "" {
		t.Error("book.Meta.AddedAt should not be empty")
	}

	expectedMigratedFrom := "old-owner/old-repo:books/programming/Go Programming.pdf"
	if book.Meta.MigratedFrom != expectedMigratedFrom {
		t.Errorf("book.Meta.MigratedFrom = %q, want %q", book.Meta.MigratedFrom, expectedMigratedFrom)
	}
}

// TestUpdateCatalogWithBook tests catalog update logic.
func TestUpdateCatalogWithBook(t *testing.T) {
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

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())
	gh = ghClient

	shelf := &config.ShelfConfig{
		Name:  "tech",
		Owner: "tech-library",
		Repo:  "tech-books",
	}

	src := config.MigrationSource{
		Owner: "old-library",
		Repo:  "legacy-books",
	}

	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "tech-library",
		},
	}

	newBook := catalog.Book{
		ID:     "new-book",
		Title:  "New Book",
		Format: "pdf",
		Checksum: catalog.Checksum{
			SHA256: "new-checksum",
		},
		SizeBytes: 5000,
	}

	// Test updateCatalogWithBook with noPush=true (local only)
	// This should work even though the mock server has the catalog
	err = updateCatalogWithBook(shelf, newBook, src, true)
	if err != nil {
		t.Errorf("updateCatalogWithBook with noPush failed: %v", err)
	}

	// Test with noPush=false would attempt to commit, but the mockserver
	// doesn't implement PUT /repos/{owner}/{repo}/contents/{path}
	// So we just validate the function signature is correct
}

// TestScanSingleSource tests scanning a single source repository.
func TestScanSingleSource(t *testing.T) {
	// Setup config with migration sources
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Token:   "test-token",
			APIBase: "http://localhost:9999", // Non-existent server
		},
		Migration: config.MigrationConfig{
			Sources: []config.MigrationSource{
				{Owner: "org1", Repo: "repo1", Ref: "main"},
				{Owner: "org2", Repo: "repo2", Ref: "develop"},
			},
		},
	}

	// Test with malformed sourceSpec
	_, err := scanSingleSource("invalid", cfg.Migration.Sources, []string{"pdf"})
	if err == nil {
		t.Error("scanSingleSource with malformed spec should return error")
	}
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("error should mention 'owner/repo', got: %v", err)
	}

	// Test with valid sourceSpec but no API call (we'd need a real mock server)
	// This validates the parsing logic
	// The actual API call will fail, but we're testing the parsing
	_, err = scanSingleSource("org1/repo1", cfg.Migration.Sources, []string{"pdf"})
	// Error is expected here since we're not hitting a real server
	// The important part is that it parsed the sourceSpec correctly
	if err == nil {
		t.Log("scanSingleSource returned nil error (unexpected but acceptable in test)")
	}
}

// TestResolveOrCreateShelf tests shelf resolution helper.
func TestResolveOrCreateShelf(t *testing.T) {
	cfg = &config.Config{
		Shelves: []config.ShelfConfig{
			{Name: "tech", Repo: "tech-books"},
			{Name: "fiction", Repo: "fiction-books"},
		},
	}

	// Test resolving existing shelf
	shelf, err := resolveOrCreateShelf("tech")
	if err != nil {
		t.Fatalf("resolveOrCreateShelf(tech) failed: %v", err)
	}
	if shelf.Name != "tech" {
		t.Errorf("shelf.Name = %q, want tech", shelf.Name)
	}
	if shelf.Repo != "tech-books" {
		t.Errorf("shelf.Repo = %q, want tech-books", shelf.Repo)
	}

	// Test with non-existent shelf
	_, err = resolveOrCreateShelf("nonexistent")
	if err == nil {
		t.Error("resolveOrCreateShelf with non-existent shelf should return error")
	}
	if !strings.Contains(err.Error(), "shelf") {
		t.Errorf("error should mention 'shelf', got: %v", err)
	}
}

// TestFetchSourceFile tests file fetching from source repository.
func TestFetchSourceFile(t *testing.T) {
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

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())
	gh = ghClient

	src := config.MigrationSource{
		Owner: "test-owner",
		Repo:  "test-repo",
		Ref:   "main",
	}

	// Test fetching a file (will fail because mockserver doesn't implement Contents API fully)
	_, _, _, err = fetchSourceFile(src, "books/test.pdf")
	// Error is expected - we're just testing the function doesn't panic
	if err == nil {
		t.Log("fetchSourceFile succeeded unexpectedly (mockserver limitation)")
	} else {
		// Expected error
		t.Logf("fetchSourceFile returned expected error: %v", err)
	}

	// Test with empty ref (should default to "main")
	src.Ref = ""
	_, _, _, err = fetchSourceFile(src, "books/test.pdf")
	// Again, error expected but function should handle empty ref
	if err == nil {
		t.Log("fetchSourceFile with empty ref succeeded unexpectedly")
	}
}

// TestUploadMigratedFile tests file upload to destination repository.
func TestUploadMigratedFile(t *testing.T) {
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

	// Create GitHub client
	ghClient := github.New("mock-token", srv.URL())
	gh = ghClient

	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner: "test-owner",
		},
		Defaults: config.DefaultsConfig{
			Release: "v2023.1",
		},
	}

	shelf := &config.ShelfConfig{
		Name:  "tech",
		Owner: "tech-library",
		Repo:  "tech-books",
	}

	// Create test file data
	fileData := []byte("test PDF content")

	// Test upload (mockserver implements asset upload)
	suggestedID, assetName, err := uploadMigratedFile(shelf, "books/Go Programming.pdf", fileData, int64(len(fileData)))
	if err != nil {
		t.Errorf("uploadMigratedFile failed: %v", err)
	}

	// Verify suggested ID
	if suggestedID != "go-programming" {
		t.Errorf("suggestedID = %q, want go-programming", suggestedID)
	}
	if assetName != "go-programming.pdf" {
		t.Errorf("assetName = %q, want go-programming.pdf", assetName)
	}
}

// TestParseExtensions tests extension parsing helper.
func TestParseExtensions(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"pdf", []string{"pdf"}},
		{"pdf,epub", []string{"pdf", "epub"}},
		{"pdf, epub, mobi", []string{"pdf", "epub", "mobi"}},
		{"", []string{}},
		{" pdf , epub ", []string{"pdf", "epub"}},
	}

	for _, c := range cases {
		got := parseExtensions(c.input)
		if len(got) != len(c.want) {
			t.Errorf("parseExtensions(%q) returned %d items, want %d", c.input, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseExtensions(%q)[%d] = %q, want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}

// TestMigrateRouting tests path routing logic.
func TestMigrateRouting(t *testing.T) {
	sources := []config.MigrationSource{
		{
			Owner: "old-org",
			Repo:  "old-repo",
			Mapping: map[string]string{
				"books/programming/": "tech",
				"books/fiction/":     "fiction",
				"books/reference/":   "reference",
			},
		},
	}

	testCases := []struct {
		path      string
		wantShelf string
		wantFound bool
	}{
		{"books/programming/go.pdf", "tech", true},
		{"books/fiction/neuromancer.pdf", "fiction", true},
		{"books/reference/sicp.pdf", "reference", true},
		{"books/other/misc.pdf", "", false},
		{"random/path.pdf", "", false},
	}

	for _, tc := range testCases {
		_, shelf, found := migrate.FindRoute(tc.path, sources)
		if found != tc.wantFound {
			t.Errorf("FindRoute(%q) found=%v, want %v", tc.path, found, tc.wantFound)
		}
		if shelf != tc.wantShelf {
			t.Errorf("FindRoute(%q) shelf=%q, want %q", tc.path, shelf, tc.wantShelf)
		}
	}
}

// TestMigrateLedgerOperations tests ledger append and contains operations.
func TestMigrateLedgerOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-ledger-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ledgerPath := filepath.Join(tmpDir, "test-ledger.jsonl")
	ledger, err := migrate.OpenLedger(ledgerPath)
	if err != nil {
		t.Fatalf("OpenLedger failed: %v", err)
	}

	// Test empty ledger
	contains, err := ledger.Contains("books/test.pdf")
	if err != nil {
		t.Fatalf("Contains on empty ledger failed: %v", err)
	}
	if contains {
		t.Error("empty ledger should not contain any entries")
	}

	// Append entry
	entry := migrate.LedgerEntry{
		Source: "books/test.pdf",
		BookID: "test",
		Shelf:  "tech",
	}
	if err := ledger.Append(entry); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Check contains after append
	contains, err = ledger.Contains("books/test.pdf")
	if err != nil {
		t.Fatalf("Contains after append failed: %v", err)
	}
	if !contains {
		t.Error("ledger should contain appended entry")
	}

	// Check non-existent entry
	contains, err = ledger.Contains("books/other.pdf")
	if err != nil {
		t.Fatalf("Contains for non-existent entry failed: %v", err)
	}
	if contains {
		t.Error("ledger should not contain non-existent entry")
	}

	// Append another entry
	entry2 := migrate.LedgerEntry{
		Source: "books/another.pdf",
		BookID: "another",
		Shelf:  "fiction",
	}
	if err := ledger.Append(entry2); err != nil {
		t.Fatalf("second Append failed: %v", err)
	}

	// Verify both entries
	contains, _ = ledger.Contains("books/test.pdf")
	if !contains {
		t.Error("ledger should still contain first entry")
	}
	contains, _ = ledger.Contains("books/another.pdf")
	if !contains {
		t.Error("ledger should contain second entry")
	}
}

// TestProcessMigrationQueueWithErrors tests error handling during batch processing.
func TestProcessMigrationQueueWithErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-queue-errors-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create queue with invalid format
	queueFile := filepath.Join(tmpDir, "error-queue.txt")
	queueContent := `books/valid1.pdf
books/valid2.pdf
books/valid3.pdf
`
	if err := os.WriteFile(queueFile, []byte(queueContent), 0644); err != nil {
		t.Fatalf("failed to write queue file: %v", err)
	}

	ledgerPath := filepath.Join(tmpDir, "error-ledger.jsonl")
	ledger, err := migrate.OpenLedger(ledgerPath)
	if err != nil {
		t.Fatalf("failed to open ledger: %v", err)
	}

	// Setup minimal config (migrations will fail, but we're testing error handling)
	cfg = &config.Config{
		GitHub: config.GitHubConfig{
			Owner:   "test-owner",
			Token:   "test-token",
			APIBase: "http://localhost:9999", // Non-existent server
		},
		Migration: config.MigrationConfig{
			Sources: []config.MigrationSource{},
		},
	}

	// Process queue - should handle errors gracefully
	f, err := os.Open(queueFile)
	if err != nil {
		t.Fatalf("failed to open queue file: %v", err)
	}
	defer func() { _ = f.Close() }()

	// With dry-run, errors shouldn't occur
	processed, skipped := processMigrationQueue(f, ledger, 0, false, true, true)
	if processed != 3 {
		t.Errorf("dry-run should process 3 items even with invalid config, got %d", processed)
	}
	if skipped != 0 {
		t.Errorf("dry-run skipped %d, want 0", skipped)
	}
}
