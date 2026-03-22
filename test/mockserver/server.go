package mockserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"gopkg.in/yaml.v3"
)

// Server defines the mock server interface.
type Server interface {
	Start() error
	Stop() error
	URL() string
}

// MockServer implements a mock GitHub API server for testing.
type MockServer struct {
	server   *httptest.Server
	fixtures *fixtures.FixtureSet
	mu       sync.RWMutex
	started  bool
}

// NewServer creates a new mock GitHub API server.
func NewServer() (*MockServer, error) {
	ms := &MockServer{
		fixtures: fixtures.DefaultFixtures(),
	}

	mux := http.NewServeMux()

	// GET /repos/{owner}/{repo}/releases/tags/{tag}
	mux.HandleFunc("/repos/", ms.handleReposRequest)

	ms.server = httptest.NewUnstartedServer(mux)

	return ms, nil
}

// Start starts the mock server.
func (ms *MockServer) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.started {
		return fmt.Errorf("server already started")
	}

	ms.server.Start()
	ms.started = true
	return nil
}

// Stop stops the mock server.
func (ms *MockServer) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if !ms.started {
		return fmt.Errorf("server not started")
	}

	ms.server.Close()
	ms.started = false
	return nil
}

// URL returns the base URL of the mock server.
func (ms *MockServer) URL() string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.server == nil {
		return ""
	}
	return ms.server.URL
}

// handleReposRequest routes /repos/* requests to the appropriate handler.
func (ms *MockServer) handleReposRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Parse path: /repos/{owner}/{repo}/...
	parts := strings.Split(strings.TrimPrefix(path, "/repos/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	owner := parts[0]
	repo := parts[1]

	// Route based on remaining path
	if len(parts) >= 4 && parts[2] == "releases" && parts[3] == "tags" {
		// GET /repos/{owner}/{repo}/releases/tags/{tag}
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		tag := parts[4]
		ms.handleGetRelease(w, r, owner, repo, tag)
		return
	}

	if len(parts) >= 5 && parts[2] == "releases" && parts[4] == "assets" {
		// GET /repos/{owner}/{repo}/releases/{id}/assets
		releaseID := parts[3]
		ms.handleGetReleaseAssets(w, r, owner, repo, releaseID)
		return
	}

	if len(parts) >= 4 && parts[2] == "releases" && parts[3] == "assets" {
		// GET /repos/{owner}/{repo}/releases/assets/{id}
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		assetID := parts[4]
		ms.handleDownloadAsset(w, r, owner, repo, assetID)
		return
	}

	if len(parts) >= 3 && parts[2] == "contents" {
		// GET /repos/{owner}/{repo}/contents/{path}
		contentPath := strings.Join(parts[3:], "/")
		ms.handleGetFileContent(w, r, owner, repo, contentPath)
		return
	}

	http.NotFound(w, r)
}

// handleGetRelease handles GET /repos/{owner}/{repo}/releases/tags/{tag}
func (ms *MockServer) handleGetRelease(w http.ResponseWriter, r *http.Request, owner, repo, tag string) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Find matching fixture
	for _, shelf := range ms.fixtures.Shelves {
		if shelf.Owner == owner && shelf.Repo == repo {
			// Return a release object
			release := map[string]interface{}{
				"id":       12345,
				"tag_name": tag,
				"name":     fmt.Sprintf("Release %s", tag),
				"assets":   []interface{}{},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(release)
			return
		}
	}

	http.NotFound(w, r)
}

// handleGetReleaseAssets handles GET /repos/{owner}/{repo}/releases/{id}/assets
func (ms *MockServer) handleGetReleaseAssets(w http.ResponseWriter, r *http.Request, owner, repo, releaseID string) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Find matching fixture
	for _, shelf := range ms.fixtures.Shelves {
		if shelf.Owner == owner && shelf.Repo == repo {
			// Build asset list from books
			assets := []map[string]interface{}{}
			for bookID := range shelf.Assets {
				assets = append(assets, map[string]interface{}{
					"id":   bookID,
					"name": fmt.Sprintf("%s.pdf", bookID),
					"url":  fmt.Sprintf("%s/repos/%s/%s/releases/assets/%s", ms.URL(), owner, repo, bookID),
				})
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(assets)
			return
		}
	}

	http.NotFound(w, r)
}

// handleGetFileContent handles GET /repos/{owner}/{repo}/contents/{path}
func (ms *MockServer) handleGetFileContent(w http.ResponseWriter, r *http.Request, owner, repo, contentPath string) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Only handle catalog.yml requests
	if contentPath != "catalog.yml" {
		http.NotFound(w, r)
		return
	}

	// Find matching fixture
	for _, shelf := range ms.fixtures.Shelves {
		if shelf.Owner == owner && shelf.Repo == repo {
			// Marshal books to YAML
			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			enc.SetIndent(2)
			if err := enc.Encode(shelf.Books); err != nil {
				http.Error(w, "failed to encode catalog", http.StatusInternalServerError)
				return
			}

			// Base64 encode the YAML content (GitHub API format)
			encodedContent := base64.StdEncoding.EncodeToString(buf.Bytes())

			content := map[string]interface{}{
				"name":    "catalog.yml",
				"path":    contentPath,
				"type":    "file",
				"content": encodedContent,
				"sha":     "mock-sha256-hash", // GitHub API includes git blob SHA
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(content)
			return
		}
	}

	http.NotFound(w, r)
}

// handleDownloadAsset handles GET /repos/{owner}/{repo}/releases/assets/{id}
func (ms *MockServer) handleDownloadAsset(w http.ResponseWriter, r *http.Request, owner, repo, assetID string) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Find matching fixture and asset
	for _, shelf := range ms.fixtures.Shelves {
		if shelf.Owner == owner && shelf.Repo == repo {
			if assetData, ok := shelf.Assets[assetID]; ok {
				w.Header().Set("Content-Type", "application/pdf")
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", assetID))
				_, _ = w.Write(assetData)
				return
			}
		}
	}

	http.NotFound(w, r)
}

