package mockserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestNewServer(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}

	if server.fixtures == nil {
		t.Error("server.fixtures is nil")
	}
}

func TestStartStop(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	// Test Start
	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify server is running
	if !server.started {
		t.Error("server.started should be true after Start()")
	}

	url := server.URL()
	if url == "" {
		t.Error("URL() returned empty string after Start()")
	}

	// Test double start (should fail)
	err = server.Start()
	if err == nil {
		t.Error("Start() on already started server should return error")
	}

	// Test Stop
	err = server.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify server is stopped
	if server.started {
		t.Error("server.started should be false after Stop()")
	}

	// Test double stop (should fail)
	err = server.Stop()
	if err == nil {
		t.Error("Stop() on already stopped server should return error")
	}
}

func TestGetRelease(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	// Get first fixture to test with
	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/tags/v1.0.0"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var release map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if release["tag_name"] != "v1.0.0" {
		t.Errorf("Expected tag_name v1.0.0, got %v", release["tag_name"])
	}
}

func TestGetFileContent(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	// Get first fixture to test with
	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/contents/catalog.yml"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var content map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&content)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if content["name"] != "catalog.yml" {
		t.Errorf("Expected name catalog.yml, got %v", content["name"])
	}

	if content["type"] != "file" {
		t.Errorf("Expected type file, got %v", content["type"])
	}
}

func TestDownloadAsset(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	// Get first fixture with assets to test with
	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	if len(shelf.Assets) == 0 {
		t.Skip("No assets available to test with")
	}

	// Get first asset ID
	var assetID string
	for id := range shelf.Assets {
		assetID = id
		break
	}

	url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/assets/" + assetID

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected Content-Type application/pdf, got %s", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedData := shelf.Assets[assetID]
	if string(body) != string(expectedData) {
		t.Errorf("Response body doesn't match expected asset data")
	}
}

func TestConcurrentRequests(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	baseURL := server.URL()

	// Make 10 concurrent requests
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			url := baseURL + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/tags/v1.0.0"
			resp, err := http.Get(url)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- err
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

func TestNotFoundEndpoints(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	tests := []struct {
		name string
		path string
	}{
		{"invalid repo", "/repos/invalid/invalid/releases/tags/v1.0.0"},
		{"invalid path", "/repos/owner/repo/invalid"},
		{"invalid asset", "/repos/owner/repo/releases/assets/nonexistent"},
		{"invalid content path", "/repos/owner/repo/contents/nonexistent.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := server.URL() + tt.path
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("GET request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected status 404, got %d for path %s", resp.StatusCode, tt.path)
			}
		})
	}
}

func TestGetReleaseAssets(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/12345/assets"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var assets []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&assets)
	if err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	// Verify assets list structure
	if len(assets) != len(shelf.Assets) {
		t.Errorf("Expected %d assets, got %d", len(shelf.Assets), len(assets))
	}

	for _, asset := range assets {
		if asset["id"] == nil {
			t.Error("Asset missing id field")
		}
		if asset["name"] == nil {
			t.Error("Asset missing name field")
		}
		if asset["url"] == nil {
			t.Error("Asset missing url field")
		}

		// Verify URL format
		if url, ok := asset["url"].(string); ok {
			if !strings.Contains(url, "/releases/assets/") {
				t.Errorf("Asset URL has wrong format: %s", url)
			}
		}
	}
}
