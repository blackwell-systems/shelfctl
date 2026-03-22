package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

func TestDetectOrphans_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	shelves := []ShelfCatalog{
		{
			Owner: "user",
			Repo:  "shelf-tech",
			Books: []catalog.Book{
				{ID: "book1", Source: catalog.Source{Asset: "book1.pdf"}},
			},
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 0 {
		t.Errorf("expected 0 orphans in empty cache, got %d", report.TotalCount)
	}
}

func TestDetectOrphans_NoOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create cache structure
	repoDir := filepath.Join(tmpDir, "shelf-tech")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Write a file that IS in the catalog
	cachedFile := filepath.Join(repoDir, "book1.pdf")
	if err := os.WriteFile(cachedFile, []byte("content"), 0640); err != nil {
		t.Fatal(err)
	}

	shelves := []ShelfCatalog{
		{
			Owner: "user",
			Repo:  "shelf-tech",
			Books: []catalog.Book{
				{ID: "book1", Source: catalog.Source{Asset: "book1.pdf"}},
			},
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 0 {
		t.Errorf("expected 0 orphans when all files are referenced, got %d", report.TotalCount)
	}
}

func TestDetectOrphans_FindsOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create cache structure with two files
	repoDir := filepath.Join(tmpDir, "shelf-tech")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	// File 1: Referenced in catalog
	file1 := filepath.Join(repoDir, "book1.pdf")
	if err := os.WriteFile(file1, []byte("content1"), 0640); err != nil {
		t.Fatal(err)
	}

	// File 2: NOT referenced in catalog (orphan)
	file2 := filepath.Join(repoDir, "orphan.pdf")
	if err := os.WriteFile(file2, []byte("orphan content"), 0640); err != nil {
		t.Fatal(err)
	}

	shelves := []ShelfCatalog{
		{
			Owner: "user",
			Repo:  "shelf-tech",
			Books: []catalog.Book{
				{ID: "book1", Source: catalog.Source{Asset: "book1.pdf"}},
			},
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 1 {
		t.Fatalf("expected 1 orphan, got %d", report.TotalCount)
	}

	if report.Entries[0].Filename != "orphan.pdf" {
		t.Errorf("expected orphan filename 'orphan.pdf', got %q", report.Entries[0].Filename)
	}

	if report.Entries[0].Repo != "shelf-tech" {
		t.Errorf("expected orphan repo 'shelf-tech', got %q", report.Entries[0].Repo)
	}

	if report.TotalSize != 14 { // len("orphan content")
		t.Errorf("expected total size 14, got %d", report.TotalSize)
	}
}

func TestDetectOrphans_SkipsTempFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	repoDir := filepath.Join(tmpDir, "shelf-tech")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a .tmp file (should be ignored)
	tmpFile := filepath.Join(repoDir, "download.pdf.tmp")
	if err := os.WriteFile(tmpFile, []byte("temp"), 0640); err != nil {
		t.Fatal(err)
	}

	shelves := []ShelfCatalog{
		{
			Owner: "user",
			Repo:  "shelf-tech",
			Books: []catalog.Book{},
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 0 {
		t.Errorf("expected 0 orphans (.tmp files should be skipped), got %d", report.TotalCount)
	}
}

func TestDetectOrphans_MultipleRepos(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	// Create two repos with orphans in each
	repo1Dir := filepath.Join(tmpDir, "shelf-tech")
	repo2Dir := filepath.Join(tmpDir, "shelf-fiction")
	if err := os.MkdirAll(repo1Dir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repo2Dir, 0750); err != nil {
		t.Fatal(err)
	}

	// Orphan in repo1
	if err := os.WriteFile(filepath.Join(repo1Dir, "orphan1.pdf"), []byte("o1"), 0640); err != nil {
		t.Fatal(err)
	}

	// Orphan in repo2
	if err := os.WriteFile(filepath.Join(repo2Dir, "orphan2.epub"), []byte("o2"), 0640); err != nil {
		t.Fatal(err)
	}

	shelves := []ShelfCatalog{
		{
			Owner: "user",
			Repo:  "shelf-tech",
			Books: []catalog.Book{},
		},
		{
			Owner: "user",
			Repo:  "shelf-fiction",
			Books: []catalog.Book{},
		},
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalCount != 2 {
		t.Errorf("expected 2 orphans across multiple repos, got %d", report.TotalCount)
	}
}

func TestClearOrphans_Success(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	repoDir := filepath.Join(tmpDir, "shelf-tech")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	orphanPath := filepath.Join(repoDir, "orphan.pdf")
	if err := os.WriteFile(orphanPath, []byte("content"), 0640); err != nil {
		t.Fatal(err)
	}

	report := OrphanReport{
		Entries: []OrphanEntry{
			{
				Path:     orphanPath,
				Repo:     "shelf-tech",
				Filename: "orphan.pdf",
				Size:     7,
			},
		},
		TotalSize:  7,
		TotalCount: 1,
	}

	deleted, err := mgr.ClearOrphans(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 file deleted, got %d", deleted)
	}

	// Verify file is gone
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("orphan file still exists after clearing")
	}
}

func TestClearOrphans_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := New(tmpDir)

	repoDir := filepath.Join(tmpDir, "shelf-tech")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create one file that exists
	existingPath := filepath.Join(repoDir, "orphan1.pdf")
	if err := os.WriteFile(existingPath, []byte("content"), 0640); err != nil {
		t.Fatal(err)
	}

	// Create report with one existing file and one non-existent file
	report := OrphanReport{
		Entries: []OrphanEntry{
			{Path: existingPath, Repo: "shelf-tech", Filename: "orphan1.pdf", Size: 7},
			{Path: filepath.Join(repoDir, "nonexistent.pdf"), Repo: "shelf-tech", Filename: "nonexistent.pdf", Size: 10},
		},
		TotalSize:  17,
		TotalCount: 2,
	}

	deleted, err := mgr.ClearOrphans(report)

	// Should delete the existing file even though one fails
	if deleted != 1 {
		t.Errorf("expected 1 file deleted, got %d", deleted)
	}

	// Should return an error for the failed deletion
	if err == nil {
		t.Error("expected error when some files fail to delete")
	}
}
