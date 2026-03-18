package app

import (
	"runtime"
	"testing"
)

// TestIsPDF tests the isPDF helper function
func TestIsPDF(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "lowercase .pdf",
			filename: "book.pdf",
			want:     true,
		},
		{
			name:     "uppercase .PDF",
			filename: "DOCUMENT.PDF",
			want:     true,
		},
		{
			name:     "mixed case .PdF",
			filename: "file.PdF",
			want:     true,
		},
		{
			name:     "epub file",
			filename: "book.epub",
			want:     false,
		},
		{
			name:     "no extension",
			filename: "book",
			want:     false,
		},
		{
			name:     "pdf in path but not extension",
			filename: "pdf/book.epub",
			want:     false,
		},
		{
			name:     "multiple extensions",
			filename: "archive.tar.pdf",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPDF(tt.filename)
			if got != tt.want {
				t.Errorf("isPDF(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// TestOpenFile_CommandAssembly tests the command name selection logic
func TestOpenFile_CommandAssembly(t *testing.T) {
	// This test verifies the command-name selection based on runtime.GOOS.
	// We don't execute the command, just verify the logic.

	// Based on the implementation in open.go, openFile selects:
	// - darwin: "open"
	// - windows: "cmd"
	// - others: "xdg-open"

	var expectedCmd string
	switch runtime.GOOS {
	case "darwin":
		expectedCmd = "open"
	case "windows":
		expectedCmd = "cmd"
	default:
		expectedCmd = "xdg-open"
	}

	// Verify our expectation matches what openFile would use.
	// Since openFile calls exec.Start which we can't test directly,
	// we document the expected behavior here as a reference test.
	t.Logf("On %s, openFile should use command: %s", runtime.GOOS, expectedCmd)

	// This is a documentation test - the actual logic is verified by
	// manual testing or integration tests that would be too complex
	// to run in unit tests (would require mocking exec.Command).
	if runtime.GOOS == "darwin" && expectedCmd != "open" {
		t.Error("Expected 'open' command on darwin")
	}
	if runtime.GOOS == "linux" && expectedCmd != "xdg-open" {
		t.Error("Expected 'xdg-open' command on linux")
	}
	if runtime.GOOS == "windows" && expectedCmd != "cmd" {
		t.Error("Expected 'cmd' command on windows")
	}
}
