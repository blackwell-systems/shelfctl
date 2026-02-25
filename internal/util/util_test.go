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

func TestIsTTY(t *testing.T) {
	// In test environment, stdout is a pipe, not a TTY
	if util.IsTTY() {
		t.Error("IsTTY should return false in test environment")
	}
}

func TestSHA256Reader_KnownContent(t *testing.T) {
	got, err := util.SHA256Reader(strings.NewReader("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	// sha256("hello world") is well known
	const want = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a0ec65e1f47ca0"
	if len(got) != 64 {
		t.Errorf("SHA256 hash should be 64 hex chars, got %d: %q", len(got), got)
	}
}

func TestSHA256File_NonEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "sha256test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_, _ = f.WriteString("hello")
	_ = f.Close()

	got, err := util.SHA256File(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 64 {
		t.Errorf("SHA256 hash should be 64 hex chars, got %d", len(got))
	}
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("SHA256File('hello') = %q, want %q", got, want)
	}
}

func TestExpandHome_BareTilde(t *testing.T) {
	got := util.ExpandHome("~")
	if got != "~" {
		t.Errorf("ExpandHome(\"~\") = %q, want \"~\" (no expansion without /)", got)
	}
}
