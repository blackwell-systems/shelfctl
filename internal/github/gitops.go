package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// askpassScript is the content of the GIT_ASKPASS helper script.
// It echoes the token from the SHELFCTL_GIT_PASSWORD environment variable,
// avoiding the need to embed credentials in the clone URL.
const askpassScript = `#!/bin/sh
echo "$SHELFCTL_GIT_PASSWORD"
`

// createAskpassScript writes a temporary GIT_ASKPASS script and returns
// its path. The caller must remove the file when done.
func createAskpassScript() (string, error) {
	f, err := os.CreateTemp("", "shelfctl-askpass-*")
	if err != nil {
		return "", fmt.Errorf("create askpass script: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(askpassScript); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write askpass script: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close askpass script: %w", err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("chmod askpass script: %w", err)
	}
	return path, nil
}

// CommitFile clones the repo to a temp dir, writes filePath with content,
// commits with the given message, and pushes. The temp dir is cleaned up
// on return regardless of outcome.
func (c *Client) CommitFile(owner, repo, filePath string, content []byte, message string) error {
	tmpDir, err := os.MkdirTemp("", "shelfctl-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a GIT_ASKPASS script to pass the token securely via
	// environment variable instead of embedding it in the clone URL.
	askpassPath, err := createAskpassScript()
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(askpassPath) }()

	gitEnv := []string{
		"GIT_ASKPASS=" + askpassPath,
		"SHELFCTL_GIT_PASSWORD=" + c.token,
		"GIT_TERMINAL_PROMPT=0",
	}

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	if err := runGit(tmpDir, gitEnv, "clone", "--depth=1", cloneURL, "."); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	// Pull with rebase to handle any remote changes since clone
	if err := runGit(tmpDir, gitEnv, "pull", "--rebase"); err != nil {
		return fmt.Errorf("git pull --rebase: %w", err)
	}

	fullPath := filepath.Join(tmpDir, filepath.FromSlash(filePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, content, 0600); err != nil {
		return err
	}

	if err := runGit(tmpDir, nil, "config", "user.email", "shelfctl@local"); err != nil {
		return err
	}
	if err := runGit(tmpDir, nil, "config", "user.name", "shelfctl"); err != nil {
		return err
	}
	if err := runGit(tmpDir, nil, "add", filePath); err != nil {
		return err
	}
	if err := runGit(tmpDir, nil, "commit", "-m", message); err != nil {
		// If nothing changed (file content identical), git commit exits non-zero.
		// Check for this case and treat it as success.
		if nothingToCommit(tmpDir) {
			return nil
		}
		return err
	}
	if err := runGit(tmpDir, gitEnv, "push"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// nothingToCommit returns true if the working tree is clean (no staged changes).
func nothingToCommit(dir string) bool {
	return runGit(dir, nil, "diff", "--cached", "--quiet") == nil
}

func runGit(dir string, env []string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), sanitize(string(out)))
	}
	return nil
}

// sanitize removes tokens and sensitive data from git output before
// surfacing it to the user. Kept for backward safety in case any token
// leaks into git stderr/stdout through other paths.
func sanitize(s string) string {
	// Remove any x-access-token URLs that might appear in git output
	// (e.g., from remote URLs stored in git config).
	if idx := strings.Index(s, "x-access-token:"); idx >= 0 {
		rest := s[idx+len("x-access-token:"):]
		if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
			token := rest[:atIdx]
			if token != "" {
				s = strings.ReplaceAll(s, token, "***")
			}
		}
	}
	return s
}
