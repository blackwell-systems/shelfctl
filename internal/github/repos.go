package github

import (
	"fmt"
	"net/http"
)

// Repo represents a GitHub repository.
type Repo struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
	Private  bool   `json:"private"`
}

// GetRepo fetches repository metadata. Returns ErrNotFound if absent.
func (c *Client) GetRepo(owner, repo string) (*Repo, error) {
	url := c.url("repos", owner, repo)
	var r Repo
	if err := c.doJSON(http.MethodGet, url, nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRepo creates a new repository under the authenticated user.
func (c *Client) CreateRepo(name string, private bool) (*Repo, error) {
	url := c.url("user", "repos")
	body := map[string]interface{}{
		"name":        name,
		"private":     private,
		"auto_init":   true, // creates an initial commit so cloning works
		"description": "shelfctl shelf",
	}
	var r Repo
	if err := c.doJSON(http.MethodPost, url, body, &r); err != nil {
		return nil, fmt.Errorf("create repo %q: %w", name, err)
	}
	return &r, nil
}

// RepoExists returns true if the repo exists and is accessible.
func (c *Client) RepoExists(owner, repo string) (bool, error) {
	_, err := c.GetRepo(owner, repo)
	if err == ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

// DeleteRepo permanently deletes a repository.
// This is a destructive operation that cannot be undone.
func (c *Client) DeleteRepo(owner, repo string) error {
	url := c.url("repos", owner, repo)
	if err := c.doJSON(http.MethodDelete, url, nil, nil); err != nil {
		return fmt.Errorf("delete repo %s/%s: %w", owner, repo, err)
	}
	return nil
}
