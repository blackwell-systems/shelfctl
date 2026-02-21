package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Asset represents a GitHub Release asset.
type Asset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	URL                string `json:"url"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// ListReleaseAssets returns all assets for the given release.
func (c *Client) ListReleaseAssets(owner, repo string, releaseID int64) ([]Asset, error) {
	url := c.url("repos", owner, repo, "releases", fmt.Sprintf("%d", releaseID), "assets")
	var assets []Asset
	if err := c.doJSON(http.MethodGet, url, nil, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}

// FindAsset returns the first asset with the given name, or nil.
func (c *Client) FindAsset(owner, repo string, releaseID int64, name string) (*Asset, error) {
	assets, err := c.ListReleaseAssets(owner, repo, releaseID)
	if err != nil {
		return nil, err
	}
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i], nil
		}
	}
	return nil, nil
}

// DownloadAsset streams the content of a release asset.
// Caller is responsible for closing the returned ReadCloser.
func (c *Client) DownloadAsset(owner, repo string, assetID int64) (io.ReadCloser, error) {
	apiURL := c.url("repos", owner, repo, "releases", "assets", fmt.Sprintf("%d", assetID))

	// Use a client that strips auth on redirect away from github.com.
	client := newHTTPClientNoRedirect()

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// GitHub returns 302 to an S3 URL; the redirect client handles it.
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download asset: unexpected status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// UploadAsset uploads a file as a release asset.
// The reader must yield exactly size bytes.
func (c *Client) UploadAsset(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*Asset, error) {
	// Upload endpoint is on uploads.github.com, not api.github.com.
	uploadBase := strings.Replace(c.apiBase, "api.github.com", "uploads.github.com", 1)
	if uploadBase == c.apiBase {
		// Enterprise or custom — keep same host, different path.
		uploadBase = c.apiBase
	}
	uploadURL := fmt.Sprintf("%s/repos/%s/%s/releases/%d/assets?name=%s",
		uploadBase, owner, repo, releaseID, name)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	req, err := http.NewRequest(http.MethodPost, uploadURL, r)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := checkStatus(resp); err != nil {
		return nil, fmt.Errorf("upload asset %q: %w", name, err)
	}

	var asset Asset
	if err := jsonDecode(resp.Body, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// DeleteAsset removes a release asset.
func (c *Client) DeleteAsset(owner, repo string, assetID int64) error {
	url := c.url("repos", owner, repo, "releases", "assets", fmt.Sprintf("%d", assetID))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return checkStatus(resp)
}
