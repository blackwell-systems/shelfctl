package github

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitize_RedactsToken(t *testing.T) {
	args := []string{
		"clone",
		"https://x-access-token:ghp_secrettoken12345@github.com/owner/repo.git",
		".",
	}
	output := "fatal: unable to access 'https://x-access-token:ghp_secrettoken12345@github.com/owner/repo.git': Could not resolve host"
	sanitized := sanitize(output, args)

	if strings.Contains(sanitized, "ghp_secrettoken12345") {
		t.Errorf("expected token to be redacted, got: %s", sanitized)
	}
	if !strings.Contains(sanitized, "***") {
		t.Errorf("expected *** placeholder, got: %s", sanitized)
	}
}

func TestSanitize_NoToken(t *testing.T) {
	args := []string{"status"}
	output := "On branch main"
	sanitized := sanitize(output, args)

	if sanitized != output {
		t.Errorf("expected no change when no token present, got: %s", sanitized)
	}
}

func TestCommitFile_CloneError(t *testing.T) {
	// Create a server that returns 401 to simulate auth failure
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create a client pointing to our test server
	c := New("bad-token", srv.URL)

	// Try to commit a file - this should fail during git clone
	// We use a non-existent repo path to ensure it fails
	err := c.CommitFile("owner", "repo", "test.txt", []byte("content"), "test commit")

	if err == nil {
		t.Fatal("expected error from git clone, got nil")
	}

	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("expected error to wrap 'git clone', got: %v", err)
	}
}

func TestRunGit_Success(t *testing.T) {
	// Use a simple git command that should succeed
	// git --version works without a repo
	err := runGit(".", "--version")
	if err != nil {
		t.Errorf("expected nil error for valid git command, got: %v", err)
	}
}

func TestRunGit_Failure(t *testing.T) {
	// Use an invalid git command
	err := runGit(".", "invalid-command-xyz")
	if err == nil {
		t.Fatal("expected error for invalid git command, got nil")
	}

	// Verify error message contains sanitized output
	errMsg := err.Error()
	if !strings.Contains(errMsg, "git invalid-command-xyz") {
		t.Errorf("expected error to contain command name, got: %v", err)
	}
}

func TestRunGit_TokenSanitization(t *testing.T) {
	// Run a git command that will fail and contain the token in args
	tokenURL := "https://x-access-token:ghp_test123@github.com/owner/repo.git"
	err := runGit("/nonexistent", "clone", tokenURL, ".")

	if err == nil {
		t.Fatal("expected error from git clone to nonexistent dir, got nil")
	}

	errMsg := err.Error()
	// The error message should have the sanitized URL in the command
	if !strings.Contains(errMsg, "x-access-token:***@github.com") {
		t.Errorf("expected sanitized URL in error message, got: %v", err)
	}
	if strings.Contains(errMsg, "ghp_test123") {
		t.Errorf("expected token to be redacted in error message, got: %v", err)
	}
}
