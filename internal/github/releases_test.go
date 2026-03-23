package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestCreateRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body["tag_name"] != "v1.0.0" {
			t.Errorf("expected tag_name=v1.0.0, got %v", body["tag_name"])
		}
		if body["name"] != "Release 1.0.0" {
			t.Errorf("expected name='Release 1.0.0', got %v", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := Release{
			ID:      123456,
			TagName: "v1.0.0",
			Name:    "Release 1.0.0",
			HTMLURL: "https://github.com/owner/repo/releases/tag/v1.0.0",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})

	_, c := newFakeServer(t, mux)

	release, err := c.CreateRelease("owner", "repo", "v1.0.0", "Release 1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if release.ID != 123456 {
		t.Errorf("expected ID=123456, got %d", release.ID)
	}
	if release.TagName != "v1.0.0" {
		t.Errorf("expected TagName=v1.0.0, got %s", release.TagName)
	}
	if release.Name != "Release 1.0.0" {
		t.Errorf("expected Name='Release 1.0.0', got %s", release.Name)
	}
}

func TestGetReleaseByTag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := Release{
			ID:      123456,
			TagName: "v1.0.0",
			Name:    "Release 1.0.0",
			HTMLURL: "https://github.com/owner/repo/releases/tag/v1.0.0",
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	})

	_, c := newFakeServer(t, mux)

	release, err := c.GetReleaseByTag("owner", "repo", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if release.ID != 123456 {
		t.Errorf("expected ID=123456, got %d", release.ID)
	}
	if release.TagName != "v1.0.0" {
		t.Errorf("expected TagName=v1.0.0, got %s", release.TagName)
	}
	if release.Name != "Release 1.0.0" {
		t.Errorf("expected Name='Release 1.0.0', got %s", release.Name)
	}
}

func TestEnsureRelease(t *testing.T) {
	t.Run("ExistingRelease", func(t *testing.T) {
		mux := http.NewServeMux()
		getCallCount := 0
		postCallCount := 0

		mux.HandleFunc("/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
			getCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := Release{
				ID:      123456,
				TagName: "v1.0.0",
				Name:    "Release 1.0.0",
				HTMLURL: "https://github.com/owner/repo/releases/tag/v1.0.0",
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		})

		mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			postCallCount++
			t.Error("should not create release when it already exists")
		})

		_, c := newFakeServer(t, mux)

		release, err := c.EnsureRelease("owner", "repo", "v1.0.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if release.ID != 123456 {
			t.Errorf("expected ID=123456, got %d", release.ID)
		}
		if getCallCount != 1 {
			t.Errorf("expected 1 GET call, got %d", getCallCount)
		}
		if postCallCount != 0 {
			t.Errorf("expected 0 POST calls, got %d", postCallCount)
		}
	})

	t.Run("CreateNewRelease", func(t *testing.T) {
		mux := http.NewServeMux()
		getCallCount := 0
		postCallCount := 0

		mux.HandleFunc("/repos/owner/repo/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
			getCallCount++
			w.WriteHeader(http.StatusNotFound)
		})

		mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			postCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			resp := Release{
				ID:      123456,
				TagName: "v1.0.0",
				Name:    "v1.0.0",
				HTMLURL: "https://github.com/owner/repo/releases/tag/v1.0.0",
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		})

		_, c := newFakeServer(t, mux)

		release, err := c.EnsureRelease("owner", "repo", "v1.0.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if release.ID != 123456 {
			t.Errorf("expected ID=123456, got %d", release.ID)
		}
		if getCallCount != 1 {
			t.Errorf("expected 1 GET call, got %d", getCallCount)
		}
		if postCallCount != 1 {
			t.Errorf("expected 1 POST call, got %d", postCallCount)
		}
	})
}

func TestCreateReleaseConflict(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	_, c := newFakeServer(t, mux)

	_, err := c.CreateRelease("owner", "repo", "v1.0.0", "Release 1.0.0")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestGetReleaseNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/tags/v99.99.99", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, c := newFakeServer(t, mux)

	_, err := c.GetReleaseByTag("owner", "repo", "v99.99.99")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
