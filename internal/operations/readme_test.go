package operations

import (
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// TestAppendToShelfREADME_QuickStatsOnly tests BUG 17 fix:
// README with Quick Stats but no other ## section; confirm that after append
// the book entry appears once, not duplicated.
func TestAppendToShelfREADME_QuickStatsOnly(t *testing.T) {
	existingREADME := `# My Shelf

A curated collection managed with shelfctl.

## Quick Stats

- **Books**: 0
- **Last Updated**: 2024-01-01

## Adding Books

Add books to this shelf using shelfctl.
`

	book := catalog.Book{
		ID:     "book-123",
		Title:  "Test Book",
		Author: "Test Author",
		Tags:   []string{"fiction", "test"},
	}

	result := AppendToShelfREADME(existingREADME, book)

	// Count occurrences of the book ID in the result
	idPattern := "(`book-123`)"
	count := strings.Count(result, idPattern)

	if count != 1 {
		t.Errorf("Expected book entry to appear exactly once, but found %d occurrences", count)
		t.Logf("Result README:\n%s", result)
	}

	// Verify Recently Added section exists
	if !strings.Contains(result, "## Recently Added") {
		t.Error("Expected '## Recently Added' section in result")
	}

	// Verify the book entry is present
	expectedEntry := "- **Test Book** by Test Author (`book-123`) - Tags: fiction, test"
	if !strings.Contains(result, expectedEntry) {
		t.Errorf("Expected book entry not found in result.\nExpected: %s\nResult:\n%s", expectedEntry, result)
	}

	// Verify the book entry appears after Quick Stats
	quickStatsIdx := strings.Index(result, "## Quick Stats")
	recentlyAddedIdx := strings.Index(result, "## Recently Added")
	bookEntryIdx := strings.Index(result, expectedEntry)

	if quickStatsIdx < 0 {
		t.Error("Quick Stats section not found in result")
	}

	if recentlyAddedIdx < 0 {
		t.Error("Recently Added section not found in result")
	}

	if bookEntryIdx < 0 {
		t.Error("Book entry not found in result")
	}

	if quickStatsIdx >= 0 && recentlyAddedIdx >= 0 {
		if recentlyAddedIdx <= quickStatsIdx {
			t.Error("Recently Added section should appear after Quick Stats section")
		}
	}

	if recentlyAddedIdx >= 0 && bookEntryIdx >= 0 {
		if bookEntryIdx <= recentlyAddedIdx {
			t.Error("Book entry should appear after Recently Added header")
		}
	}

	// Count total sections to ensure no duplication
	sectionCount := strings.Count(result, "## Recently Added")
	if sectionCount != 1 {
		t.Errorf("Expected exactly 1 '## Recently Added' section, found %d", sectionCount)
	}
}

// TestAppendToShelfREADME_QuickStatsWithNextSection tests the normal case
// where there's a section after Quick Stats (should insert before it).
func TestAppendToShelfREADME_QuickStatsWithNextSection(t *testing.T) {
	existingREADME := `# My Shelf

## Quick Stats

- **Books**: 0
- **Last Updated**: 2024-01-01

## Adding Books

Add books using shelfctl.
`

	book := catalog.Book{
		ID:     "book-456",
		Title:  "Another Book",
		Author: "Another Author",
		Tags:   []string{},
	}

	result := AppendToShelfREADME(existingREADME, book)

	// Verify Recently Added section is inserted between Quick Stats and Adding Books
	quickStatsIdx := strings.Index(result, "## Quick Stats")
	recentlyAddedIdx := strings.Index(result, "## Recently Added")
	addingBooksIdx := strings.Index(result, "## Adding Books")

	if quickStatsIdx < 0 || recentlyAddedIdx < 0 || addingBooksIdx < 0 {
		t.Error("Expected all three sections to be present")
	}

	if !(quickStatsIdx < recentlyAddedIdx && recentlyAddedIdx < addingBooksIdx) {
		t.Error("Expected order: Quick Stats < Recently Added < Adding Books")
		t.Logf("Result:\n%s", result)
	}

	// Verify book entry is present exactly once
	bookIDPattern := "(`book-456`)"
	count := strings.Count(result, bookIDPattern)
	if count != 1 {
		t.Errorf("Expected book entry to appear exactly once, but found %d occurrences", count)
	}
}

// TestAppendToShelfREADME_ExistingRecentlyAdded tests adding to existing Recently Added section.
func TestAppendToShelfREADME_ExistingRecentlyAdded(t *testing.T) {
	existingREADME := `# My Shelf

## Quick Stats

- **Books**: 1
- **Last Updated**: 2024-01-01

## Recently Added

- **Old Book** by Old Author (` + "`old-id`" + `)

## Adding Books

Add books using shelfctl.
`

	book := catalog.Book{
		ID:     "new-id",
		Title:  "New Book",
		Author: "New Author",
		Tags:   []string{},
	}

	result := AppendToShelfREADME(existingREADME, book)

	// Verify both books are present
	if !strings.Contains(result, "(`new-id`)") {
		t.Error("New book not found in result")
	}

	if !strings.Contains(result, "(`old-id`)") {
		t.Error("Old book should still be present")
	}

	// Verify new book appears before old book (should be at top)
	newBookIdx := strings.Index(result, "(`new-id`)")
	oldBookIdx := strings.Index(result, "(`old-id`)")

	if newBookIdx >= oldBookIdx {
		t.Error("New book should appear before old book (most recent first)")
		t.Logf("Result:\n%s", result)
	}
}

// TestAppendToShelfREADME_NoDuplicates tests that adding the same book twice
// doesn't create duplicates.
func TestAppendToShelfREADME_NoDuplicates(t *testing.T) {
	existingREADME := `# My Shelf

## Quick Stats

- **Books**: 1

## Recently Added

- **Test Book** by Test Author (` + "`test-123`" + `)

## Adding Books
`

	book := catalog.Book{
		ID:     "test-123",
		Title:  "Test Book Updated",
		Author: "Test Author Updated",
		Tags:   []string{},
	}

	result := AppendToShelfREADME(existingREADME, book)

	// Verify book ID appears exactly once
	count := strings.Count(result, "(`test-123`)")
	if count != 1 {
		t.Errorf("Expected book ID to appear exactly once, found %d occurrences", count)
		t.Logf("Result:\n%s", result)
	}

	// Verify the updated title is present (not the old one)
	if !strings.Contains(result, "Test Book Updated") {
		t.Error("Updated book title not found")
	}
}

// TestUpdateShelfREADMEStats tests the stats update functionality.
func TestUpdateShelfREADMEStats(t *testing.T) {
	existingREADME := `# My Shelf

## Quick Stats

- **Books**: 0
- **Last Updated**: 2024-01-01

## Recently Added

Some content here.
`

	result := UpdateShelfREADMEStats(existingREADME, 42)

	// Verify book count is updated
	if !strings.Contains(result, "- **Books**: 42") {
		t.Error("Book count not updated correctly")
		t.Logf("Result:\n%s", result)
	}

	// Verify Recently Added section is preserved
	if !strings.Contains(result, "## Recently Added") {
		t.Error("Recently Added section should be preserved")
	}

	if !strings.Contains(result, "Some content here.") {
		t.Error("Content after Quick Stats should be preserved")
	}
}

// TestRemoveFromShelfREADME tests book removal from Recently Added section.
func TestRemoveFromShelfREADME(t *testing.T) {
	existingREADME := `# My Shelf

## Recently Added

- **Book One** by Author One (` + "`book-1`" + `)
- **Book Two** by Author Two (` + "`book-2`" + `)
- **Book Three** by Author Three (` + "`book-3`" + `)

## Adding Books
`

	result := RemoveFromShelfREADME(existingREADME, "book-2")

	// Verify book-2 is removed
	if strings.Contains(result, "(`book-2`)") {
		t.Error("Book-2 should be removed")
		t.Logf("Result:\n%s", result)
	}

	// Verify other books are still present
	if !strings.Contains(result, "(`book-1`)") {
		t.Error("Book-1 should still be present")
	}

	if !strings.Contains(result, "(`book-3`)") {
		t.Error("Book-3 should still be present")
	}
}
