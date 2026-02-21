package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// Custom transport that strips auth when redirecting off github.com
	// (e.g. to S3 for asset downloads).
	transport := &redirectStripAuth{base: http.DefaultTransport}

	return &Client{
		token:   token,
		apiBase: apiBase,
		http: &http.Client{
			Timeout:   5 * time.Minute, // generous for large uploads
			Transport: transport,
		},
	}
}

// do executes the request with standard GitHub headers.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

// doJSON sends a request and decodes the JSON response into out.
func (c *Client) doJSON(method, url string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, bodyReader)
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
	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// redirectStripAuth is an http.RoundTripper that strips the Authorization
// header when the redirect target is not github.com (e.g. S3).
type redirectStripAuth struct {
	base http.RoundTripper
}

func (t *redirectStripAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req)
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
