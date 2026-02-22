package app

import (
	"testing"
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
