package cache

import (
	"os"
	"path/filepath"
)

// MigrateOwnerLayout moves cache entries from the old 2-level layout
// (<baseDir>/<repo>/<asset>) to the new 3-level owner-scoped layout
// (<baseDir>/<owner>/<repo>/<asset>).
//
// It checks each configured repo: if <baseDir>/<repo> exists as a directory
// but <baseDir>/<owner>/<repo> does not, the directory is moved. This is a
// one-time, idempotent migration that runs on startup.
func (m *Manager) MigrateOwnerLayout(shelves []ShelfMigrationEntry) {
	for _, s := range shelves {
		oldPath := filepath.Join(m.baseDir, s.Repo)
		newPath := filepath.Join(m.baseDir, s.Owner, s.Repo)

		oldInfo, err := os.Stat(oldPath)
		if err != nil || !oldInfo.IsDir() {
			continue // old-layout dir doesn't exist
		}

		if _, err := os.Stat(newPath); err == nil {
			continue // new-layout dir already exists, don't clobber
		}

		// Create owner directory and move repo dir under it
		if err := os.MkdirAll(filepath.Join(m.baseDir, s.Owner), 0750); err != nil {
			continue
		}
		_ = os.Rename(oldPath, newPath)
	}
}

// ShelfMigrationEntry holds the owner and repo needed for cache layout migration.
type ShelfMigrationEntry struct {
	Owner string
	Repo  string
}
