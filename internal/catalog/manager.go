package catalog

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/github"
)

// Manager provides high-level catalog operations.
// It centralizes the common pattern of load → parse → modify → marshal → commit.
type Manager struct {
	gh          *github.Client
	owner       string
	repo        string
	catalogPath string
}

// NewManager creates a new catalog manager.
func NewManager(gh *github.Client, owner, repo, catalogPath string) *Manager {
	return &Manager{
		gh:          gh,
		owner:       owner,
		repo:        repo,
		catalogPath: catalogPath,
	}
}

// Load retrieves and parses the catalog from GitHub.
// Returns an empty slice if the catalog doesn't exist (not an error).
func (m *Manager) Load() ([]Book, error) {
	data, _, err := m.gh.GetFileContent(m.owner, m.repo, m.catalogPath, "")
	if err != nil {
		if err.Error() == "not found" {
			return []Book{}, nil
		}
		return nil, fmt.Errorf("reading catalog: %w", err)
	}

	books, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing catalog: %w", err)
	}

	return books, nil
}

// Save marshals and commits the catalog to GitHub.
func (m *Manager) Save(books []Book, commitMsg string) error {
	data, err := Marshal(books)
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}

	if err := m.gh.CommitFile(m.owner, m.repo, m.catalogPath, data, commitMsg); err != nil {
		return fmt.Errorf("committing catalog: %w", err)
	}

	return nil
}

// Update loads the catalog, applies a modification function, and saves it.
// This is the most common pattern: load → modify → save.
//
// Example:
//
//	err := mgr.Update(func(books []Book) ([]Book, error) {
//	    return Append(books, newBook), nil
//	}, "add: new-book")
func (m *Manager) Update(fn func([]Book) ([]Book, error), commitMsg string) error {
	books, err := m.Load()
	if err != nil {
		return err
	}

	books, err = fn(books)
	if err != nil {
		return err
	}

	return m.Save(books, commitMsg)
}

// FindByID returns the book with the given ID, or nil if not found.
func (m *Manager) FindByID(id string) (*Book, error) {
	books, err := m.Load()
	if err != nil {
		return nil, err
	}

	for _, book := range books {
		if book.ID == id {
			return &book, nil
		}
	}

	return nil, nil
}

// Remove deletes a book by ID and saves the catalog.
// Returns the updated book list and whether the book was found.
func (m *Manager) Remove(bookID, commitMsg string) ([]Book, bool, error) {
	books, err := m.Load()
	if err != nil {
		return nil, false, err
	}

	books, found := Remove(books, bookID)

	if err := m.Save(books, commitMsg); err != nil {
		return nil, found, err
	}

	return books, found, nil
}

// Append adds a book and saves the catalog.
func (m *Manager) Append(book Book, commitMsg string) ([]Book, error) {
	books, err := m.Load()
	if err != nil {
		return nil, err
	}

	books = Append(books, book)

	if err := m.Save(books, commitMsg); err != nil {
		return nil, err
	}

	return books, nil
}
