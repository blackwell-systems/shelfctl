package migrate

import (
	"strings"
	"testing"
)

func TestMatchExt_EmptyList(t *testing.T) {
	if !matchExt("anything.pdf", nil) {
		t.Error("empty ext list should match everything")
	}
}

func TestMatchExt_Match(t *testing.T) {
	if !matchExt("book.pdf", []string{"pdf"}) {
		t.Error("should match pdf")
	}
}

func TestMatchExt_CaseInsensitive(t *testing.T) {
	if !matchExt("BOOK.PDF", []string{"pdf"}) {
		t.Error("should match case-insensitively")
	}
}

func TestMatchExt_DotPrefix(t *testing.T) {
	if !matchExt("book.pdf", []string{".pdf"}) {
		t.Error("should handle dot-prefixed extensions")
	}
}

func TestMatchExt_NoMatch(t *testing.T) {
	if matchExt("book.epub", []string{"pdf"}) {
		t.Error("should not match different extension")
	}
}

func TestMatchExt_MultipleExts(t *testing.T) {
	if !matchExt("book.epub", []string{"pdf", "epub"}) {
		t.Error("should match when extension is in list")
	}
}

func TestMatchExt_NoExtension(t *testing.T) {
	if matchExt("README", []string{"pdf"}) {
		t.Error("file with no extension should not match")
	}
}

func TestDefaultLedgerPath(t *testing.T) {
	p := DefaultLedgerPath()
	if p == "" {
		t.Fatal("DefaultLedgerPath returned empty string")
	}
	if !strings.HasSuffix(p, "migrated.jsonl") {
		t.Errorf("DefaultLedgerPath = %q, should end with migrated.jsonl", p)
	}
}
