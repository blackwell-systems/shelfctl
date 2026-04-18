package unified

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghpkg "github.com/blackwell-systems/shelfctl/internal/github"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestServer creates a fake httptest.Server and a *ghpkg.Client pointed at it.
// handler is called for every request.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *ghpkg.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := ghpkg.New("test-token", srv.URL)
	return srv, c
}

// fakeBook creates a minimal catalog.Book for use in tests.
func fakeBook(id, title, asset, sha string) catalog.Book {
	return catalog.Book{
		ID:    id,
		Title: title,
		Checksum: catalog.Checksum{
			SHA256: sha,
		},
		Source: catalog.Source{
			Type:  "github",
			Asset: asset,
		},
	}
}

// TestSyncAllModel_NewSyncAllModel verifies that a freshly created SyncAllModel
// starts in the detect phase with no books and reasonable zero-values.
func TestSyncAllModel_NewSyncAllModel(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{}

	m := NewSyncAllModel(nil, cfg, cacheMgr)

	if m.phase != syncAllDetecting {
		t.Errorf("initial phase = %v, want syncAllDetecting (%v)", m.phase, syncAllDetecting)
	}
	if len(m.books) != 0 {
		t.Errorf("initial books len = %d, want 0", len(m.books))
	}
	if m.current != 0 {
		t.Errorf("initial current = %d, want 0", m.current)
	}
	if m.synced != 0 {
		t.Errorf("initial synced = %d, want 0", m.synced)
	}
	if len(m.errors) != 0 {
		t.Errorf("initial errors len = %d, want 0", len(m.errors))
	}
	if m.autoMode {
		t.Error("initial autoMode should be false")
	}
}

// TestSyncAllModel_DetectAsync_ParseError verifies that when the catalog YAML is
// malformed, detectAsync surfaces a syncDetectErrorMsg (not a panic or silent skip).
// This tests the M8 fix: catalog parse errors are propagated, not swallowed.
func TestSyncAllModel_DetectAsync_ParseError(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())

	cfg := &config.Config{
		GitHub: config.GitHubConfig{Owner: "testowner"},
		Shelves: []config.ShelfConfig{
			{Name: "prog", Repo: "shelf-prog"},
		},
	}

	// Serve malformed YAML encoded as base64 (GitHub API returns base64-encoded content).
	malformedYAML := "\t: this is: not: valid: yaml: [[["
	encoded := base64.StdEncoding.EncodeToString([]byte(malformedYAML))

	_, ghClient := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"name":"catalog.yml","sha":"deadbeef","size":%d,"encoding":"base64","content":%q}`,
			len(malformedYAML), encoded)
	})

	m := SyncAllModel{
		phase:    syncAllDetecting,
		gh:       ghClient,
		cfg:      cfg,
		cacheMgr: cacheMgr,
	}

	// Run detectAsync synchronously by invoking the returned Tea command.
	cmd := m.detectAsync()
	msg := cmd()

	errMsg, ok := msg.(syncDetectErrorMsg)
	if !ok {
		t.Fatalf("detectAsync returned %T, want syncDetectErrorMsg", msg)
	}
	if errMsg.err == nil {
		t.Error("syncDetectErrorMsg.err should be non-nil for malformed catalog")
	}
}

// TestSyncAllModel_Update_Confirm verifies that pressing "enter" in the
// confirming phase transitions the model to the processing phase.
func TestSyncAllModel_Update_Confirm(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{}

	m := NewSyncAllModel(nil, cfg, cacheMgr)

	// Manually put the model into the confirming phase with a fake book entry.
	m.phase = syncAllConfirming
	m.books = []syncEntry{
		{
			book: fakeBook("book-1", "Book One", "asset.pdf", "abc123"),
		},
	}

	// Send "enter" key.
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(keyMsg)

	if updated.phase != syncAllProcessing {
		t.Errorf("after enter, phase = %v, want syncAllProcessing (%v)", updated.phase, syncAllProcessing)
	}
	if updated.current != 0 {
		t.Errorf("after enter, current = %d, want 0", updated.current)
	}
}

// TestSyncAllModel_Update_DetectedMsg_EmptyBooks verifies that receiving a
// syncDetectedMsg with zero books advances directly to syncAllDone.
func TestSyncAllModel_Update_DetectedMsg_EmptyBooks(t *testing.T) {
	cacheMgr := cache.New(t.TempDir())
	cfg := &config.Config{}

	m := NewSyncAllModel(nil, cfg, cacheMgr)
	m.phase = syncAllDetecting

	updated, _ := m.Update(syncDetectedMsg{books: nil})

	if updated.phase != syncAllDone {
		t.Errorf("empty detected books phase = %v, want syncAllDone (%v)", updated.phase, syncAllDone)
	}
}
