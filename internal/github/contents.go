package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FileContent is the GitHub Contents API response for a file.
type FileContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Size     int    `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	HTMLURL  string `json:"html_url"`
}

// GetFileContent fetches a file's content via the Contents API.
// Returns (content, blobSHA, error). blobSHA is needed for PUT updates.
// For files > 1 MB it falls back to the Git Blobs API.
func (c *Client) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	url := c.url("repos", owner, repo, "contents", path)
	if ref != "" {
		url += "?ref=" + ref
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return nil, "", err
	}

	var fc FileContent
	if err := json.NewDecoder(resp.Body).Decode(&fc); err != nil {
		return nil, "", err
	}

	if fc.Encoding == "none" && fc.Size > 1*1024*1024 {
		// File too large for the Contents API — use the Blobs API for raw bytes.
		data, err := c.getRawBlob(owner, repo, fc.SHA)
		return data, fc.SHA, err
	}

	// Decode base64 (GitHub wraps lines at 60 chars with newlines).
	cleaned := strings.ReplaceAll(fc.Content, "\n", "")
	data, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return nil, "", fmt.Errorf("decoding contents: %w", err)
	}
	return data, fc.SHA, nil
}

// getRawBlob downloads a blob by its SHA using the raw accept header.
// This bypasses the 1 MB base64 limit of the Contents API.
func (c *Client) getRawBlob(owner, repo, sha string) ([]byte, error) {
	url := c.url("repos", owner, repo, "git", "blobs", sha)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

// jsonDecode is a helper used by assets.go.
func jsonDecode(r io.Reader, out interface{}) error {
	return json.NewDecoder(r).Decode(out)
}
