package cache

import (
	"io"
	"os"
	"path/filepath"
)

// StoreCatalogCover saves a catalog cover image from a reader to the cache.
// Returns the path to the cached cover, or empty string on failure.
// Catalog covers are stored separately from extracted thumbnails.
func (m *Manager) StoreCatalogCover(owner, repo, bookID string, r io.Reader) string {
	// Ensure .covers directory exists
	coversDir := filepath.Join(m.baseDir, owner, repo, ".covers")
	if err := os.MkdirAll(coversDir, 0750); err != nil {
		return ""
	}

	// Path for catalog cover: <cache>/<repo>/.covers/<book-id>-catalog.jpg
	coverPath := filepath.Join(coversDir, bookID+"-catalog.jpg")

	// Remove existing catalog cover if present
	_ = os.Remove(coverPath)

	// Write cover to disk
	f, err := os.Create(coverPath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		_ = os.Remove(coverPath)
		return ""
	}

	return coverPath
}

// CatalogCoverPath returns the path where a catalog cover would be stored.
func (m *Manager) CatalogCoverPath(owner, repo, bookID string) string {
	return filepath.Join(m.baseDir, owner, repo, ".covers", bookID+"-catalog.jpg")
}

// HasCatalogCover checks if a catalog cover exists for the given book.
func (m *Manager) HasCatalogCover(owner, repo, bookID string) bool {
	_, err := os.Stat(m.CatalogCoverPath(owner, repo, bookID))
	return err == nil
}

// RemoveCatalogCover deletes the catalog cover for a book if it exists.
func (m *Manager) RemoveCatalogCover(owner, repo, bookID string) error {
	path := m.CatalogCoverPath(owner, repo, bookID)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetCoverPath returns the best available cover path for a book.
// Priority: catalog cover (user-curated) > extracted thumbnail > empty string.
func (m *Manager) GetCoverPath(owner, repo, bookID string) string {
	// Check for catalog cover first (user-curated, higher priority)
	if m.HasCatalogCover(owner, repo, bookID) {
		return m.CatalogCoverPath(owner, repo, bookID)
	}

	// Fall back to extracted thumbnail
	if m.HasCover(owner, repo, bookID) {
		return m.CoverPath(owner, repo, bookID)
	}

	return ""
}
