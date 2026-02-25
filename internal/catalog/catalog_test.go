package catalog_test

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

var sampleYAML = []byte(`
- id: sicp
  title: "Structure and Interpretation of Computer Programs"
  author: "Abelson & Sussman"
  year: 1996
  tags: [lisp, cs, programming]
  format: pdf
  checksum:
    sha256: "abc123"
  size_bytes: 10485760
  source:
    type: github_release
    owner: alice
    repo: shelf-programming
    release: library
    asset: sicp.pdf
  meta:
    added_at: "2026-01-01T00:00:00Z"

- id: ostep
  title: "Operating Systems: Three Easy Pieces"
  author: "Arpaci-Dusseau"
  year: 2018
  tags: [os, systems]
  format: pdf
  source:
    type: github_release
    owner: alice
    repo: shelf-programming
    release: systems
    asset: ostep.pdf
`)

// --- Parse / Marshal round-trip ---

func TestParse_ValidYAML(t *testing.T) {
	books, err := catalog.Parse(sampleYAML)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}
	if books[0].ID != "sicp" {
		t.Errorf("books[0].ID = %q, want %q", books[0].ID, "sicp")
	}
	if books[1].Source.Release != "systems" {
		t.Errorf("books[1].Source.Release = %q, want %q", books[1].Source.Release, "systems")
	}
}

func TestParse_Empty(t *testing.T) {
	books, err := catalog.Parse([]byte(""))
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}
}

func TestParse_EmptyList(t *testing.T) {
	books, err := catalog.Parse([]byte("[]\n"))
	if err != nil {
		t.Fatalf("Parse []: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}
}

func TestMarshal_RoundTrip(t *testing.T) {
	books, err := catalog.Parse(sampleYAML)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	data, err := catalog.Marshal(books)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	books2, err := catalog.Parse(data)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if len(books2) != len(books) {
		t.Fatalf("round-trip length: got %d, want %d", len(books2), len(books))
	}
	for i := range books {
		if books[i].ID != books2[i].ID {
			t.Errorf("[%d] ID mismatch: %q vs %q", i, books[i].ID, books2[i].ID)
		}
		if books[i].Source.Release != books2[i].Source.Release {
			t.Errorf("[%d] Source.Release mismatch", i)
		}
	}
}

// --- Append / Remove ---

func TestAppend_New(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	newBook := catalog.Book{ID: "newbook", Title: "New Book", Format: "epub"}
	books = catalog.Append(books, newBook)
	if len(books) != 3 {
		t.Errorf("expected 3 after append, got %d", len(books))
	}
	if books[2].ID != "newbook" {
		t.Errorf("last book ID = %q, want %q", books[2].ID, "newbook")
	}
}

func TestAppend_ReplacesExisting(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	updated := catalog.Book{ID: "sicp", Title: "SICP (updated)", Format: "pdf"}
	books = catalog.Append(books, updated)
	if len(books) != 2 {
		t.Errorf("expected 2 after update, got %d", len(books))
	}
	if books[0].Title != "SICP (updated)" {
		t.Errorf("title not updated: %q", books[0].Title)
	}
}

func TestRemove_Existing(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	books, ok := catalog.Remove(books, "sicp")
	if !ok {
		t.Error("Remove returned ok=false for existing book")
	}
	if len(books) != 1 {
		t.Errorf("expected 1 book after remove, got %d", len(books))
	}
	if books[0].ID != "ostep" {
		t.Errorf("remaining book = %q, want %q", books[0].ID, "ostep")
	}
}

func TestRemove_Missing(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	books, ok := catalog.Remove(books, "nope")
	if ok {
		t.Error("Remove returned ok=true for missing book")
	}
	if len(books) != 2 {
		t.Errorf("expected 2 books after no-op remove, got %d", len(books))
	}
}

// --- ByID ---

func TestByID_Found(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	b := catalog.ByID(books, "ostep")
	if b == nil {
		t.Fatal("ByID returned nil for existing book")
	}
	if b.ID != "ostep" {
		t.Errorf("ID = %q, want %q", b.ID, "ostep")
	}
}

func TestByID_NotFound(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	b := catalog.ByID(books, "missing")
	if b != nil {
		t.Errorf("ByID returned non-nil for missing book")
	}
}

// --- Filter ---

func TestFilter_ByTag(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Tag: "os"}
	result := f.Apply(books)
	if len(result) != 1 || result[0].ID != "ostep" {
		t.Errorf("tag filter: got %v", ids(result))
	}
}

func TestFilter_ByTagCaseInsensitive(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Tag: "OS"}
	result := f.Apply(books)
	if len(result) != 1 {
		t.Errorf("case-insensitive tag filter failed: got %v", ids(result))
	}
}

func TestFilter_ByFormat(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Format: "pdf"}
	result := f.Apply(books)
	if len(result) != 2 {
		t.Errorf("format filter: expected 2, got %d", len(result))
	}
}

func TestFilter_BySearch_Title(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Search: "operating systems"}
	result := f.Apply(books)
	if len(result) != 1 || result[0].ID != "ostep" {
		t.Errorf("search by title failed: got %v", ids(result))
	}
}

func TestFilter_BySearch_Tag(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Search: "lisp"}
	result := f.Apply(books)
	if len(result) != 1 || result[0].ID != "sicp" {
		t.Errorf("search by tag failed: got %v", ids(result))
	}
}

func TestFilter_NoMatch(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Search: "zzznomatch"}
	result := f.Apply(books)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestFilter_Empty(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	result := catalog.Filter{}.Apply(books)
	if len(result) != 2 {
		t.Errorf("empty filter should return all books, got %d", len(result))
	}
}

func ids(books []catalog.Book) []string {
	out := make([]string, len(books))
	for i, b := range books {
		out[i] = b.ID
	}
	return out
}

// --- Additional Filter tests ---

func TestFilter_BySearch_Author(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Search: "abelson"}
	result := f.Apply(books)
	if len(result) != 1 || result[0].ID != "sicp" {
		t.Errorf("search by author failed: got %v", ids(result))
	}
}

func TestFilter_Combined_TagAndFormat(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Tag: "os", Format: "pdf"}
	result := f.Apply(books)
	if len(result) != 1 || result[0].ID != "ostep" {
		t.Errorf("combined filter: got %v", ids(result))
	}
}

func TestFilter_Combined_NoMatch(t *testing.T) {
	books, _ := catalog.Parse(sampleYAML)
	f := catalog.Filter{Tag: "os", Format: "epub"}
	result := f.Apply(books)
	if len(result) != 0 {
		t.Errorf("combined filter with no match: expected 0, got %d", len(result))
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := catalog.Parse([]byte(":: bad yaml ["))
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestMarshal_EmptySlice(t *testing.T) {
	data, err := catalog.Marshal([]catalog.Book{})
	if err != nil {
		t.Fatalf("Marshal empty slice: %v", err)
	}
	if len(data) == 0 {
		t.Error("Marshal empty slice returned empty bytes")
	}
}
