package util

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.input), func(t *testing.T) {
			got := HumanBytes(tt.input)
			if got != tt.want {
				t.Errorf("HumanBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsPDF(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"document.pdf", true},
		{"DOCUMENT.PDF", true},
		{"mixed.Pdf", true},
		{"book.epub", false},
		{"readme.txt", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsPDF(tt.filename)
			if got != tt.want {
				t.Errorf("IsPDF(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestComputeFileHash(t *testing.T) {
	content := []byte("hello world\n")
	tmp := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmp, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, size, err := ComputeFileHash(tmp)
	if err != nil {
		t.Fatalf("ComputeFileHash() error = %v", err)
	}

	wantSize := int64(len(content))
	if size != wantSize {
		t.Errorf("size = %d, want %d", size, wantSize)
	}

	wantHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if hash != wantHash {
		t.Errorf("hash = %q, want %q", hash, wantHash)
	}
}

func TestComputeFileHash_NonExistent(t *testing.T) {
	_, _, err := ComputeFileHash("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestOpenFile_InvalidCommand(t *testing.T) {
	err := OpenFile("/nonexistent/file", "/nonexistent/command/binary")
	if err == nil {
		t.Error("expected error for invalid command, got nil")
	}
}
