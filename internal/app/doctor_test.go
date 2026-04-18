package app

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/config"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
)

// newFakeGitHubServer creates a minimal httptest server that serves /user and
// optionally /repos/{owner}/{name}.
func newFakeGitHubServer(t *testing.T, login, scopes string, repoOwner, repoName string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-OAuth-Scopes", scopes)
		_, _ = fmt.Fprintf(w, `{"login":%q}`, login)
	})
	if repoOwner != "" && repoName != "" {
		path := fmt.Sprintf("/repos/%s/%s", repoOwner, repoName)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"id":1,"full_name":%q}`, repoOwner+"/"+repoName)
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// ---------------------------------------------------------------------------
// checkConfig tests
// ---------------------------------------------------------------------------

func TestCheckConfig_Missing(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yml")
	result, cfg := checkConfig(cfgPath)
	if result.Status != "fail" {
		t.Errorf("expected status fail, got %s", result.Status)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
	if !strings.Contains(result.Message, "not found") {
		t.Errorf("expected 'not found' in message, got %s", result.Message)
	}
}

func TestCheckConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	cfgYAML := `github:
  owner: testowner
  token_env: TEST_TOKEN_VALID
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELFCTL_CONFIG", cfgPath)

	result, cfg := checkConfig(cfgPath)
	if result.Status != "pass" {
		t.Errorf("expected status pass, got %s: %s", result.Status, result.Message)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestCheckConfig_MissingOwner(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	cfgYAML := `github:
  token_env: TEST_TOKEN_OWNER
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELFCTL_CONFIG", cfgPath)

	result, _ := checkConfig(cfgPath)
	if result.Status != "warn" {
		t.Errorf("expected status warn, got %s: %s", result.Status, result.Message)
	}
}

// ---------------------------------------------------------------------------
// checkToken tests
// ---------------------------------------------------------------------------

func TestCheckToken_Missing(t *testing.T) {
	cfg := &config.Config{GitHub: config.GitHubConfig{TokenEnv: "TEST_TOKEN"}}
	result := checkToken(cfg)
	if result.Status != "fail" {
		t.Errorf("expected status fail, got %s", result.Status)
	}
	if !strings.Contains(result.Message, "TEST_TOKEN") {
		t.Errorf("expected message to mention TEST_TOKEN, got %s", result.Message)
	}
}

func TestCheckToken_Set(t *testing.T) {
	cfg := &config.Config{GitHub: config.GitHubConfig{TokenEnv: "TEST_TOKEN", Token: "tok123"}}
	result := checkToken(cfg)
	if result.Status != "pass" {
		t.Errorf("expected status pass, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// checkAPIAndScopes tests
// ---------------------------------------------------------------------------

func TestCheckAPIAndScopes_Success(t *testing.T) {
	srv := newFakeGitHubServer(t, "octocat", "repo, gist", "", "")
	gh := ghclient.New("fake-token", srv.URL)

	connResult, scopeResult := checkAPIAndScopes(gh, srv.URL)
	if connResult.Status != "pass" {
		t.Errorf("expected conn pass, got %s: %s", connResult.Status, connResult.Message)
	}
	if scopeResult.Status != "pass" {
		t.Errorf("expected scope pass, got %s: %s", scopeResult.Status, scopeResult.Message)
	}
}

func TestCheckAPIAndScopes_NoScope(t *testing.T) {
	srv := newFakeGitHubServer(t, "octocat", "gist", "", "")
	gh := ghclient.New("fake-token", srv.URL)

	_, scopeResult := checkAPIAndScopes(gh, srv.URL)
	if scopeResult.Status != "warn" {
		t.Errorf("expected scope warn, got %s: %s", scopeResult.Status, scopeResult.Message)
	}
}

func TestCheckAPIAndScopes_Unauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"message":"Bad credentials"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	gh := ghclient.New("bad-token", srv.URL)
	connResult, scopeResult := checkAPIAndScopes(gh, srv.URL)
	if connResult.Status != "fail" {
		t.Errorf("expected conn fail, got %s: %s", connResult.Status, connResult.Message)
	}
	if scopeResult.Status != "skip" {
		t.Errorf("expected scope skip, got %s: %s", scopeResult.Status, scopeResult.Message)
	}
}

// ---------------------------------------------------------------------------
// checkShelf tests
// ---------------------------------------------------------------------------

func TestCheckShelf_Found(t *testing.T) {
	srv := newFakeGitHubServer(t, "octocat", "repo", "testowner", "shelf-test")
	gh := ghclient.New("fake-token", srv.URL)

	result := checkShelf(gh, "test-shelf", "testowner", "shelf-test")
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckShelf_NotFound(t *testing.T) {
	// Server with no repo handler returns 404 for /repos/...
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"Not Found"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	gh := ghclient.New("fake-token", srv.URL)
	result := checkShelf(gh, "missing-shelf", "testowner", "shelf-missing")
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s: %s", result.Status, result.Message)
	}
}

// ---------------------------------------------------------------------------
// runDoctor integration tests
// ---------------------------------------------------------------------------

func TestRunDoctor_AllPass(t *testing.T) {
	srv := newFakeGitHubServer(t, "octocat", "repo", "testowner", "shelf-test")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	cfgYAML := fmt.Sprintf(`github:
  owner: testowner
  token_env: DOCTOR_TEST_TOKEN
  api_base: %s
shelves:
  - name: test-shelf
    repo: shelf-test
`, srv.URL)
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELFCTL_CONFIG", cfgPath)
	t.Setenv("DOCTOR_TEST_TOKEN", "fake-token-for-test")

	// Set cache dir to temp to avoid touching real cache
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	allOK, err := runDoctor(&buf)
	if err != nil {
		t.Fatalf("runDoctor returned error: %v", err)
	}
	if !allOK {
		t.Errorf("expected allOK=true, got false\nOutput:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "All checks passed") {
		t.Errorf("expected 'All checks passed' in output, got:\n%s", buf.String())
	}
}

func TestRunDoctor_NoConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "nonexistent", "config.yml")
	t.Setenv("SHELFCTL_CONFIG", cfgPath)

	var buf bytes.Buffer
	allOK, err := runDoctor(&buf)
	if err != nil {
		t.Fatalf("runDoctor returned error: %v", err)
	}
	if allOK {
		t.Error("expected allOK=false for missing config")
	}
	if !strings.Contains(buf.String(), "not found") {
		t.Errorf("expected 'not found' in output, got:\n%s", buf.String())
	}
}

func TestRunDoctor_NoToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	cfgYAML := `github:
  owner: testowner
  token_env: DOCTOR_MISSING_TOKEN
`
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELFCTL_CONFIG", cfgPath)
	t.Setenv("DOCTOR_MISSING_TOKEN", "") // explicitly blank

	var buf bytes.Buffer
	allOK, err := runDoctor(&buf)
	if err != nil {
		t.Fatalf("runDoctor returned error: %v", err)
	}
	if allOK {
		t.Error("expected allOK=false when token is missing")
	}
	if !strings.Contains(buf.String(), "is not set") {
		t.Errorf("expected 'is not set' in output, got:\n%s", buf.String())
	}
}
