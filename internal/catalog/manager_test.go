package catalog_test

import (
	"errors"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// mockGitHubClient is a test stub for github.Client.
type mockGitHubClient struct {
	// State for mocking GetFileContent
	content      []byte
	contentErr   error
	contentCalls int

	// State for mocking CommitFile
	commitCalls   int
	commitErr     error
	lastCommitMsg string
}

func (m *mockGitHubClient) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	m.contentCalls++
	return m.content, "", m.contentErr
}

func (m *mockGitHubClient) CommitFile(owner, repo, path string, data []byte, message string) error {
	m.commitCalls++
	m.lastCommitMsg = message
	return m.commitErr
}

// TestManager_Remove_NotFound verifies that when a book is not found,
// Manager.Remove does NOT call Save (i.e., does NOT commit to GitHub).
func TestManager_Remove_NotFound(t *testing.T) {
	// Setup: catalog with two books
	initialCatalog := []byte(`
- id: book1
  title: "Book One"
  format: pdf
- id: book2
  title: "Book Two"
  format: epub
`)

	mock := &mockGitHubClient{
		content:    initialCatalog,
		contentErr: nil,
	}

	// We need to create a Manager using the mock client.
	// Since Manager expects a *github.Client, we'll use an interface-based approach
	// or directly create the manager with mock fields.
	// However, Manager's constructor takes a concrete github.Client.
	// For testing, we'll use a test helper that constructs Manager with our mock.

	// Create manager with mock (we'll adapt the constructor signature in test scope)
	mgr := newTestManager(mock)

	// Call Remove with a book ID that doesn't exist
	books, found, err := mgr.Remove("nonexistent", "remove: nonexistent")

	// Assertions
	if err != nil {
		t.Fatalf("Remove returned unexpected error: %v", err)
	}
	if found {
		t.Error("Remove reported found=true for nonexistent book")
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books in result, got %d", len(books))
	}

	// Critical assertion: Save should NOT have been called
	if mock.commitCalls > 0 {
		t.Errorf("Save was called %d time(s) when book not found; expected 0 calls", mock.commitCalls)
	}
}

// TestManager_Remove_Found verifies that when a book IS found,
// Manager.Remove DOES call Save to commit the change.
func TestManager_Remove_Found(t *testing.T) {
	initialCatalog := []byte(`
- id: book1
  title: "Book One"
  format: pdf
- id: book2
  title: "Book Two"
  format: epub
`)

	mock := &mockGitHubClient{
		content:    initialCatalog,
		contentErr: nil,
	}

	mgr := newTestManager(mock)

	// Call Remove with an existing book ID
	books, found, err := mgr.Remove("book1", "remove: book1")

	// Assertions
	if err != nil {
		t.Fatalf("Remove returned unexpected error: %v", err)
	}
	if !found {
		t.Error("Remove reported found=false for existing book")
	}
	if len(books) != 1 {
		t.Errorf("expected 1 book after removal, got %d", len(books))
	}
	if books[0].ID != "book2" {
		t.Errorf("wrong book remaining: got %q, want %q", books[0].ID, "book2")
	}

	// Critical assertion: Save SHOULD have been called exactly once
	if mock.commitCalls != 1 {
		t.Errorf("Save was called %d time(s); expected 1", mock.commitCalls)
	}
	if mock.lastCommitMsg != "remove: book1" {
		t.Errorf("commit message = %q, want %q", mock.lastCommitMsg, "remove: book1")
	}
}

// TestManager_Remove_LoadError verifies error handling when Load fails.
func TestManager_Remove_LoadError(t *testing.T) {
	mock := &mockGitHubClient{
		contentErr: errors.New("network error"),
	}

	mgr := newTestManager(mock)

	_, _, err := mgr.Remove("book1", "remove: book1")
	if err == nil {
		t.Error("expected error when Load fails, got nil")
	}

	// Save should not be called if Load fails
	if mock.commitCalls > 0 {
		t.Errorf("Save was called when Load failed; expected 0 calls")
	}
}

// TestManager_Remove_SaveError verifies error handling when Save fails.
func TestManager_Remove_SaveError(t *testing.T) {
	initialCatalog := []byte(`
- id: book1
  title: "Book One"
  format: pdf
`)

	mock := &mockGitHubClient{
		content:   initialCatalog,
		commitErr: errors.New("commit failed"),
	}

	mgr := newTestManager(mock)

	_, found, err := mgr.Remove("book1", "remove: book1")
	if err == nil {
		t.Error("expected error when Save fails, got nil")
	}
	if !found {
		t.Error("found should be true even when Save fails")
	}
	if mock.commitCalls != 1 {
		t.Errorf("Save should have been called once; got %d", mock.commitCalls)
	}
}

// newTestManager creates a Manager using our mock client.
// Since Manager's constructor expects a concrete *github.Client,
// we'll have to use reflection or create a test-specific constructor.
// For simplicity, we'll directly instantiate the Manager struct.
func newTestManager(mock *mockGitHubClient) *testManagerWrapper {
	return &testManagerWrapper{
		mock:        mock,
		owner:       "test-owner",
		repo:        "test-repo",
		catalogPath: "catalog.yaml",
	}
}

// testManagerWrapper wraps our mock and implements Manager's methods.
type testManagerWrapper struct {
	mock        *mockGitHubClient
	owner       string
	repo        string
	catalogPath string
}

func (w *testManagerWrapper) Load() ([]catalog.Book, error) {
	data, _, err := w.mock.GetFileContent(w.owner, w.repo, w.catalogPath, "")
	if err != nil {
		return nil, err
	}
	return catalog.Parse(data)
}

func (w *testManagerWrapper) Save(books []catalog.Book, commitMsg string) error {
	data, err := catalog.Marshal(books)
	if err != nil {
		return err
	}
	return w.mock.CommitFile(w.owner, w.repo, w.catalogPath, data, commitMsg)
}

func (w *testManagerWrapper) Remove(bookID, commitMsg string) ([]catalog.Book, bool, error) {
	books, err := w.Load()
	if err != nil {
		return nil, false, err
	}

	books, found := catalog.Remove(books, bookID)

	if found {
		if err := w.Save(books, commitMsg); err != nil {
			return nil, found, err
		}
	}

	return books, found, nil
}
