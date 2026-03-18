package github

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestGetFileContent_SmallFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/contents/path/to/file.txt", func(w http.ResponseWriter, r *http.Request) {
		content := "Hello, World!"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{
			"name": "file.txt",
			"path": "path/to/file.txt",
			"sha": "abc123",
			"size": %d,
			"encoding": "base64",
			"content": "%s"
		}`, len(content), encoded)
	})
	_, c := newFakeServer(t, mux)

	data, sha, err := c.GetFileContent("owner", "repo", "path/to/file.txt", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("expected sha=%q, got %q", "abc123", sha)
	}
	expected := "Hello, World!"
	if string(data) != expected {
		t.Errorf("expected content=%q, got %q", expected, string(data))
	}
}

func TestGetFileContent_LargeFileFallback(t *testing.T) {
	mux := http.NewServeMux()
	blobSHA := "largefile123"
	blobContent := "This is large file content"

	// First request to contents API returns encoding=none and size>1MB
	mux.HandleFunc("/repos/owner/repo/contents/large.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"name": "large.bin",
			"path": "large.bin",
			"sha": "%s",
			"size": %d,
			"encoding": "none",
			"content": ""
		}`, blobSHA, 2*1024*1024) // 2 MB
	})

	// Blobs API endpoint should be called for large files
	mux.HandleFunc("/repos/owner/repo/git/blobs/"+blobSHA, func(w http.ResponseWriter, r *http.Request) {
		// Check for raw accept header
		accept := r.Header.Get("Accept")
		if accept != "application/vnd.github.raw" {
			t.Errorf("expected Accept=application/vnd.github.raw, got %q", accept)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(blobContent))
	})

	_, c := newFakeServer(t, mux)

	data, sha, err := c.GetFileContent("owner", "repo", "large.bin", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != blobSHA {
		t.Errorf("expected sha=%q, got %q", blobSHA, sha)
	}
	if string(data) != blobContent {
		t.Errorf("expected content=%q, got %q", blobContent, string(data))
	}
}

func TestGetFileContent_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/contents/missing.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, c := newFakeServer(t, mux)

	_, _, err := c.GetFileContent("owner", "repo", "missing.txt", "")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetFileContent_WithRef(t *testing.T) {
	mux := http.NewServeMux()
	var capturedRef string
	mux.HandleFunc("/repos/owner/repo/contents/file.txt", func(w http.ResponseWriter, r *http.Request) {
		capturedRef = r.URL.Query().Get("ref")
		content := "ref content"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"name": "file.txt",
			"path": "file.txt",
			"sha": "ref123",
			"size": %d,
			"encoding": "base64",
			"content": "%s"
		}`, len(content), encoded)
	})
	_, c := newFakeServer(t, mux)

	_, _, err := c.GetFileContent("owner", "repo", "file.txt", "feature-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRef != "feature-branch" {
		t.Errorf("expected ref query param=%q, got %q", "feature-branch", capturedRef)
	}
}

func TestGetFileContent_StripNewlines(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/contents/file.txt", func(w http.ResponseWriter, r *http.Request) {
		content := "This is a longer content that GitHub wraps"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		// Simulate GitHub's line wrapping at 60 chars by inserting newlines
		wrappedEncoded := ""
		for i := 0; i < len(encoded); i += 60 {
			end := i + 60
			if end > len(encoded) {
				end = len(encoded)
			}
			wrappedEncoded += encoded[i:end] + "\n"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"name": "file.txt",
			"path": "file.txt",
			"sha": "wrapped123",
			"size": %d,
			"encoding": "base64",
			"content": "%s"
		}`, len(content), strings.TrimSpace(wrappedEncoded))
	})
	_, c := newFakeServer(t, mux)

	data, sha, err := c.GetFileContent("owner", "repo", "file.txt", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "wrapped123" {
		t.Errorf("expected sha=%q, got %q", "wrapped123", sha)
	}
	expected := "This is a longer content that GitHub wraps"
	if string(data) != expected {
		t.Errorf("expected content=%q, got %q", expected, string(data))
	}
}
