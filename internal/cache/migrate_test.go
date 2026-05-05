package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateOwnerLayout_MovesOldToNew(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create old-layout directory: <baseDir>/my-repo/book.pdf
	oldDir := filepath.Join(tmpDir, "my-repo")
	if err := os.MkdirAll(oldDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "book.pdf"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr.MigrateOwnerLayout([]ShelfMigrationEntry{
		{Owner: "someowner", Repo: "my-repo"},
	})

	// Old path should be gone
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("old directory still exists after migration")
	}

	// New path should exist
	newFile := filepath.Join(tmpDir, "someowner", "my-repo", "book.pdf")
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("expected migrated file at %s: %v", newFile, err)
	}
}

func TestMigrateOwnerLayout_SkipsIfNewExists(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create both old and new layout
	oldDir := filepath.Join(tmpDir, "my-repo")
	newDir := filepath.Join(tmpDir, "owner", "my-repo")
	if err := os.MkdirAll(oldDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "old.pdf"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "new.pdf"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr.MigrateOwnerLayout([]ShelfMigrationEntry{
		{Owner: "owner", Repo: "my-repo"},
	})

	// Old dir should still exist (not moved, since new already exists)
	if _, err := os.Stat(filepath.Join(oldDir, "old.pdf")); err != nil {
		t.Error("old directory was incorrectly moved when new already existed")
	}

	// New dir should still have its original content
	if _, err := os.Stat(filepath.Join(newDir, "new.pdf")); err != nil {
		t.Error("new directory was clobbered")
	}
}

func TestMigrateOwnerLayout_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create old layout
	oldDir := filepath.Join(tmpDir, "my-repo")
	if err := os.MkdirAll(oldDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "book.pdf"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []ShelfMigrationEntry{{Owner: "owner", Repo: "my-repo"}}

	// Run twice
	mgr.MigrateOwnerLayout(entries)
	mgr.MigrateOwnerLayout(entries)

	// Should still be fine
	newFile := filepath.Join(tmpDir, "owner", "my-repo", "book.pdf")
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("expected file at %s after double migration: %v", newFile, err)
	}
}

func TestMigrateOwnerLayout_NoOldDir(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// No old directory exists — should be a no-op
	mgr.MigrateOwnerLayout([]ShelfMigrationEntry{
		{Owner: "owner", Repo: "nonexistent"},
	})

	// Nothing should have been created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty cache dir, got %d entries", len(entries))
	}
}
