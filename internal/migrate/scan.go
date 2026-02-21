package migrate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// FileEntry is one file discovered in a source repo.
type FileEntry struct {
	Path string
	SHA  string
	Size int
}

// dirContent is the Contents API response for a directory.
type dirContent []struct {
	Type string `json:"type"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Size int    `json:"size"`
}

// ScanRepo lists all files in the source repo matching the given extensions.
// Pass nil or empty exts to match everything.
func ScanRepo(token, apiBase, owner, repo, ref string, exts []string) ([]FileEntry, error) {
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	client := &http.Client{}
	var results []FileEntry
	if err := scanDir(client, token, apiBase, owner, repo, ref, "", exts, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func scanDir(client *http.Client, token, apiBase, owner, repo, ref, dirPath string, exts []string, out *[]FileEntry) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", apiBase, owner, repo, dirPath, ref)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list %s: status %d", dirPath, resp.StatusCode)
	}

	var entries dirContent
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return err
	}

	for _, e := range entries {
		switch e.Type {
		case "file":
			if matchExt(e.Path, exts) {
				*out = append(*out, FileEntry{Path: e.Path, SHA: e.SHA, Size: e.Size})
			}
		case "dir":
			if err := scanDir(client, token, apiBase, owner, repo, ref, e.Path, exts, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchExt(path string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	for _, e := range exts {
		if strings.EqualFold(ext, strings.TrimPrefix(e, ".")) {
			return true
		}
	}
	return false
}
