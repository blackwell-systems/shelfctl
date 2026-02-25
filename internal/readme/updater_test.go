package readme

import (
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

func TestUpdateStats_PluralToPlural(t *testing.T) {
	content := "# My Shelf\n\n**5 books** | Last updated: today"
	got := updateStats(content, 8)
	if !strings.Contains(got, "**8 books**") {
		t.Errorf("expected **8 books**, got %q", got)
	}
}

func TestUpdateStats_PluralToSingular(t *testing.T) {
	content := "**5 books** in this shelf"
	got := updateStats(content, 1)
	if !strings.Contains(got, "**1 book**") {
		t.Errorf("expected **1 book**, got %q", got)
	}
}

func TestUpdateStats_SingularToPlural(t *testing.T) {
	content := "**1 book** in this shelf"
	got := updateStats(content, 3)
	if !strings.Contains(got, "**3 books**") {
		t.Errorf("expected **3 books**, got %q", got)
	}
}

func TestUpdateStats_NoMatch(t *testing.T) {
	content := "No stats here"
	got := updateStats(content, 5)
	if got != content {
		t.Errorf("expected unchanged content, got %q", got)
	}
}

func TestAppendToRecentlyAdded_Inserts(t *testing.T) {
	content := "# Shelf\n\n## Recently Added\n\n- **old** — Old Book by Author\n"
	book := catalog.Book{ID: "new", Title: "New Book", Author: "Author"}
	got := appendToRecentlyAdded(content, book)
	idx := strings.Index(got, "- **new**")
	oldIdx := strings.Index(got, "- **old**")
	if idx == -1 {
		t.Fatal("new entry not found")
	}
	if idx >= oldIdx {
		t.Error("new entry should appear before old entry")
	}
}

func TestAppendToRecentlyAdded_SkipsDupe(t *testing.T) {
	content := "## Recently Added\n\n- **sicp** — SICP by Abelson\n"
	book := catalog.Book{ID: "sicp", Title: "SICP", Author: "Abelson"}
	got := appendToRecentlyAdded(content, book)
	if got != content {
		t.Error("should not add duplicate entry")
	}
}

func TestAppendToRecentlyAdded_NoSection(t *testing.T) {
	content := "# Shelf\n\nNo recently added section here.\n"
	book := catalog.Book{ID: "test", Title: "Test", Author: "Author"}
	got := appendToRecentlyAdded(content, book)
	if got != content {
		t.Error("should return unchanged when section missing")
	}
}

func TestAppendToRecentlyAdded_WithTags(t *testing.T) {
	content := "## Recently Added\n\n"
	book := catalog.Book{ID: "tagged", Title: "Tagged Book", Author: "Author", Tags: []string{"go", "testing"}}
	got := appendToRecentlyAdded(content, book)
	if !strings.Contains(got, "| go, testing") {
		t.Errorf("expected tags in entry, got %q", got)
	}
}

func TestAppendToRecentlyAdded_NoTags(t *testing.T) {
	content := "## Recently Added\n\n"
	book := catalog.Book{ID: "notag", Title: "No Tags", Author: "Author"}
	got := appendToRecentlyAdded(content, book)
	if strings.Contains(got, " | ") {
		t.Error("should not contain tag separator when no tags")
	}
}

func TestRemoveFromRecentlyAdded_Found(t *testing.T) {
	content := "## Recently Added\n\n- **sicp** — SICP by Abelson\n- **ostep** — OSTEP by Arpaci\n"
	got := removeFromRecentlyAdded(content, "sicp")
	if strings.Contains(got, "sicp") {
		t.Error("sicp entry should be removed")
	}
	if !strings.Contains(got, "ostep") {
		t.Error("ostep entry should remain")
	}
}

func TestRemoveFromRecentlyAdded_NotFound(t *testing.T) {
	content := "## Recently Added\n\n- **sicp** — SICP by Abelson\n"
	got := removeFromRecentlyAdded(content, "missing")
	if got != content {
		t.Error("content should be unchanged when book not found")
	}
}
