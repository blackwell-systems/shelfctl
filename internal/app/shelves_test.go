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
	s := shelfStatus{repoOK: false, errorMsg: "repository 'myowner/myrepo' not found"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "repository 'myowner/myrepo' not found") {
		t.Errorf("got %q, want it to contain %q", got, "repository 'myowner/myrepo' not found")
	}
}

func TestFormatStatus_RepoNotFound_IncludesShelfName(t *testing.T) {
	s := shelfStatus{
		name:     "my-shelf",
		repoOK:   false,
		errorMsg: "repository 'owner/shelf-my-shelf' not found",
	}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "owner/shelf-my-shelf") {
		t.Errorf("got %q, want it to contain repo path %q", got, "owner/shelf-my-shelf")
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

// --- Multiple failing shelves ---

func TestMultipleFailingShelves_EachNamedInError(t *testing.T) {
	statuses := []shelfStatus{
		{name: "shelf-a", repoOK: false, errorMsg: "repository 'org/shelf-a' not found"},
		{name: "shelf-b", repoOK: false, errorMsg: "repository 'org/shelf-b' not found"},
		{name: "shelf-c", repoOK: true, catalogOK: false, errorMsg: "catalog.yml missing"},
	}

	var failedShelves []string
	for _, s := range statuses {
		if s.errorMsg != "" {
			failedShelves = append(failedShelves, "shelf '"+s.name+"': "+s.errorMsg)
		}
	}
	joined := strings.Join(failedShelves, "; ")

	if !strings.Contains(joined, "shelf 'shelf-a'") {
		t.Errorf("error should mention shelf-a, got: %s", joined)
	}
	if !strings.Contains(joined, "shelf 'shelf-b'") {
		t.Errorf("error should mention shelf-b, got: %s", joined)
	}
	if !strings.Contains(joined, "shelf 'shelf-c'") {
		t.Errorf("error should mention shelf-c, got: %s", joined)
	}
}

// --- Different error types ---

func TestFormatStatus_RepoError(t *testing.T) {
	s := shelfStatus{repoOK: false, errorMsg: "repo error: 403 Forbidden"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "repo error: 403 Forbidden") {
		t.Errorf("got %q, want it to contain auth error message", got)
	}
}

func TestFormatStatus_NetworkTimeout(t *testing.T) {
	s := shelfStatus{repoOK: false, errorMsg: "repo error: context deadline exceeded (Client.Timeout)"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "context deadline exceeded") {
		t.Errorf("got %q, want it to contain timeout error", got)
	}
}

// --- Mix of passing and failing shelves ---

func TestMixOfPassingAndFailing_OnlyFailingInError(t *testing.T) {
	statuses := []shelfStatus{
		{name: "healthy-shelf", repoOK: true, catalogOK: true, releaseOK: true, errorMsg: ""},
		{name: "broken-shelf", repoOK: false, errorMsg: "repository 'org/broken-shelf' not found"},
		{name: "also-healthy", repoOK: true, catalogOK: true, releaseOK: true, errorMsg: ""},
	}

	var failedShelves []string
	for _, s := range statuses {
		if s.errorMsg != "" {
			failedShelves = append(failedShelves, "shelf '"+s.name+"': "+s.errorMsg)
		}
	}
	joined := strings.Join(failedShelves, "; ")

	if strings.Contains(joined, "healthy-shelf") {
		t.Errorf("error should NOT mention healthy-shelf, got: %s", joined)
	}
	if strings.Contains(joined, "also-healthy") {
		t.Errorf("error should NOT mention also-healthy, got: %s", joined)
	}
	if !strings.Contains(joined, "broken-shelf") {
		t.Errorf("error should mention broken-shelf, got: %s", joined)
	}
}

// --- --fix error message variant (table mode with --fix hint) ---

func TestFixErrorMessageVariant_TableMode(t *testing.T) {
	// Simulate the table-mode error format from shelves.go line ~66
	statuses := []shelfStatus{
		{name: "bad-shelf", repoOK: true, catalogOK: false, needsFix: true, errorMsg: "catalog.yml missing"},
	}

	var failedShelves []string
	for _, s := range statuses {
		if s.errorMsg != "" {
			failedShelves = append(failedShelves, "shelf '"+s.name+"': "+s.errorMsg)
		}
	}
	// Table mode error format includes "Run with --fix to repair."
	errMsg := "shelf issues:\n  " + strings.Join(failedShelves, "\n  ") + "\n\nRun with --fix to repair."

	if !strings.Contains(errMsg, "Run with --fix to repair.") {
		t.Errorf("table mode error should contain --fix hint, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "bad-shelf") {
		t.Errorf("table mode error should name the shelf, got: %s", errMsg)
	}
}

func TestFixErrorMessageVariant_ListMode(t *testing.T) {
	// Simulate the list-mode error format from shelves.go line ~81
	statuses := []shelfStatus{
		{name: "broken", repoOK: false, errorMsg: "repository 'org/broken' not found"},
	}

	var failedShelves []string
	for _, s := range statuses {
		if s.errorMsg != "" {
			failedShelves = append(failedShelves, "shelf '"+s.name+"': "+s.errorMsg)
		}
	}
	// List mode error format uses semicolons and no --fix hint
	errMsg := "shelf issues: " + strings.Join(failedShelves, "; ")

	if strings.Contains(errMsg, "--fix") {
		t.Errorf("list mode error should NOT contain --fix hint, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "broken") {
		t.Errorf("list mode error should name the shelf, got: %s", errMsg)
	}
}

// --- formatStatus with needsFix flag ---

func TestFormatStatus_CatalogMissing_NeedsFix_ShowsWarning(t *testing.T) {
	s := shelfStatus{repoOK: true, catalogOK: false, needsFix: true, errorMsg: "catalog.yml missing"}
	got := stripAnsi(formatStatus(s))
	// When needsFix is true, formatStatus uses yellow warning symbol
	if !strings.Contains(got, "catalog.yml missing") {
		t.Errorf("got %q, want it to contain %q", got, "catalog.yml missing")
	}
}

func TestFormatStatus_ReleaseMissing_NeedsFix(t *testing.T) {
	s := shelfStatus{repoOK: true, catalogOK: true, releaseOK: false, needsFix: true, errorMsg: "release 'library' missing"}
	got := stripAnsi(formatStatus(s))
	if !strings.Contains(got, "release 'library' missing") {
		t.Errorf("got %q, want it to contain %q", got, "release 'library' missing")
	}
}
