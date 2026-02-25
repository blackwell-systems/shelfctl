package util_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/util"
)

func TestSHA256Reader(t *testing.T) {
	// sha256("") is well known
	got, err := util.SHA256Reader(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256('') = %q, want %q", got, want)
	}
}

func TestSHA256File(t *testing.T) {
	f, err := os.CreateTemp("", "sha256test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.WriteString("")
	_ = f.Close()

	got, err := util.SHA256File(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256File(empty) = %q, want %q", got, want)
	}
}

func TestSHA256File_MissingFile(t *testing.T) {
	_, err := util.SHA256File("/no/such/file.bin")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct{ in, want string }{
		{"~/foo/bar", filepath.Join(home, "foo", "bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, c := range cases {
		got := util.ExpandHome(c.in)
		if got != c.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
