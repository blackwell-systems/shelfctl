package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// --- isPDF ---

func TestIsPDF_Lowercase(t *testing.T) {
	if !isPDF("book.pdf") {
		t.Error("book.pdf should be PDF")
	}
}

func TestIsPDF_Uppercase(t *testing.T) {
	if !isPDF("BOOK.PDF") {
		t.Error("BOOK.PDF should be PDF")
	}
}

func TestIsPDF_MixedCase(t *testing.T) {
	if !isPDF("Book.Pdf") {
		t.Error("Book.Pdf should be PDF")
	}
}

func TestIsPDF_NotPDF(t *testing.T) {
	if isPDF("book.epub") {
		t.Error("book.epub should not be PDF")
	}
}

func TestIsPDF_Empty(t *testing.T) {
	if isPDF("") {
		t.Error("empty string should not be PDF")
	}
}

func TestIsPDF_NoExtension(t *testing.T) {
	if isPDF("README") {
		t.Error("README should not be PDF")
	}
}

// --- GetPopplerInstallHint ---

func TestGetPopplerInstallHint(t *testing.T) {
	hint := GetPopplerInstallHint()
	if hint == "" {
		t.Fatal("GetPopplerInstallHint returned empty string")
	}
	if !strings.Contains(hint, "poppler") {
		t.Errorf("hint should mention poppler: %q", hint)
	}
}

// --- CoverPath / CatalogCoverPath ---

func TestCoverPath(t *testing.T) {
	m := New("/base")
	got := m.CoverPath("myrepo", "sicp")
	want := filepath.Join("/base", "myrepo", ".covers", "sicp.jpg")
	if got != want {
		t.Errorf("CoverPath = %q, want %q", got, want)
	}
}

func TestCatalogCoverPath(t *testing.T) {
	m := New("/base")
	got := m.CatalogCoverPath("myrepo", "sicp")
	want := filepath.Join("/base", "myrepo", ".covers", "sicp-catalog.jpg")
	if got != want {
		t.Errorf("CatalogCoverPath = %q, want %q", got, want)
	}
}

// --- GetCoverPath priority ---

func TestGetCoverPath_NoCover(t *testing.T) {
	m := New(t.TempDir())
	got := m.GetCoverPath("repo", "missing")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestGetCoverPath_ExtractedOnly(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	coverDir := filepath.Join(dir, "repo", ".covers")
	_ = os.MkdirAll(coverDir, 0750)
	_ = os.WriteFile(filepath.Join(coverDir, "book.jpg"), []byte("img"), 0644)

	got := m.GetCoverPath("repo", "book")
	if got != m.CoverPath("repo", "book") {
		t.Errorf("expected extracted cover path, got %q", got)
	}
}

func TestGetCoverPath_CatalogPriority(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	coverDir := filepath.Join(dir, "repo", ".covers")
	_ = os.MkdirAll(coverDir, 0750)
	_ = os.WriteFile(filepath.Join(coverDir, "book.jpg"), []byte("extracted"), 0644)
	_ = os.WriteFile(filepath.Join(coverDir, "book-catalog.jpg"), []byte("catalog"), 0644)

	got := m.GetCoverPath("repo", "book")
	if got != m.CatalogCoverPath("repo", "book") {
		t.Errorf("expected catalog cover path (higher priority), got %q", got)
	}
}

// --- StoreCatalogCover / HasCatalogCover / RemoveCatalogCover ---

func TestCatalogCover_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	if m.HasCatalogCover("repo", "book") {
		t.Fatal("should not have cover before store")
	}

	path := m.StoreCatalogCover("repo", "book", strings.NewReader("image data"))
	if path == "" {
		t.Fatal("StoreCatalogCover returned empty path")
	}

	if !m.HasCatalogCover("repo", "book") {
		t.Error("should have cover after store")
	}

	if err := m.RemoveCatalogCover("repo", "book"); err != nil {
		t.Fatalf("RemoveCatalogCover: %v", err)
	}

	if m.HasCatalogCover("repo", "book") {
		t.Error("should not have cover after remove")
	}
}

// --- Remove ---

func TestRemove_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	_, err := m.Store("owner", "repo", "book", "book.pdf", strings.NewReader("data"), "")
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !m.Exists("owner", "repo", "book", "book.pdf") {
		t.Fatal("should exist after store")
	}

	if err := m.Remove("owner", "repo", "book", "book.pdf"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if m.Exists("owner", "repo", "book", "book.pdf") {
		t.Error("should not exist after remove")
	}
}

func TestRemove_NonExistent(t *testing.T) {
	m := New(t.TempDir())
	if err := m.Remove("owner", "repo", "missing", "file.pdf"); err != nil {
		t.Errorf("Remove non-existent: %v", err)
	}
}

// --- generateHTML ---

func TestGenerateHTML_Empty(t *testing.T) {
	html := generateHTML(nil)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should contain DOCTYPE")
	}
	if !strings.Contains(html, "Library") {
		t.Error("should contain Library in title")
	}
	if !strings.Contains(html, "0 books") {
		t.Error("should show 0 books")
	}
}

func TestGenerateHTML_WithBooks(t *testing.T) {
	books := []IndexBook{
		{
			Book:      catalog.Book{ID: "sicp", Title: "SICP", Author: "Abelson", Tags: []string{"lisp"}},
			ShelfName: "programming",
			FilePath:  "/cache/sicp.pdf",
			IsCached:  true,
		},
	}
	html := generateHTML(books)
	if !strings.Contains(html, "sicp") {
		t.Error("should contain book ID")
	}
	if !strings.Contains(html, "SICP") {
		t.Error("should contain book title")
	}
	if !strings.Contains(html, "programming") {
		t.Error("should contain shelf name")
	}
}

func TestGenerateHTML_UncachedSection(t *testing.T) {
	books := []IndexBook{
		{
			Book:      catalog.Book{ID: "cached-book", Title: "Cached"},
			ShelfName: "shelf",
			FilePath:  "/cache/cached.pdf",
			IsCached:  true,
		},
		{
			Book:     catalog.Book{ID: "uncached-book", Title: "Uncached"},
			IsCached: false,
		},
	}
	html := generateHTML(books)
	if !strings.Contains(html, "Not yet downloaded") {
		t.Error("should contain uncached section title")
	}
	if !strings.Contains(html, "shelfctl open uncached-book") {
		t.Error("should contain shelfctl open hint for uncached book")
	}
}

// --- renderUncachedCard ---

func TestRenderUncachedCard(t *testing.T) {
	var s strings.Builder
	book := IndexBook{
		Book: catalog.Book{
			ID:     "test-book",
			Title:  "Test Book",
			Author: "Author",
			Tags:   []string{"go"},
		},
	}
	renderUncachedCard(&s, book, 0)
	html := s.String()

	if !strings.Contains(html, "uncached") {
		t.Error("should have uncached class")
	}
	if !strings.Contains(html, "shelfctl open test-book") {
		t.Error("should contain shelfctl open hint")
	}
	if !strings.Contains(html, "Test Book") {
		t.Error("should contain title")
	}
	if !strings.Contains(html, "Author") {
		t.Error("should contain author")
	}
}

// --- GenerateHTMLIndex (disk write) ---

func TestGenerateHTMLIndex(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)

	books := []IndexBook{
		{
			Book:      catalog.Book{ID: "test", Title: "Test Book"},
			ShelfName: "shelf",
			FilePath:  "/cache/test.pdf",
			IsCached:  true,
		},
	}
	if err := m.GenerateHTMLIndex(books); err != nil {
		t.Fatalf("GenerateHTMLIndex: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatalf("reading index.html: %v", err)
	}
	if !strings.Contains(string(data), "<!DOCTYPE html>") {
		t.Error("index.html should contain DOCTYPE")
	}
}

// --- renderBookCard variants ---

func TestRenderBookCard_WithCover(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")
	_ = os.WriteFile(coverPath, []byte("fake image"), 0644)

	var s strings.Builder
	book := IndexBook{
		Book:      catalog.Book{ID: "book1", Title: "Book One"},
		FilePath:  filepath.Join(dir, "book1.pdf"),
		CoverPath: coverPath,
		HasCover:  true,
		IsCached:  true,
	}
	renderBookCard(&s, book, 0)
	html := s.String()

	if !strings.Contains(html, "<img") {
		t.Error("should contain <img> tag for cover")
	}
	if !strings.Contains(html, "Cover") {
		t.Error("should contain alt text")
	}
}

func TestRenderBookCard_NoCover(t *testing.T) {
	var s strings.Builder
	book := IndexBook{
		Book:     catalog.Book{ID: "book2", Title: "Book Two"},
		FilePath: "/fake/path.pdf",
		HasCover: false,
		IsCached: true,
	}
	renderBookCard(&s, book, 0)
	html := s.String()

	if !strings.Contains(html, "no-cover") {
		t.Error("should have no-cover class")
	}
	if strings.Contains(html, "<img") {
		t.Error("should not contain <img> tag without cover")
	}
}

func TestGenerateHTML_TagFilters(t *testing.T) {
	books := []IndexBook{
		{
			Book:      catalog.Book{ID: "b1", Title: "B1", Tags: []string{"go", "testing"}},
			ShelfName: "shelf",
			FilePath:  "/cache/b1.pdf",
			IsCached:  true,
		},
	}
	html := generateHTML(books)
	if !strings.Contains(html, "tag-filter") {
		t.Error("should contain tag filter buttons")
	}
	if !strings.Contains(html, "go") {
		t.Error("should contain 'go' tag")
	}
	if !strings.Contains(html, "testing") {
		t.Error("should contain 'testing' tag")
	}
}

func TestGenerateHTML_CachedCountSubtitle(t *testing.T) {
	books := []IndexBook{
		{Book: catalog.Book{ID: "a"}, IsCached: true, ShelfName: "s", FilePath: "/a.pdf"},
		{Book: catalog.Book{ID: "b"}, IsCached: false},
	}
	html := generateHTML(books)
	if !strings.Contains(html, "2 books (1 cached)") {
		t.Errorf("subtitle should show cached count, got html containing subtitle")
	}
}
