package ingest

import "testing"

func TestGuessFilenameFromURL_Simple(t *testing.T) {
	got := guessFilenameFromURL("https://example.com/books/sicp.pdf")
	if got != "sicp.pdf" {
		t.Errorf("got %q, want %q", got, "sicp.pdf")
	}
}

func TestGuessFilenameFromURL_WithQueryString(t *testing.T) {
	got := guessFilenameFromURL("https://example.com/book.pdf?token=abc123")
	if got != "book.pdf" {
		t.Errorf("got %q, want %q", got, "book.pdf")
	}
}

func TestGuessFilenameFromURL_TrailingSlash(t *testing.T) {
	got := guessFilenameFromURL("https://example.com/")
	// filepath.Base strips trailing slash, returns host as last component
	if got == "" || got == "." || got == "/" {
		t.Errorf("got %q, want non-empty fallback", got)
	}
}

func TestGuessFilenameFromURL_NoPath(t *testing.T) {
	got := guessFilenameFromURL("https://example.com")
	if got == "" || got == "." || got == "/" {
		t.Errorf("got %q, want non-empty fallback", got)
	}
}
