package cache

import (
	"os"
	"path/filepath"
)

// Manager handles the local file cache.
type Manager struct {
	baseDir string
}

// New creates a cache Manager rooted at baseDir.
func New(baseDir string) *Manager {
	return &Manager{baseDir: baseDir}
}

// Path returns the full cache path for a given shelf repo and book.
// Layout: <baseDir>/<repo>/<assetFilename>
func (m *Manager) Path(owner, repo, bookID, assetFilename string) string {
	return filepath.Join(m.baseDir, repo, assetFilename)
}

// Exists reports whether the cached file exists.
func (m *Manager) Exists(owner, repo, bookID, assetFilename string) bool {
	_, err := os.Stat(m.Path(owner, repo, bookID, assetFilename))
	return err == nil
}

// EnsureDir creates all intermediate directories for a cache path.
func (m *Manager) EnsureDir(owner, repo, bookID string) error {
	dir := filepath.Join(m.baseDir, repo)
	return os.MkdirAll(dir, 0750)
}

// Remove deletes the cached file if it exists.
func (m *Manager) Remove(owner, repo, bookID, assetFilename string) error {
	path := m.Path(owner, repo, bookID, assetFilename)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
