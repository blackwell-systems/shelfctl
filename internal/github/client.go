package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.github.com"

// Client is an authenticated GitHub API client.
type Client struct {
	token   string
	apiBase string
	http    *http.Client
}

// New creates a Client with the given token and API base URL.
// If apiBase is empty, the public GitHub API is used.
func New(token, apiBase string) *Client {
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	// Strip trailing slash for consistent URL building.
	apiBase = strings.TrimRight(apiBase, "/")

	return &Client{
		token:   token,
		apiBase: apiBase,
		http: &http.Client{
			Timeout:   5 * time.Minute, // generous for large uploads
			Transport: http.DefaultTransport,
		},
	}
}

// do executes the request with standard GitHub headers.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	// Only set Accept if not already set (allow custom Accept headers)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("request to GitHub API timed out: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("request to GitHub API timed out: %w", err)
		}
		return nil, fmt.Errorf("unable to reach GitHub API: check your internet connection: %w", err)
	}
	return resp, nil
}

// doJSON sends a request and decodes the JSON response into out.
func (c *Client) doJSON(ctx context.Context, method, url string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := checkStatus(resp); err != nil {
		return err
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// APIBase returns the API base URL for this client.
func (c *Client) APIBase() string {
	return c.apiBase
}

// Token returns the authentication token for this client.
func (c *Client) Token() string {
	return c.token
}

// DirEntry is a single item returned by the GitHub Contents API for a directory.
type DirEntry struct {
	Type string `json:"type"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Size int    `json:"size"`
}

// ListDirContents returns the Contents API response for a directory path.
// Pass ref="" to use the default branch.
func (c *Client) ListDirContents(owner, repo, dirPath, ref string) ([]DirEntry, error) {
	u := c.url("repos", owner, repo, "contents", dirPath)
	if ref != "" {
		u += "?ref=" + ref
	}
	var entries []DirEntry
	if err := c.doJSON(context.Background(), http.MethodGet, u, nil, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// GetUser fetches the authenticated user's login and OAuth scopes.
// Returns (login, scopes, error). scopes is the raw X-OAuth-Scopes header
// value (e.g. "repo, gist"). Returns ErrUnauthorized if the token is invalid.
func (c *Client) GetUser() (login string, scopes string, err error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, c.url("user"), nil)
	if err != nil {
		return "", "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := checkStatus(resp); err != nil {
		return "", "", err
	}
	var u struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", "", err
	}
	return u.Login, resp.Header.Get("X-OAuth-Scopes"), nil
}

// url builds an API URL from path segments.
func (c *Client) url(parts ...string) string {
	return c.apiBase + "/" + strings.Join(parts, "/")
}

// checkStatus returns a typed error for non-2xx responses.
func checkStatus(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusConflict:
		return ErrConflict
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if resp.StatusCode >= 500 {
			return ErrServerError
		}
		return fmt.Errorf("github API error %d", resp.StatusCode)
	}
}

// newHTTPClientNoRedirect creates an http.Client that does NOT follow
// redirects automatically. Used for asset downloads where we handle the
// redirect ourselves to strip the auth header.
func newHTTPClientNoRedirect() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Strip auth when redirecting away from github.com.
			if !strings.Contains(req.URL.Host, "github.com") {
				req.Header.Del("Authorization")
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}
