package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommitFile clones the repo to a temp dir, writes filePath with content,
// commits with the given message, and pushes. The temp dir is cleaned up
// on return regardless of outcome.
func (c *Client) CommitFile(owner, repo, filePath string, content []byte, message string) error {
	tmpDir, err := os.MkdirTemp("", "shelfctl-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git",
		c.token, owner, repo)

	if err := runGit(tmpDir, "clone", "--depth=1", cloneURL, "."); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	fullPath := filepath.Join(tmpDir, filepath.FromSlash(filePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, content, 0600); err != nil {
		return err
	}

	if err := runGit(tmpDir, "config", "user.email", "shelfctl@local"); err != nil {
		return err
	}
	if err := runGit(tmpDir, "config", "user.name", "shelfctl"); err != nil {
		return err
	}
	if err := runGit(tmpDir, "add", filePath); err != nil {
		return err
	}
	if err := runGit(tmpDir, "commit", "-m", message); err != nil {
		return err
	}
	if err := runGit(tmpDir, "push"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// ReadFile clones the repo (shallow) and reads filePath, returning its bytes.
// For simple catalog reads, prefer GetFileContent which avoids a clone.
func (c *Client) ReadFile(owner, repo, filePath string) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "shelfctl-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git",
		c.token, owner, repo)
	if err := runGit(tmpDir, "clone", "--depth=1", cloneURL, "."); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(tmpDir, filepath.FromSlash(filePath)))
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Suppress token from leaking into error messages.
	sanitizedArgs := make([]string, len(args))
	for i, a := range args {
		if strings.Contains(a, "x-access-token:") {
			sanitizedArgs[i] = "https://x-access-token:***@github.com/..."
		} else {
			sanitizedArgs[i] = a
		}
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s", strings.Join(sanitizedArgs, " "), sanitize(string(out), args))
	}
	return nil
}

// sanitize removes the token from git output before surfacing it to the user.
func sanitize(s string, args []string) string {
	for _, a := range args {
		if strings.Contains(a, "x-access-token:") {
			// Extract token from URL.
			if idx := strings.Index(a, "x-access-token:"); idx >= 0 {
				rest := a[idx+len("x-access-token:"):]
				if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
					token := rest[:atIdx]
					s = strings.ReplaceAll(s, token, "***")
				}
			}
		}
	}
	return s
}
