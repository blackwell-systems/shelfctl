package ingest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Source holds a resolved input ready for reading.
type Source struct {
	// Name is the original filename (no directory), used for asset naming.
	Name string
	// Size is the byte count if known in advance (-1 if unknown).
	Size int64
	// Open returns a new ReadCloser. May be called once.
	Open func() (io.ReadCloser, error)
}

// githubPathRe matches "github:owner/repo@ref:path/to/file"
var githubPathRe = regexp.MustCompile(`^github:([^/]+)/([^@]+)@([^:]+):(.+)$`)

// Resolve determines the type of input and returns a Source.
// Supported formats:
//
//	/path/to/file.pdf          — local file
//	https://example.com/f.pdf  — HTTP URL
//	github:owner/repo@ref:path — GitHub repo path (authenticated)
func Resolve(input string, ghToken, ghAPIBase string) (*Source, error) {
	switch {
	case strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://"):
		return resolveHTTP(input)
	case strings.HasPrefix(input, "github:"):
		return resolveGitHub(input, ghToken, ghAPIBase)
	default:
		return resolveFile(input)
	}
}

func resolveFile(path string) (*Source, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("%q is a directory", path)
	}
	return &Source{
		Name: filepath.Base(path),
		Size: fi.Size(),
		Open: func() (io.ReadCloser, error) { return os.Open(path) },
	}, nil
}

func resolveHTTP(url string) (*Source, error) {
	// HEAD the URL to try to get Content-Length and filename.
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Head(url)
	size := int64(-1)
	if err == nil && resp.StatusCode == http.StatusOK {
		if cl := resp.ContentLength; cl > 0 {
			size = cl
		}
		resp.Body.Close()
	}

	name := guessFilenameFromURL(url)

	return &Source{
		Name: name,
		Size: size,
		Open: func() (io.ReadCloser, error) {
			r, err := client.Get(url)
			if err != nil {
				return nil, err
			}
			if r.StatusCode != http.StatusOK {
				r.Body.Close()
				return nil, fmt.Errorf("GET %s: status %d", url, r.StatusCode)
			}
			return r.Body, nil
		},
	}, nil
}

func resolveGitHub(input, token, apiBase string) (*Source, error) {
	m := githubPathRe.FindStringSubmatch(input)
	if m == nil {
		return nil, fmt.Errorf("invalid github: path %q — expected github:owner/repo@ref:path/to/file", input)
	}
	owner, repo, ref, path := m[1], m[2], m[3], m[4]
	name := filepath.Base(path)

	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	contentURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", apiBase, owner, repo, path, ref)

	return &Source{
		Name: name,
		Size: -1,
		Open: func() (io.ReadCloser, error) {
			req, _ := http.NewRequest(http.MethodGet, contentURL, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/vnd.github.raw")
			req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			client := &http.Client{Timeout: 5 * time.Minute}
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return nil, fmt.Errorf("GitHub contents %s: status %d", path, resp.StatusCode)
			}
			return resp.Body, nil
		},
	}, nil
}

func guessFilenameFromURL(rawURL string) string {
	// Strip query string.
	if idx := strings.Index(rawURL, "?"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	base := filepath.Base(rawURL)
	if base == "" || base == "." || base == "/" {
		return "download"
	}
	return base
}
