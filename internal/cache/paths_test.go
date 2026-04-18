package cache_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/cache"
)

// TestPath_IncludesOwner verifies that Path returns a path containing the
// owner segment in the layout <baseDir>/<owner>/<repo>/<assetFilename>.
func TestPath_IncludesOwner(t *testing.T) {
	dir := t.TempDir()
	m := cache.New(dir)

	got := m.Path("alice", "myrepo", "book1", "file.pdf")
	want := filepath.Join(dir, "alice", "myrepo", "file.pdf")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
	if !strings.Contains(got, "alice") {
		t.Errorf("Path() = %q does not contain owner segment %q", got, "alice")
	}
}

// TestPath_DifferentOwnersSeparated verifies that two shelves with the same
// repo name but different owners produce different cache paths.
func TestPath_DifferentOwnersSeparated(t *testing.T) {
	dir := t.TempDir()
	m := cache.New(dir)

	pathAlice := m.Path("alice", "myrepo", "book1", "file.pdf")
	pathBob := m.Path("bob", "myrepo", "book1", "file.pdf")

	if pathAlice == pathBob {
		t.Errorf("Path() returned same path for different owners: %q", pathAlice)
	}
	if !strings.Contains(pathAlice, "alice") {
		t.Errorf("alice path %q does not contain owner segment", pathAlice)
	}
	if !strings.Contains(pathBob, "bob") {
		t.Errorf("bob path %q does not contain owner segment", pathBob)
	}
}

// TestEnsureDir_CreatesOwnerSubdir verifies that EnsureDir creates the
// <baseDir>/<owner>/<repo>/ directory structure.
func TestEnsureDir_CreatesOwnerSubdir(t *testing.T) {
	dir := t.TempDir()
	m := cache.New(dir)

	if err := m.EnsureDir("alice", "myrepo", "book1"); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}

	expectedDir := filepath.Join(dir, "alice", "myrepo")
	info, err := os.Stat(expectedDir)
	if err != nil {
		t.Fatalf("EnsureDir did not create %q: %v", expectedDir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q exists but is not a directory", expectedDir)
	}
}

// TestBaseDir_ReturnsCacheRoot verifies that BaseDir() returns the root cache
// directory and not any owner/repo subpath.
func TestBaseDir_ReturnsCacheRoot(t *testing.T) {
	dir := t.TempDir()
	m := cache.New(dir)

	got := m.BaseDir()
	if got != dir {
		t.Errorf("BaseDir() = %q, want %q", got, dir)
	}

	// Ensure BaseDir is not a subpath — it should equal the root exactly.
	rel, err := filepath.Rel(dir, got)
	if err != nil {
		t.Fatalf("filepath.Rel: %v", err)
	}
	if rel != "." {
		t.Errorf("BaseDir() %q is a subpath of root %q (rel=%q); expected root itself", got, dir, rel)
	}
}
