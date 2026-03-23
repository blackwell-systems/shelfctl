package mockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/blackwell-systems/shelfctl/test/fixtures"
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

	// Get first asset ID (bookID is the string key, but URL needs numeric hash)
	var bookID string
	for id := range shelf.Assets {
		bookID = id
		break
	}

	// Convert bookID to hashed numeric ID for URL (matching real GitHub API behavior)
	hashedID := hashString(bookID)
	url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/assets/" + fmt.Sprintf("%d", hashedID)

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

	expectedData := shelf.Assets[bookID]
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

// TestAssetEndpointSchema verifies that Asset JSON has int64 ID field
func TestAssetEndpointSchema(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
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

	if len(assets) == 0 {
		t.Skip("No assets returned to verify schema")
	}

	// Verify each asset has int64 ID (represented as float64 in JSON unmarshaling)
	for i, asset := range assets {
		id, ok := asset["id"]
		if !ok {
			t.Errorf("Asset %d missing id field", i)
			continue
		}

		// JSON numbers unmarshal to float64, but we need to verify it's an integer
		idFloat, ok := id.(float64)
		if !ok {
			t.Errorf("Asset %d id field is not numeric: type=%T", i, id)
			continue
		}

		// Verify it's a whole number (int64 compatible)
		if idFloat != float64(int64(idFloat)) {
			t.Errorf("Asset %d id is not an integer: %v", i, idFloat)
		}

		// Verify other required fields
		requiredFields := []string{"name", "size", "url", "browser_download_url", "content_type"}
		for _, field := range requiredFields {
			if _, ok := asset[field]; !ok {
				t.Errorf("Asset %d missing required field: %s", i, field)
			}
		}
	}
}

// TestReleaseEndpointSchema verifies that Release JSON returns a single object
func TestReleaseEndpointSchema(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

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

	// Attempt to decode as array first (should fail)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var asArray []interface{}
	if err := json.Unmarshal(body, &asArray); err == nil {
		t.Error("Release endpoint returned array instead of single object")
	}

	// Now decode as object (should succeed)
	var release map[string]interface{}
	if err := json.Unmarshal(body, &release); err != nil {
		t.Fatalf("Failed to decode release as object: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"id", "tag_name", "name", "assets"}
	for _, field := range requiredFields {
		if _, ok := release[field]; !ok {
			t.Errorf("Release missing required field: %s", field)
		}
	}

	// Verify assets is an array
	if assets, ok := release["assets"]; ok {
		if _, ok := assets.([]interface{}); !ok {
			t.Errorf("Release assets field is not an array: type=%T", assets)
		}
	}
}

// TestAssetDownload tests the asset download endpoint with real fixtures
func TestAssetDownload(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]
	if len(shelf.Assets) == 0 {
		t.Skip("No assets available to test with")
	}

	// Test download for each asset
	for bookID, expectedData := range shelf.Assets {
		t.Run(bookID, func(t *testing.T) {
			hashedID := hashString(bookID)
			url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/assets/" + fmt.Sprintf("%d", hashedID)

			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("GET request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Verify content type
			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/pdf" {
				t.Errorf("Expected Content-Type application/pdf, got %s", contentType)
			}

			// Verify content disposition header
			contentDisposition := resp.Header.Get("Content-Disposition")
			expectedDisposition := fmt.Sprintf("attachment; filename=%s.pdf", bookID)
			if contentDisposition != expectedDisposition {
				t.Errorf("Expected Content-Disposition %s, got %s", expectedDisposition, contentDisposition)
			}

			// Verify body matches expected data
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if !bytes.Equal(body, expectedData) {
				t.Errorf("Response body doesn't match expected asset data for %s", bookID)
			}
		})
	}
}

// TestAssetNotFound tests 404 handling for missing assets
func TestAssetNotFound(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	shelf := server.fixtures.Shelves[0]

	tests := []struct {
		name    string
		assetID string
	}{
		{"nonexistent numeric ID", "999999999"},
		{"zero ID", "0"},
		{"negative ID", "-1"},
		{"very large ID", "9223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := server.URL() + "/repos/" + shelf.Owner + "/" + shelf.Repo + "/releases/assets/" + tt.assetID

			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("GET request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected status 404 for missing asset, got %d", resp.StatusCode)
			}
		})
	}
}

// TestMultipleAssets tests releases with multiple assets
func TestMultipleAssets(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Errorf("failed to stop mock server: %v", err)
		}
	}()

	if len(server.fixtures.Shelves) == 0 {
		t.Skip("No fixtures available to test with")
	}

	// Find a shelf with multiple assets
	var multiAssetShelf *fixtures.ShelfFixture

	for i := range server.fixtures.Shelves {
		if len(server.fixtures.Shelves[i].Assets) > 1 {
			shelf := server.fixtures.Shelves[i]
			multiAssetShelf = &shelf
			break
		}
	}

	if multiAssetShelf == nil {
		t.Skip("No shelves with multiple assets found")
	}

	url := server.URL() + "/repos/" + multiAssetShelf.Owner + "/" + multiAssetShelf.Repo + "/releases/12345/assets"

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

	// Verify we got multiple assets
	if len(assets) != len(multiAssetShelf.Assets) {
		t.Errorf("Expected %d assets, got %d", len(multiAssetShelf.Assets), len(assets))
	}

	if len(assets) <= 1 {
		t.Error("Expected multiple assets, got <= 1")
	}

	// Verify each asset has unique ID
	seenIDs := make(map[float64]bool)
	seenNames := make(map[string]bool)

	for i, asset := range assets {
		// Verify unique ID
		id, ok := asset["id"].(float64)
		if !ok {
			t.Errorf("Asset %d has non-numeric id", i)
			continue
		}

		if seenIDs[id] {
			t.Errorf("Duplicate asset ID found: %v", id)
		}
		seenIDs[id] = true

		// Verify unique name
		name, ok := asset["name"].(string)
		if !ok {
			t.Errorf("Asset %d has non-string name", i)
			continue
		}

		if seenNames[name] {
			t.Errorf("Duplicate asset name found: %s", name)
		}
		seenNames[name] = true

		// Verify all required fields are present
		requiredFields := []string{"id", "name", "size", "url", "browser_download_url", "content_type"}
		for _, field := range requiredFields {
			if _, ok := asset[field]; !ok {
				t.Errorf("Asset %d (%s) missing required field: %s", i, name, field)
			}
		}
	}

	// Verify all IDs are unique (no collisions in hash function)
	if len(seenIDs) != len(assets) {
		t.Errorf("Expected %d unique IDs, got %d", len(assets), len(seenIDs))
	}

	// Verify all names are unique
	if len(seenNames) != len(assets) {
		t.Errorf("Expected %d unique names, got %d", len(assets), len(seenNames))
	}
}
