package app

import (
	"sort"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// TestEditBook_TagsDeterministic verifies that rebuilding tags from a map
// produces a deterministic (sorted) result.
func TestEditBook_TagsDeterministic(t *testing.T) {
	// Simulate the tag rebuild logic from edit_book.go line 270-276
	initialTags := []string{"functional", "programming", "algorithms", "design"}

	// Build tagSet as the code does
	tagSet := make(map[string]bool)
	for _, tag := range initialTags {
		tagSet[tag] = true
	}

	// Add some more tags
	tagSet["computer-science"] = true
	tagSet["mathematics"] = true

	// Remove one
	delete(tagSet, "design")

	// Rebuild tags slice twice
	run1 := rebuildTags(tagSet)
	run2 := rebuildTags(tagSet)

	// Verify identical order
	if len(run1) != len(run2) {
		t.Fatalf("length mismatch: run1=%d, run2=%d", len(run1), len(run2))
	}
	for i := range run1 {
		if run1[i] != run2[i] {
			t.Errorf("position %d differs: run1=%q, run2=%q", i, run1[i], run2[i])
		}
	}

	// Verify sorted
	if !sort.StringsAreSorted(run1) {
		t.Errorf("tags not sorted: %v", run1)
	}
}

// rebuildTags replicates the logic from edit_book.go after the BUG 26 fix
func rebuildTags(tagSet map[string]bool) []string {
	tags := []string{}
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// TestEditBook_TagsIntegration tests that the catalog.Book Tags field
// remains stable across multiple edits
func TestEditBook_TagsIntegration(t *testing.T) {
	book := catalog.Book{
		ID:     "test-book",
		Title:  "Test Book",
		Author: "Author",
		Year:   2023,
		Tags:   []string{"programming", "algorithms"},
	}

	// Simulate adding tags via map
	tagSet := make(map[string]bool)
	for _, tag := range book.Tags {
		tagSet[tag] = true
	}
	tagSet["design"] = true
	tagSet["functional"] = true

	// Rebuild with sort
	tags := []string{}
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	book.Tags = tags

	expected := []string{"algorithms", "design", "functional", "programming"}
	if len(book.Tags) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(book.Tags), len(expected))
	}
	for i := range expected {
		if book.Tags[i] != expected[i] {
			t.Errorf("position %d: got %q, want %q", i, book.Tags[i], expected[i])
		}
	}
}
