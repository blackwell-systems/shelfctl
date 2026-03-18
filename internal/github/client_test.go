package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newFakeServer spins up an httptest.Server with a mux that handles
// the GitHub API endpoints needed by each test case.
// Returns server + *Client pointed at server.URL.
func newFakeServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, *Client) {
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New("test-token", srv.URL)
	return srv, c
}

func TestNew_DefaultsAPIBase(t *testing.T) {
	c := New("token", "")
	if c.apiBase != defaultAPIBase {
		t.Errorf("expected apiBase=%q, got %q", defaultAPIBase, c.apiBase)
	}
	if c.token != "token" {
		t.Errorf("expected token=%q, got %q", "token", c.token)
	}
	if c.http == nil {
		t.Error("expected non-nil http client")
	}
}

func TestNew_StripTrailingSlash(t *testing.T) {
	c := New("token", "https://example.com/api/v3/")
	if c.apiBase != "https://example.com/api/v3" {
		t.Errorf("expected trailing slash stripped, got %q", c.apiBase)
	}
}

func TestNew_CustomAPIBase(t *testing.T) {
	c := New("token", "https://github.enterprise.com/api/v3")
	if c.apiBase != "https://github.enterprise.com/api/v3" {
		t.Errorf("expected custom apiBase preserved, got %q", c.apiBase)
	}
}

func TestClientDo_SetsAuthHeader(t *testing.T) {
	mux := http.NewServeMux()
	var capturedAuth string
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})
	_, c := newFakeServer(t, mux)

	req, _ := http.NewRequest(http.MethodGet, c.url("test"), nil)
	_, err := c.do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Bearer test-token"
	if capturedAuth != expected {
		t.Errorf("expected Authorization=%q, got %q", expected, capturedAuth)
	}
}

func TestClientDo_SetsGitHubAPIVersion(t *testing.T) {
	mux := http.NewServeMux()
	var capturedVersion string
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedVersion = r.Header.Get("X-GitHub-Api-Version")
		w.WriteHeader(http.StatusOK)
	})
	_, c := newFakeServer(t, mux)

	req, _ := http.NewRequest(http.MethodGet, c.url("test"), nil)
	_, err := c.do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "2022-11-28"
	if capturedVersion != expected {
		t.Errorf("expected X-GitHub-Api-Version=%q, got %q", expected, capturedVersion)
	}
}

func TestCheckStatus_OK(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"202 Accepted", http.StatusAccepted},
		{"204 NoContent", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status}
			if err := checkStatus(resp); err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
		})
	}
}

func TestCheckStatus_NotFound(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusNotFound}
	err := checkStatus(resp)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCheckStatus_Unauthorized(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusUnauthorized}
	err := checkStatus(resp)
	if err != ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestCheckStatus_Forbidden(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusForbidden}
	err := checkStatus(resp)
	if err != ErrForbidden {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCheckStatus_Conflict(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusConflict}
	err := checkStatus(resp)
	if err != ErrConflict {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCheckStatus_UnknownError(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	err := checkStatus(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "github API error 500"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDoJSON_DecodeResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"message":"hello"}`)
	})
	_, c := newFakeServer(t, mux)

	var result struct {
		Message string `json:"message"`
	}
	err := c.doJSON(http.MethodGet, c.url("test"), nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "hello" {
		t.Errorf("expected message=%q, got %q", "hello", result.Message)
	}
}

func TestDoJSON_NilBody(t *testing.T) {
	mux := http.NewServeMux()
	var bodyWasNil bool
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		bodyWasNil = (r.Body == nil || r.ContentLength == 0)
		w.WriteHeader(http.StatusOK)
	})
	_, c := newFakeServer(t, mux)

	err := c.doJSON(http.MethodGet, c.url("test"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When body is nil, request should have no/empty body
	if !bodyWasNil {
		t.Error("expected request with nil body")
	}
}

func TestDoJSON_ErrorOnBadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{invalid json}`)
	})
	_, c := newFakeServer(t, mux)

	var result struct {
		Message string `json:"message"`
	}
	err := c.doJSON(http.MethodGet, c.url("test"), nil, &result)
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
	// Just verify we got a JSON decode error
	if !strings.Contains(err.Error(), "invalid") &&
		!strings.Contains(err.Error(), "character") {
		// json.Decoder returns errors containing these strings for invalid JSON
		t.Errorf("expected JSON decode error, got: %v", err)
	}
}
