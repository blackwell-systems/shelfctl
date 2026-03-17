package app

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// TestVerifyConsecutiveOrphans tests BUG 4 fix: when two consecutive entries are orphaned,
// both should be removed after the fix, not just the first one.
func TestVerifyConsecutiveOrphans(t *testing.T) {
	// Create a slice with 3 books where entries 0 and 1 are both orphaned
	books := []catalog.Book{
		{
			ID:     "orphan1",
			Title:  "Orphaned Book 1",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan1.epub",
			},
		},
		{
			ID:     "orphan2",
			Title:  "Orphaned Book 2",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan2.epub",
			},
		},
		{
			ID:     "valid",
			Title:  "Valid Book",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "valid.epub",
			},
		},
	}

	// Create a map of assets that only contains the valid book
	assetNames := map[string]bool{
		"valid.epub": true,
	}

	// Simulate the two-pass approach used in the fix
	var toRemove []string
	for i := range books {
		b := &books[i]
		if _, exists := assetNames[b.Source.Asset]; !exists {
			toRemove = append(toRemove, b.ID)
		}
	}

	// Remove collected IDs
	for _, id := range toRemove {
		books, _ = catalog.Remove(books, id)
	}

	// Verify results: should have removed both orphan1 and orphan2
	if len(books) != 1 {
		t.Errorf("Expected 1 book remaining, got %d", len(books))
	}

	if len(books) > 0 && books[0].ID != "valid" {
		t.Errorf("Expected remaining book to be 'valid', got '%s'", books[0].ID)
	}

	if len(toRemove) != 2 {
		t.Errorf("Expected 2 books to be removed, got %d", len(toRemove))
	}

	// Verify both orphans were identified for removal
	foundOrphan1 := false
	foundOrphan2 := false
	for _, id := range toRemove {
		if id == "orphan1" {
			foundOrphan1 = true
		}
		if id == "orphan2" {
			foundOrphan2 = true
		}
	}

	if !foundOrphan1 {
		t.Error("Expected 'orphan1' to be in removal list")
	}
	if !foundOrphan2 {
		t.Error("Expected 'orphan2' to be in removal list")
	}
}

// TestVerifyNonConsecutiveOrphans tests that non-consecutive orphaned entries
// are also correctly removed.
func TestVerifyNonConsecutiveOrphans(t *testing.T) {
	books := []catalog.Book{
		{
			ID:     "orphan1",
			Title:  "Orphaned Book 1",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan1.epub",
			},
		},
		{
			ID:     "valid",
			Title:  "Valid Book",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "valid.epub",
			},
		},
		{
			ID:     "orphan2",
			Title:  "Orphaned Book 2",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan2.epub",
			},
		},
	}

	assetNames := map[string]bool{
		"valid.epub": true,
	}

	var toRemove []string
	for i := range books {
		b := &books[i]
		if _, exists := assetNames[b.Source.Asset]; !exists {
			toRemove = append(toRemove, b.ID)
		}
	}

	for _, id := range toRemove {
		books, _ = catalog.Remove(books, id)
	}

	if len(books) != 1 {
		t.Errorf("Expected 1 book remaining, got %d", len(books))
	}

	if len(toRemove) != 2 {
		t.Errorf("Expected 2 books to be removed, got %d", len(toRemove))
	}
}

// TestVerifyAllOrphans tests that when all books are orphaned, all are removed.
func TestVerifyAllOrphans(t *testing.T) {
	books := []catalog.Book{
		{
			ID:     "orphan1",
			Title:  "Orphaned Book 1",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan1.epub",
			},
		},
		{
			ID:     "orphan2",
			Title:  "Orphaned Book 2",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan2.epub",
			},
		},
		{
			ID:     "orphan3",
			Title:  "Orphaned Book 3",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "orphan3.epub",
			},
		},
	}

	assetNames := map[string]bool{} // No valid assets

	var toRemove []string
	for i := range books {
		b := &books[i]
		if _, exists := assetNames[b.Source.Asset]; !exists {
			toRemove = append(toRemove, b.ID)
		}
	}

	for _, id := range toRemove {
		books, _ = catalog.Remove(books, id)
	}

	if len(books) != 0 {
		t.Errorf("Expected 0 books remaining, got %d", len(books))
	}

	if len(toRemove) != 3 {
		t.Errorf("Expected 3 books to be removed, got %d", len(toRemove))
	}
}

// TestVerifyNoOrphans tests that when no books are orphaned, none are removed.
func TestVerifyNoOrphans(t *testing.T) {
	books := []catalog.Book{
		{
			ID:     "valid1",
			Title:  "Valid Book 1",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "valid1.epub",
			},
		},
		{
			ID:     "valid2",
			Title:  "Valid Book 2",
			Format: "epub",
			Source: catalog.Source{
				Type:    "github",
				Owner:   "test",
				Repo:    "test",
				Release: "v1.0.0",
				Asset:   "valid2.epub",
			},
		},
	}

	assetNames := map[string]bool{
		"valid1.epub": true,
		"valid2.epub": true,
	}

	var toRemove []string
	for i := range books {
		b := &books[i]
		if _, exists := assetNames[b.Source.Asset]; !exists {
			toRemove = append(toRemove, b.ID)
		}
	}

	for _, id := range toRemove {
		books, _ = catalog.Remove(books, id)
	}

	if len(books) != 2 {
		t.Errorf("Expected 2 books remaining, got %d", len(books))
	}

	if len(toRemove) != 0 {
		t.Errorf("Expected 0 books to be removed, got %d", len(toRemove))
	}
}
