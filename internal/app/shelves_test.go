package app

import (
	"strings"
	"testing"
)

// --- stripAnsi ---

func TestStripAnsi_Plain(t *testing.T) {
	got := stripAnsi("hello world")
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestStripAnsi_Empty(t *testing.T) {
	got := stripAnsi("")
	if got != "" {
		t.Errorf("got %q, want %q", got, "")
	}
}

func TestStripAnsi_Color(t *testing.T) {
	got := stripAnsi("\x1b[31mred\x1b[0m")
	if got != "red" {
		t.Errorf("got %q, want %q", got, "red")
	}
}

func TestStripAnsi_Bold(t *testing.T) {
	got := stripAnsi("\x1b[1mBold\x1b[0m")
	if got != "Bold" {
		t.Errorf("got %q, want %q", got, "Bold")
	}
}

func TestStripAnsi_Mixed(t *testing.T) {
	got := stripAnsi("\x1b[32mgreen\x1b[0m plain")
	if got != "green plain" {
		t.Errorf("got %q, want %q", got, "green plain")
	}
}

// --- padRight ---

func TestPadRight_Short(t *testing.T) {
	got := padRight("hi", 6)
	if got != "hi    " {
		t.Errorf("got %q, want %q", got, "hi    ")
	}
}

func TestPadRight_Exact(t *testing.T) {
	got := padRight("hello", 5)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestPadRight_Longer(t *testing.T) {
	got := padRight("toolong", 4)
	if got != "tool" {
		t.Errorf("got %q, want %q", got, "tool")
	}
}

// --- padRightColored ---

func TestPadRightColored_Plain(t *testing.T) {
	got := padRightColored("hi", 6)
	if got != "hi    " {
		t.Errorf("got %q, want %q", got, "hi    ")
	}
}

func TestPadRightColored_Colored(t *testing.T) {
	// "\x1b[31mHi\x1b[0m" has visual width 2, padded to 6 → 4 spaces appended
	input := "\x1b[31mHi\x1b[0m"
	got := padRightColored(input, 6)
	plain := stripAnsi(got)
	if plain != "Hi    " {
		t.Errorf("plain content = %q, want %q", plain, "Hi    ")
	}
}

func TestPadRightColored_AlreadyWide(t *testing.T) {
	// Visual width >= requested width: no padding added.
	got := padRightColored("hello", 3)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// --- formatBookCount ---

func TestFormatBookCount_Zero(t *testing.T) {
	got := formatBookCount(0)
	if got != "-" {
		t.Errorf("got %q, want %q", got, "-")
	}
}

func TestFormatBookCount_One(t *testing.T) {
	got := formatBookCount(1)
	if got != "1" {
		t.Errorf("got %q, want %q", got, "1")
	}
}

func TestFormatBookCount_Many(t *testing.T) {
	got := formatBookCount(42)
	if got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

// --- formatStatus ---

func TestFormatStatus_RepoNotOK(t *testing.T) {
	s := shelfStatus{repoOK: false, errorMsg: "repo not found"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "repo not found") {
		t.Errorf("got %q, want it to contain %q", got, "repo not found")
	}
}

func TestFormatStatus_CatalogMissing(t *testing.T) {
	s := shelfStatus{repoOK: true, catalogOK: false, needsFix: true, errorMsg: "catalog.yml missing"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "catalog.yml missing") {
		t.Errorf("got %q, want it to contain %q", got, "catalog.yml missing")
	}
}

func TestFormatStatus_ReleaseMissing(t *testing.T) {
	s := shelfStatus{repoOK: true, catalogOK: true, releaseOK: false, needsFix: false, errorMsg: "release missing"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "release missing") {
		t.Errorf("got %q, want it to contain %q", got, "release missing")
	}
}

func TestFormatStatus_Healthy(t *testing.T) {
	s := shelfStatus{repoOK: true, catalogOK: true, releaseOK: true}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "Healthy") {
		t.Errorf("got %q, want it to contain %q", got, "Healthy")
	}
}
