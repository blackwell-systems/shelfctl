package cache

import (
	"os"
	"os/exec"
	"path/filepath"
)

// ExtractCover extracts the first page of a PDF as a JPEG thumbnail.
// Returns the path to the generated cover image, or empty string on failure.
// Only one cover exists per book - overwrites if already present.
func (m *Manager) ExtractCover(repo, bookID, pdfPath string) string {
	// Check if pdftoppm is available
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return "" // Silently skip if not installed
	}

	// Ensure .covers directory exists
	coversDir := filepath.Join(m.baseDir, repo, ".covers")
	if err := os.MkdirAll(coversDir, 0750); err != nil {
		return ""
	}

	// Output path: <cache>/<repo>/.covers/<book-id>.jpg
	// Using bookID ensures one cover per book (overwrites existing)
	coverPath := filepath.Join(coversDir, bookID+".jpg")

	// Remove existing cover if present (ensures only 1 per book)
	_ = os.Remove(coverPath)

	// Extract first page as JPEG with quality settings
	// -jpeg: Output JPEG format
	// -f 1 -l 1: First page only
	// -scale-to 300: Scale to max 300px (maintains aspect ratio)
	// -jpegopt quality=85: Reasonable quality, keeps size down
	outputPrefix := filepath.Join(coversDir, bookID)
	cmd := exec.Command("pdftoppm",
		"-jpeg",
		"-f", "1",
		"-l", "1",
		"-scale-to", "300",
		"-jpegopt", "quality=85",
		pdfPath,
		outputPrefix,
	)

	if err := cmd.Run(); err != nil {
		return "" // Silently fail
	}

	// pdftoppm creates <prefix>-1.jpg for page 1
	generatedPath := outputPrefix + "-1.jpg"

	// Rename to final path without page number: <book-id>.jpg
	if err := os.Rename(generatedPath, coverPath); err != nil {
		_ = os.Remove(generatedPath) // Clean up
		return ""
	}

	return coverPath
}

// CoverPath returns the path where a cover image would be stored.
func (m *Manager) CoverPath(repo, bookID string) string {
	return filepath.Join(m.baseDir, repo, ".covers", bookID+".jpg")
}

// HasCover checks if a cover image exists for the given book.
func (m *Manager) HasCover(repo, bookID string) bool {
	_, err := os.Stat(m.CoverPath(repo, bookID))
	return err == nil
}

// RemoveCover deletes the cover image for a book if it exists.
func (m *Manager) RemoveCover(repo, bookID string) error {
	path := m.CoverPath(repo, bookID)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
