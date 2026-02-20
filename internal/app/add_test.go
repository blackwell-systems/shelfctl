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
