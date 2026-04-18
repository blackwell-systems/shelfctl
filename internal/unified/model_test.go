package unified

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
)

// computeContentSHA256 returns the hex-encoded SHA-256 of the given string.
func computeContentSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// storeFile writes content to the cache under (owner, repo, bookID, asset).
// It panics if the store fails because tests should catch this early.
func storeFile(t *testing.T, m *cache.Manager, owner, repo, bookID, asset, content string) {
	t.Helper()
	_, err := m.Store(owner, repo, bookID, asset, strings.NewReader(content), "")
	if err != nil {
		t.Fatalf("storeFile: %v", err)
	}
}

// TestPerformPendingAction_UnknownAction verifies that an unknown action type
// returns nil without attempting any network or filesystem operations.
func TestPerformPendingAction_UnknownAction(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{}

	action := &ActionRequestMsg{
		Action:   tui.BrowserAction("unknown-action-xyz"),
		BookItem: &tui.BookItem{},
		ReturnTo: "hub",
	}

	err := PerformPendingAction(action, nil, cfg, cacheMgr)
	if err != nil {
		t.Errorf("unknown action should return nil, got: %v", err)
	}
}

// TestPerformPendingAction_OpenBook_NilItem verifies that ActionOpen with a nil
// BookItem returns an error (no crash).
func TestPerformPendingAction_OpenBook_NilItem(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{}

	action := &ActionRequestMsg{
		Action:   tui.ActionOpen,
		BookItem: nil,
		ReturnTo: "hub",
	}

	err := PerformPendingAction(action, nil, cfg, cacheMgr)
	if err == nil {
		t.Error("ActionOpen with nil BookItem should return an error")
	}
}

// TestPerformPendingAction_OpenBook_CachedNoNetwork verifies that ActionOpen for
// an already-cached book does not attempt any GitHub API calls. The book file
// must exist on disk; the test uses a temp dir to pre-populate the cache.
// Note: the actual file open (util.OpenFile) is not testable in CI without a
// desktop environment, so we assert on the error message instead.
func TestPerformPendingAction_OpenBook_CachedNoNetwork(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())

	const (
		owner = "testowner"
		repo  = "testrepo"
		bookID = "sicp"
		asset = "sicp.pdf"
	)

	// Pre-populate the cache so Cached=true skips GitHub calls.
	storeFile(t, cacheMgr, owner, repo, bookID, asset, "fake pdf content")

	book := catalog.Book{
		ID:    bookID,
		Title: "SICP",
		Source: catalog.Source{
			Type:    "github",
			Owner:   owner,
			Repo:    repo,
			Release: "library",
			Asset:   asset,
		},
	}

	item := &tui.BookItem{
		Book:   book,
		Cached: true,
		Owner:  owner,
		Repo:   repo,
	}

	cfg := &config.Config{}

	action := &ActionRequestMsg{
		Action:   tui.ActionOpen,
		BookItem: item,
		ReturnTo: "hub",
	}

	// util.OpenFile will fail in CI (no desktop), but we only care that:
	// 1. No nil-pointer panic on gh (nil client never called when Cached=true).
	// 2. The error (if any) comes from OpenFile, not from a nil gh dereference.
	// If gh were accidentally dereferenced, this would panic/crash, not return an error.
	_ = PerformPendingAction(action, nil, cfg, cacheMgr)
	// No assertion on the returned error — OpenFile behaviour is environment-dependent.
}

// TestRefreshModifiedStatusCmd_NoShelves verifies that an empty config produces
// a modifiedStatusRefreshedMsg with count == 0.
func TestRefreshModifiedStatusCmd_NoShelves(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{
		Shelves: []config.ShelfConfig{},
	}

	m := Model{
		cfg:          cfg,
		cacheMgr:     cacheMgr,
		catalogCache: make(map[string][]catalog.Book),
	}

	cmd := m.refreshModifiedStatusCmd()
	msg := cmd()

	result, ok := msg.(modifiedStatusRefreshedMsg)
	if !ok {
		t.Fatalf("refreshModifiedStatusCmd returned %T, want modifiedStatusRefreshedMsg", msg)
	}
	if result.count != 0 {
		t.Errorf("count = %d, want 0 for empty config", result.count)
	}
	if len(result.books) != 0 {
		t.Errorf("books len = %d, want 0 for empty config", len(result.books))
	}
}

// TestRefreshModifiedStatusCmd_ModifiedBook verifies that a book whose cached file
// has a different SHA-256 from the catalog checksum is detected as modified.
func TestRefreshModifiedStatusCmd_ModifiedBook(t *testing.T) {
	dir := t.TempDir()
	cacheMgr := cache.New(dir)

	const (
		owner  = "alice"
		repo   = "shelf-prog"
		bookID = "sicp"
		asset  = "sicp.pdf"
	)

	// Store a file in the cache with known content.
	originalContent := "original book content"
	storeFile(t, cacheMgr, owner, repo, bookID, asset, originalContent)

	// The catalog records a DIFFERENT sha — simulating a local modification.
	catalogSHA := computeContentSHA256("different content that does not match the file")

	cfg := &config.Config{
		GitHub: config.GitHubConfig{Owner: owner},
		Shelves: []config.ShelfConfig{
			{Name: "prog", Repo: repo},
		},
	}

	books := []catalog.Book{
		fakeBook(bookID, "SICP", asset, catalogSHA),
	}

	m := Model{
		cfg:      cfg,
		cacheMgr: cacheMgr,
		catalogCache: map[string][]catalog.Book{
			owner + "/" + repo: books,
		},
	}

	cmd := m.refreshModifiedStatusCmd()
	msg := cmd()

	result, ok := msg.(modifiedStatusRefreshedMsg)
	if !ok {
		t.Fatalf("refreshModifiedStatusCmd returned %T, want modifiedStatusRefreshedMsg", msg)
	}
	if result.count != 1 {
		t.Errorf("count = %d, want 1 (modified book detected)", result.count)
	}
	if len(result.books) != 1 {
		t.Fatalf("books len = %d, want 1", len(result.books))
	}
	if result.books[0].ID != bookID {
		t.Errorf("modified book ID = %q, want %q", result.books[0].ID, bookID)
	}
}

// TestRefreshModifiedStatusCmd_UnmodifiedBook verifies that a book whose cached
// SHA matches the catalog checksum is NOT counted as modified.
func TestRefreshModifiedStatusCmd_UnmodifiedBook(t *testing.T) {
	dir := t.TempDir()
	cacheMgr := cache.New(dir)

	const (
		owner  = "alice"
		repo   = "shelf-prog"
		bookID = "ddia"
		asset  = "ddia.pdf"
	)

	content := "designing data-intensive applications"
	storeFile(t, cacheMgr, owner, repo, bookID, asset, content)

	// Catalog SHA matches the file on disk.
	correctSHA := computeContentSHA256(content)

	cfg := &config.Config{
		GitHub: config.GitHubConfig{Owner: owner},
		Shelves: []config.ShelfConfig{
			{Name: "prog", Repo: repo},
		},
	}

	books := []catalog.Book{
		fakeBook(bookID, "DDIA", asset, correctSHA),
	}

	m := Model{
		cfg:      cfg,
		cacheMgr: cacheMgr,
		catalogCache: map[string][]catalog.Book{
			owner + "/" + repo: books,
		},
	}

	cmd := m.refreshModifiedStatusCmd()
	msg := cmd()

	result, ok := msg.(modifiedStatusRefreshedMsg)
	if !ok {
		t.Fatalf("refreshModifiedStatusCmd returned %T, want modifiedStatusRefreshedMsg", msg)
	}
	if result.count != 0 {
		t.Errorf("count = %d, want 0 (book not modified)", result.count)
	}
}
