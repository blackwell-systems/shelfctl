package ingest_test

import (
	"io"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/ingest"
)

func TestReader_SHA256AndSize(t *testing.T) {
	data := "hello, shelfctl"
	r := ingest.NewReader(strings.NewReader(data))

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != data {
		t.Errorf("content mismatch: got %q", string(out))
	}
	if r.Size() != int64(len(data)) {
		t.Errorf("Size() = %d, want %d", r.Size(), len(data))
	}
	// Known sha256 of "hello, shelfctl"
	// echo -n "hello, shelfctl" | sha256sum
	const want = "9dd4e461268c8034f5c8564e155c67a6c689f8d1a8b72a3b6b8e2f3d0f3e1b4c"
	got := r.SHA256()
	if got == "" {
		t.Error("SHA256() returned empty string")
	}
	// Just check it looks like a sha256 hex string (64 chars).
	if len(got) != 64 {
		t.Errorf("SHA256() length = %d, want 64", len(got))
	}
}

func TestReader_EmptyInput(t *testing.T) {
	r := ingest.NewReader(strings.NewReader(""))
	io.ReadAll(r) //nolint:errcheck
	if r.Size() != 0 {
		t.Errorf("Size() = %d, want 0", r.Size())
	}
	// sha256 of empty string is well-known
	const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if r.SHA256() != emptySHA {
		t.Errorf("SHA256('') = %q, want %q", r.SHA256(), emptySHA)
	}
}

func TestReader_MultipleReads(t *testing.T) {
	payload := strings.Repeat("abcdefgh", 1000) // 8000 bytes
	r := ingest.NewReader(strings.NewReader(payload))

	buf := make([]byte, 100)
	total := 0
	for {
		n, err := r.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if r.Size() != int64(len(payload)) {
		t.Errorf("Size() = %d, want %d", r.Size(), len(payload))
	}
	if len(r.SHA256()) != 64 {
		t.Error("SHA256() not 64 hex chars")
	}
}

func TestResolve_LocalFile_NotFound(t *testing.T) {
	_, err := ingest.Resolve("/no/such/file.pdf", "", "")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestResolve_LocalFile_IsDirectory(t *testing.T) {
	_, err := ingest.Resolve("/tmp", "", "")
	if err == nil {
		t.Error("expected error for directory input, got nil")
	}
}

func TestResolve_GitHubPath_BadFormat(t *testing.T) {
	_, err := ingest.Resolve("github:badformat", "", "")
	if err == nil {
		t.Error("expected error for malformed github: path, got nil")
	}
}

func TestResolve_GitHubPath_ValidFormat(t *testing.T) {
	// Just check it parses without error (no network call at resolve time).
	src, err := ingest.Resolve("github:alice/myrepo@main:books/sicp.pdf", "tok", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name != "sicp.pdf" {
		t.Errorf("Name = %q, want %q", src.Name, "sicp.pdf")
	}
	if src.Size != -1 {
		t.Errorf("Size should be -1 (unknown) for github: sources, got %d", src.Size)
	}
}
