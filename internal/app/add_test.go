package app

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SICP", "sicp"},
		{"Operating Systems: Three Easy Pieces", "operating-systems-three-easy-pieces"},
		{"  leading/trailing  ", "leading-trailing"},
		{"hello world", "hello-world"},
		{"café au lait", "caf-au-lait"},
		{"123abc", "123abc"},
		{"---all-dashes---", "all-dashes"},
		{"", "book"},
		{"Let's Go Further", "lets-go-further"},
		{"Don't Stop Believin'", "dont-stop-believin"},
		{"The \"Best\" Book", "the-best-book"},
		{"O'Reilly's Guide", "oreillys-guide"},
		{"It's a \u201Cquoted\u201D title", "its-a-quoted-title"},
	}
	for _, c := range cases {
		got := slugify(c.in)
		if got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugify_MaxLength(t *testing.T) {
	// 100 'a' chars should be trimmed to 63.
	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	got := slugify(long)
	if len(got) > 63 {
		t.Errorf("slugify result length = %d, want ≤63", len(got))
	}
}

func TestSlugify_TrailingHyphenAfterTruncation(t *testing.T) {
	// Test case: long filename that truncates at a hyphen position
	// "how-linux-works-what-every-superuser-should-know-3rd-edition" is 59 chars
	// But with longer variants, truncation might land on a hyphen
	long := "how-linux-works-what-every-superuser-should-know-3rd-edition-revised"
	got := slugify(long)

	// Should not end with hyphen
	if len(got) > 0 && got[len(got)-1] == '-' {
		t.Errorf("slugify(%q) ends with hyphen: %q", long, got)
	}

	// Should be ≤63 chars
	if len(got) > 63 {
		t.Errorf("slugify result length = %d, want ≤63", len(got))
	}

	// Should match ID regex
	if got != "" && !idRe.MatchString(got) {
		t.Errorf("slugify(%q) = %q, does not match ID regex", long, got)
	}
}

func TestSlugify_SpecialUnicode(t *testing.T) {
	cases := []struct{ in, want string }{
		// All-numeric edge case
		{"123", "123"},
		{"42", "42"},
		{"2023", "2023"},

		// Mixed unicode letters that aren't ASCII alphanumeric
		// Non-ASCII letters become separators (hyphens)
		{"Résumé", "r-sum"},
		{"naïve", "na-ve"},
		{"São Paulo", "s-o-paulo"},
		{"München", "m-nchen"},
		{"Ångström", "ngstr-m"},

		// Special cases with unicode combining characters
		{"café", "caf"},
		{"mañana", "ma-ana"},

		// Mixed digits and non-ASCII
		{"c++11", "c-11"},
		{"python3.11", "python3-11"},
		{"utf-8", "utf-8"},
	}
	for _, c := range cases {
		got := slugify(c.in)
		if got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckDuplicates_EmptyBooks(t *testing.T) {
	// Empty catalog should never report a duplicate
	emptyBooks := []catalog.Book{}

	err := checkDuplicates(emptyBooks, "abc123", false)
	if err != nil {
		t.Errorf("checkDuplicates with empty catalog returned error: %v", err)
	}
}

func TestCheckDuplicates_NoDuplicate(t *testing.T) {
	books := []catalog.Book{
		{ID: "book1", Checksum: catalog.Checksum{SHA256: "sha1"}},
		{ID: "book2", Checksum: catalog.Checksum{SHA256: "sha2"}},
	}

	// Different SHA256 should not be a duplicate
	err := checkDuplicates(books, "sha3", false)
	if err != nil {
		t.Errorf("checkDuplicates with unique SHA256 returned error: %v", err)
	}
}

func TestCheckDuplicates_FoundDuplicate(t *testing.T) {
	books := []catalog.Book{
		{ID: "book1", Title: "First Book", Checksum: catalog.Checksum{SHA256: "abc123"}},
		{ID: "book2", Title: "Second Book", Checksum: catalog.Checksum{SHA256: "def456"}},
	}

	// Matching SHA256 should report duplicate
	err := checkDuplicates(books, "abc123", false)
	if err == nil {
		t.Error("checkDuplicates with duplicate SHA256 should return error")
	}
}

func TestCheckDuplicates_ForceSkipsDuplicateCheck(t *testing.T) {
	books := []catalog.Book{
		{ID: "book1", Title: "First Book", Checksum: catalog.Checksum{SHA256: "abc123"}},
	}

	// With force=true, duplicate should be allowed
	err := checkDuplicates(books, "abc123", true)
	if err != nil {
		t.Errorf("checkDuplicates with force=true should not return error, got: %v", err)
	}
}
