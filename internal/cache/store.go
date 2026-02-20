package cache

import (
	"fmt"
	"io"
	"os"
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
	return destPath, nil
}
