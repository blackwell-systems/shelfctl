package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Store writes r to the cache path for the given coordinates, verifying
// the sha256 checksum after write if expectedSHA256 is non-empty.
// Returns the final file path.
func (m *Manager) Store(owner, repo, bookID, assetFilename string, r io.Reader, expectedSHA256 string) (string, error) {
	if err := m.EnsureDir(owner, repo, bookID); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	destPath := m.Path(owner, repo, bookID, assetFilename)
	tmpPath := destPath + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("writing to cache: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	if err := VerifyFile(tmpPath, expectedSHA256); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	// Extract cover thumbnail for PDFs (best-effort, silently skips on failure)
	if isPDF(assetFilename) {
		_ = m.ExtractCover(repo, bookID, destPath)
	}

	return destPath, nil
}

// isPDF checks if the filename indicates a PDF file
func isPDF(filename string) bool {
	ext := filepath.Ext(strings.ToLower(filename))
	return ext == ".pdf"
}

// HasBeenModified checks if the cached file's SHA256 differs from the expected value.
// Returns true if the file has been modified locally (e.g., annotations added).
func (m *Manager) HasBeenModified(owner, repo, bookID, assetFilename, expectedSHA256 string) bool {
	if !m.Exists(owner, repo, bookID, assetFilename) {
		return false
	}

	path := m.Path(owner, repo, bookID, assetFilename)
	if err := VerifyFile(path, expectedSHA256); err != nil {
		return true // Checksum mismatch means modified
	}

	return false
}
