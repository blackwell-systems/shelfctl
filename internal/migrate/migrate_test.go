package migrate_test

import (
	"path/filepath"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/migrate"
)

// --- RouteSource / FindRoute ---

func TestRouteSource_ExactPrefix(t *testing.T) {
	src := config.MigrationSource{
		Mapping: map[string]string{
			"programming/": "programming",
			"history/":     "history",
		},
	}
	shelf := migrate.RouteSource("programming/sicp.pdf", src)
	if shelf != "programming" {
		t.Errorf("RouteSource = %q, want %q", shelf, "programming")
	}
}

func TestRouteSource_LongestPrefixWins(t *testing.T) {
	src := config.MigrationSource{
		Mapping: map[string]string{
			"cs/":          "programming",
			"cs/advanced/": "advanced",
		},
	}
	shelf := migrate.RouteSource("cs/advanced/algo.pdf", src)
	if shelf != "advanced" {
		t.Errorf("longest-prefix: got %q, want %q", shelf, "advanced")
	}
}

func TestRouteSource_NoMatch(t *testing.T) {
	src := config.MigrationSource{
		Mapping: map[string]string{
			"programming/": "programming",
		},
	}
	shelf := migrate.RouteSource("unknown/file.pdf", src)
	if shelf != "" {
		t.Errorf("expected empty shelf for no match, got %q", shelf)
	}
}

func TestFindRoute_Found(t *testing.T) {
	sources := []config.MigrationSource{
		{Owner: "alice", Repo: "books", Ref: "main",
			Mapping: map[string]string{"history/": "history"}},
		{Owner: "alice", Repo: "papers", Ref: "main",
			Mapping: map[string]string{"cs/": "programming"}},
	}
	src, shelf, ok := migrate.FindRoute("cs/compilers.pdf", sources)
	if !ok {
		t.Fatal("FindRoute returned ok=false, want true")
	}
	if shelf != "programming" {
		t.Errorf("shelf = %q, want %q", shelf, "programming")
	}
	if src.Repo != "papers" {
		t.Errorf("src.Repo = %q, want %q", src.Repo, "papers")
	}
}

func TestFindRoute_NotFound(t *testing.T) {
	sources := []config.MigrationSource{
		{Mapping: map[string]string{"known/": "shelf"}},
	}
	_, _, ok := migrate.FindRoute("unknown/file.pdf", sources)
	if ok {
		t.Error("FindRoute returned ok=true for unmapped path")
	}
}

// --- Ledger ---

func TestLedger_AppendAndContains(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrated.jsonl")

	l, err := migrate.OpenLedger(path)
	if err != nil {
		t.Fatal(err)
	}

	// Not yet migrated.
	found, err := l.Contains("programming/sicp.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("Contains returned true before any entries")
	}

	// Add entry.
	if err := l.Append(migrate.LedgerEntry{
		Source: "programming/sicp.pdf",
		BookID: "sicp",
		Shelf:  "programming",
	}); err != nil {
		t.Fatal(err)
	}

	found, err = l.Contains("programming/sicp.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("Contains returned false after append")
	}
}

func TestLedger_ContainsMissingFile(t *testing.T) {
	dir := t.TempDir()
	l, _ := migrate.OpenLedger(filepath.Join(dir, "nonexistent.jsonl"))
	found, err := l.Contains("any/path.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("Contains on missing ledger file should return false, not true")
	}
}

func TestLedger_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	l, _ := migrate.OpenLedger(filepath.Join(dir, "ledger.jsonl"))

	paths := []string{"a/1.pdf", "b/2.pdf", "c/3.pdf"}
	for _, p := range paths {
		_ = l.Append(migrate.LedgerEntry{Source: p, BookID: p, Shelf: "test"})
	}

	for _, p := range paths {
		found, _ := l.Contains(p)
		if !found {
			t.Errorf("Contains(%q) = false, want true", p)
		}
	}
	found, _ := l.Contains("x/missing.pdf")
	if found {
		t.Error("Contains returned true for path not in ledger")
	}
}

// --- matchExt (via ScanRepo indirectly tested through FileEntry) ---
