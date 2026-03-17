package catalog_test

import (
	"errors"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

type mockGitHubClient struct {
	content        []byte
	contentErr     error
	commitCalls    int
	commitErr      error
	lastCommitData []byte
	lastCommitMsg  string
}

func (m *mockGitHubClient) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	return m.content, "", m.contentErr
}

func (m *mockGitHubClient) CommitFile(owner, repo, path string, data []byte, message string) error {
	m.commitCalls++
	m.lastCommitData = data
	m.lastCommitMsg = message
	return m.commitErr
}

func newMgr(mock *mockGitHubClient) *catalog.Manager {
	return catalog.NewManager(mock, "owner", "repo", "catalog.yaml")
}

var sampleCatalog = []byte(`
- id: book1
  title: "Book One"
  format: pdf
- id: book2
  title: "Book Two"
  format: epub
`)

// --- Load ---

func TestLoad_Happy(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	books, err := newMgr(mock).Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}
}

func TestLoad_NotFound(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("not found")}
	books, err := newMgr(mock).Load()
	if err != nil {
		t.Fatalf("not-found should return empty slice, got error: %v", err)
	}
	if len(books) != 0 {
		t.Fatalf("expected 0 books, got %d", len(books))
	}
}

func TestLoad_OtherError(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("network error")}
	// "network error" != "not found", so Load should propagate it.
	// But the current implementation only special-cases "not found".
	// The mock returns "network error" so Load should return an error.
	_, err := newMgr(mock).Load()
	if err == nil {
		t.Fatal("expected error for non-not-found failure, got nil")
	}
}

// --- Save ---

func TestSave_Happy(t *testing.T) {
	mock := &mockGitHubClient{}
	books := []catalog.Book{{ID: "b1", Title: "T", Format: "pdf"}}
	err := newMgr(mock).Save(books, "save msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.commitCalls != 1 {
		t.Fatalf("expected 1 commit call, got %d", mock.commitCalls)
	}
	if mock.lastCommitMsg != "save msg" {
		t.Errorf("commit msg = %q, want %q", mock.lastCommitMsg, "save msg")
	}
}

func TestSave_CommitError(t *testing.T) {
	mock := &mockGitHubClient{commitErr: errors.New("commit failed")}
	books := []catalog.Book{{ID: "b1", Title: "T", Format: "pdf"}}
	err := newMgr(mock).Save(books, "msg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Update ---

func TestUpdate_Happy(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	called := false
	err := newMgr(mock).Update(func(books []catalog.Book) ([]catalog.Book, error) {
		called = true
		return append(books, catalog.Book{ID: "book3", Title: "Book Three", Format: "pdf"}), nil
	}, "update msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("update fn was not called")
	}
	if mock.commitCalls != 1 {
		t.Errorf("expected 1 commit, got %d", mock.commitCalls)
	}
}

func TestUpdate_LoadError(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("network error")}
	err := newMgr(mock).Update(func(books []catalog.Book) ([]catalog.Book, error) {
		return books, nil
	}, "msg")
	if err == nil {
		t.Fatal("expected error from load, got nil")
	}
	if mock.commitCalls != 0 {
		t.Errorf("should not commit when load fails, got %d", mock.commitCalls)
	}
}

func TestUpdate_FnError(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	err := newMgr(mock).Update(func(books []catalog.Book) ([]catalog.Book, error) {
		return nil, errors.New("fn failed")
	}, "msg")
	if err == nil {
		t.Fatal("expected fn error to propagate, got nil")
	}
	if mock.commitCalls != 0 {
		t.Errorf("should not commit when fn fails, got %d", mock.commitCalls)
	}
}

// --- FindByID ---

func TestFindByID_Found(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	book, err := newMgr(mock).FindByID("book1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book == nil {
		t.Fatal("expected book, got nil")
	}
	if book.ID != "book1" {
		t.Errorf("got ID %q, want %q", book.ID, "book1")
	}
}

func TestFindByID_NotFound(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	book, err := newMgr(mock).FindByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if book != nil {
		t.Errorf("expected nil, got %+v", book)
	}
}

func TestFindByID_LoadError(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("network error")}
	_, err := newMgr(mock).FindByID("book1")
	if err == nil {
		t.Fatal("expected error from load, got nil")
	}
}

// --- Remove ---

func TestRemove_Found(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	books, found, err := newMgr(mock).Remove("book1", "remove: book1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected found=true")
	}
	if len(books) != 1 || books[0].ID != "book2" {
		t.Errorf("unexpected books after remove: %+v", books)
	}
	if mock.commitCalls != 1 {
		t.Errorf("expected 1 commit, got %d", mock.commitCalls)
	}
}

func TestRemove_NotFound(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	books, found, err := newMgr(mock).Remove("nonexistent", "remove: nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected found=false")
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books unchanged, got %d", len(books))
	}
	if mock.commitCalls != 0 {
		t.Errorf("should not commit when not found, got %d", mock.commitCalls)
	}
}

func TestRemove_LoadError(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("network error")}
	_, _, err := newMgr(mock).Remove("book1", "remove: book1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.commitCalls != 0 {
		t.Errorf("should not commit when load fails, got %d", mock.commitCalls)
	}
}

func TestRemove_SaveError(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog, commitErr: errors.New("commit failed")}
	_, found, err := newMgr(mock).Remove("book1", "remove: book1")
	if err == nil {
		t.Fatal("expected error from save, got nil")
	}
	if !found {
		t.Error("found should be true even when save fails")
	}
}

// --- Append ---

func TestAppend_AddsBook(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	newBook := catalog.Book{ID: "book3", Title: "Book Three", Format: "mobi"}
	books, err := newMgr(mock).Append(newBook, "add: book3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(books) != 3 {
		t.Fatalf("expected 3 books, got %d", len(books))
	}
	if mock.commitCalls != 1 {
		t.Errorf("expected 1 commit, got %d", mock.commitCalls)
	}
	if mock.lastCommitMsg != "add: book3" {
		t.Errorf("commit msg = %q, want %q", mock.lastCommitMsg, "add: book3")
	}
}

func TestManager_Append_ReplacesExisting(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog}
	updated := catalog.Book{ID: "book1", Title: "Book One Updated", Format: "epub"}
	books, err := newMgr(mock).Append(updated, "update: book1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Append replaces if same ID exists
	found := false
	for _, b := range books {
		if b.ID == "book1" {
			found = true
			if b.Title != "Book One Updated" {
				t.Errorf("title not updated: got %q", b.Title)
			}
		}
	}
	if !found {
		t.Error("book1 not found after append/replace")
	}
}

func TestAppend_LoadError(t *testing.T) {
	mock := &mockGitHubClient{contentErr: errors.New("network error")}
	_, err := newMgr(mock).Append(catalog.Book{ID: "b"}, "msg")
	if err == nil {
		t.Fatal("expected error from load, got nil")
	}
}

func TestAppend_SaveError(t *testing.T) {
	mock := &mockGitHubClient{content: sampleCatalog, commitErr: errors.New("commit failed")}
	_, err := newMgr(mock).Append(catalog.Book{ID: "book3"}, "msg")
	if err == nil {
		t.Fatal("expected error from save, got nil")
	}
}
