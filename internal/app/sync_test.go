package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFileHash(t *testing.T) {
	// Create temp file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute hash
	hash, size, err := computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	// Verify size
	if size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", size, len(content))
	}

	// Verify hash (SHA256 of "hello world")
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expectedHash {
		t.Errorf("Hash = %q, want %q", hash, expectedHash)
	}
}

func TestComputeFileHash_NonExistent(t *testing.T) {
	_, _, err := computeFileHash("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestComputeFileHash_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, size, err := computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if size != 0 {
		t.Errorf("Size = %d, want 0", size)
	}

	// SHA256 of empty string
	expectedHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expectedHash {
		t.Errorf("Hash = %q, want %q", hash, expectedHash)
	}
}

func TestComputeFileHash_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create 1MB file
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, size, err := computeFileHash(testFile)
	if err != nil {
		t.Fatalf("computeFileHash failed: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", size, len(content))
	}

	// Hash should be deterministic
	hash2, _, err := computeFileHash(testFile)
	if err != nil {
		t.Fatalf("Second computeFileHash failed: %v", err)
	}

	if hash != hash2 {
		t.Error("Hash is not deterministic")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		list     []string
		item     string
		expected bool
	}{
		{"empty list", []string{}, "foo", false},
		{"found at start", []string{"foo", "bar", "baz"}, "foo", true},
		{"found in middle", []string{"foo", "bar", "baz"}, "bar", true},
		{"found at end", []string{"foo", "bar", "baz"}, "baz", true},
		{"not found", []string{"foo", "bar", "baz"}, "qux", false},
		{"empty string in list", []string{"", "foo"}, "", true},
		{"empty string not in list", []string{"foo", "bar"}, "", false},
		{"single item match", []string{"only"}, "only", true},
		{"single item no match", []string{"only"}, "other", false},
		{"case sensitive", []string{"Foo", "Bar"}, "foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.list, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.list, tt.item, result, tt.expected)
			}
		})
	}
}
