package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// OrphanEntry represents a cached file with no corresponding catalog entry
type OrphanEntry struct {
	Path     string // Full path to cached file
	Repo     string // Repository name
	Filename string // Asset filename
	Size     int64  // File size in bytes
}

// OrphanReport contains all detected orphaned cache entries
type OrphanReport struct {
	Entries    []OrphanEntry
	TotalSize  int64
	TotalCount int
}

// ShelfCatalog represents a shelf's catalog for orphan detection
type ShelfCatalog struct {
	Owner string
	Repo  string
	Books []catalog.Book
}

// DetectOrphans finds cached files that don't correspond to any book in the provided catalogs
func (m *Manager) DetectOrphans(shelves []ShelfCatalog) (OrphanReport, error) {
	report := OrphanReport{
		Entries: []OrphanEntry{},
	}

	// Build a map of all known assets: repo -> {assetFilename -> true}
	knownAssets := make(map[string]map[string]bool)
	for _, shelf := range shelves {
		if _, exists := knownAssets[shelf.Repo]; !exists {
			knownAssets[shelf.Repo] = make(map[string]bool)
		}
		for _, book := range shelf.Books {
			if book.Source.Asset != "" {
				knownAssets[shelf.Repo][book.Source.Asset] = true
			}
		}
	}

	// Walk the cache directory
	baseDir := m.baseDir
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		// Cache doesn't exist - no orphans
		return report, nil
	}

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip temporary files
		if strings.HasSuffix(path, ".tmp") {
			return nil
		}

		// Skip cover files (they're associated with assets, cleaned up automatically)
		if strings.Contains(path, "/.covers/") {
			return nil
		}

		// Determine repo and filename from path
		// Path structure: <baseDir>/<repo>/<assetFilename>
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil // Skip files we can't parse
		}

		parts := strings.SplitN(relPath, string(filepath.Separator), 2)
		if len(parts) != 2 {
			return nil // Skip files not in expected structure
		}

		repo := parts[0]
		filename := parts[1]

		// Check if this asset is known in any catalog
		if repoAssets, exists := knownAssets[repo]; exists {
			if repoAssets[filename] {
				return nil // Asset is referenced, not an orphan
			}
		}

		// This is an orphan
		report.Entries = append(report.Entries, OrphanEntry{
			Path:     path,
			Repo:     repo,
			Filename: filename,
			Size:     info.Size(),
		})
		report.TotalSize += info.Size()
		report.TotalCount++

		return nil
	})

	if err != nil {
		return report, fmt.Errorf("walking cache directory: %w", err)
	}

	return report, nil
}

// ClearOrphans deletes all files in the provided orphan report
// Returns the number of files successfully deleted
func (m *Manager) ClearOrphans(report OrphanReport) (int, error) {
	deleted := 0
	var lastErr error

	for _, entry := range report.Entries {
		if err := os.Remove(entry.Path); err != nil {
			lastErr = err
			continue
		}
		deleted++
	}

	if lastErr != nil {
		return deleted, fmt.Errorf("some files failed to delete: %w", lastErr)
	}

	return deleted, nil
}
