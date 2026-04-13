package github

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSanitize_RedactsToken(t *testing.T) {
	output := "fatal: unable to access 'https://x-access-token:ghp_secrettoken12345@github.com/owner/repo.git': Could not resolve host"
	sanitized := sanitize(output)

	if strings.Contains(sanitized, "ghp_secrettoken12345") {
		t.Errorf("expected token to be redacted, got: %s", sanitized)
	}
	if !strings.Contains(sanitized, "***") {
		t.Errorf("expected *** placeholder, got: %s", sanitized)
	}
}

func TestSanitize_NoToken(t *testing.T) {
	output := "On branch main"
	sanitized := sanitize(output)

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
	err := c.CommitFile("owner", "repo", "test.txt", []byte("content"), "test commit")

	if err == nil {
		t.Fatal("expected error from git clone, got nil")
	}

	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("expected error to wrap 'git clone', got: %v", err)
	}
}

func TestCommitFile_CloneURL_NoToken(t *testing.T) {
	// Verify that CommitFile does not embed the token in the clone URL.
	// We do this by attempting a clone (which will fail) and checking
	// that the error message does not contain the token.
	c := New("ghp_supersecrettoken999", "https://api.github.com")

	err := c.CommitFile("owner", "repo", "test.txt", []byte("content"), "test commit")
	if err == nil {
		t.Fatal("expected error from git clone, got nil")
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "ghp_supersecrettoken999") {
		t.Errorf("token should NOT appear in error message, got: %v", errMsg)
	}
	// The clone URL should be a plain HTTPS URL without credentials
	if strings.Contains(errMsg, "x-access-token:") {
		t.Errorf("clone URL should not contain x-access-token, got: %v", errMsg)
	}
}

func TestRunGit_Success(t *testing.T) {
	// Use a simple git command that should succeed
	// git --version works without a repo
	err := runGit(".", nil, "--version")
	if err != nil {
		t.Errorf("expected nil error for valid git command, got: %v", err)
	}
}

func TestRunGit_Failure(t *testing.T) {
	// Use an invalid git command
	err := runGit(".", nil, "invalid-command-xyz")
	if err == nil {
		t.Fatal("expected error for invalid git command, got nil")
	}

	// Verify error message contains command name
	errMsg := err.Error()
	if !strings.Contains(errMsg, "git invalid-command-xyz") {
		t.Errorf("expected error to contain command name, got: %v", err)
	}
}

func TestRunGit_WithEnv(t *testing.T) {
	// Verify runGit passes environment variables to the command
	err := runGit(".", []string{"GIT_TERMINAL_PROMPT=0"}, "--version")
	if err != nil {
		t.Errorf("expected nil error for valid git command with env, got: %v", err)
	}
}

func TestRunGit_TokenNotInArgs(t *testing.T) {
	// With the new approach, the token should never be in args.
	// Run a failing command and verify no token appears in error output.
	err := runGit("/nonexistent", nil, "clone", "https://github.com/owner/repo.git", ".")
	if err == nil {
		t.Fatal("expected error from git clone to nonexistent dir, got nil")
	}

	errMsg := err.Error()
	// The URL in the args should be a plain HTTPS URL
	if strings.Contains(errMsg, "x-access-token:") {
		t.Errorf("expected no token in args/error, got: %v", errMsg)
	}
}

func TestCreateAskpassScript(t *testing.T) {
	path, err := createAskpassScript()
	if err != nil {
		t.Fatalf("failed to create askpass script: %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	// Verify the file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("askpass script not found: %v", err)
	}

	// Verify it is executable (mode 0700)
	mode := info.Mode().Perm()
	if mode&0100 == 0 {
		t.Errorf("askpass script should be executable, got mode: %o", mode)
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read askpass script: %v", err)
	}
	if !strings.Contains(string(content), "SHELFCTL_GIT_PASSWORD") {
		t.Errorf("askpass script should reference SHELFCTL_GIT_PASSWORD, got: %s", content)
	}
	if !strings.HasPrefix(string(content), "#!/bin/sh") {
		t.Errorf("askpass script should start with shebang, got: %s", content)
	}
}

func TestCreateAskpassScript_Cleanup(t *testing.T) {
	// Verify the script can be created and cleaned up
	path, err := createAskpassScript()
	if err != nil {
		t.Fatalf("failed to create askpass script: %v", err)
	}

	// Remove it (simulating defer cleanup in CommitFile)
	if err := os.Remove(path); err != nil {
		t.Fatalf("failed to remove askpass script: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected askpass script to be removed, but it still exists")
	}
}
