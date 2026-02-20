package github

import (
	"fmt"
	"net/http"
)

// Release represents a GitHub Release.
type Release struct {
	ID      int64  `json:"id"`
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// GetReleaseByTag fetches a release by its tag name.
// Returns ErrNotFound if the tag does not exist.
func (c *Client) GetReleaseByTag(owner, repo, tag string) (*Release, error) {
	url := c.url("repos", owner, repo, "releases", "tags", tag)
	var r Release
	if err := c.doJSON(http.MethodGet, url, nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRelease creates a new release with the given tag.
func (c *Client) CreateRelease(owner, repo, tag, name string) (*Release, error) {
	url := c.url("repos", owner, repo, "releases")
	body := map[string]interface{}{
		"tag_name":         tag,
		"name":             name,
		"draft":            false,
		"prerelease":       false,
		"generate_release_notes": false,
	}
	var r Release
	if err := c.doJSON(http.MethodPost, url, body, &r); err != nil {
		return nil, fmt.Errorf("create release %q: %w", tag, err)
	}
	return &r, nil
}

// EnsureRelease returns the existing release for tag, creating it if absent.
func (c *Client) EnsureRelease(owner, repo, tag string) (*Release, error) {
	r, err := c.GetReleaseByTag(owner, repo, tag)
	if err == ErrNotFound {
		return c.CreateRelease(owner, repo, tag, tag)
	}
	return r, err
}
