package migrate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/github"
)

// FileEntry is one file discovered in a source repo.
type FileEntry struct {
	Path string
	SHA  string
	Size int
}

// ScanRepo lists all files in the source repo matching the given extensions.
// Pass nil or empty exts to match everything.
func ScanRepo(gh *github.Client, owner, repo, ref string, exts []string) ([]FileEntry, error) {
	var results []FileEntry
	if err := scanDir(gh, owner, repo, ref, "", exts, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func scanDir(gh *github.Client, owner, repo, ref, dirPath string, exts []string, out *[]FileEntry) error {
	entries, err := gh.ListDirContents(owner, repo, dirPath, ref)
	if err != nil {
		return fmt.Errorf("list %s: %w", dirPath, err)
	}

	for _, e := range entries {
		switch e.Type {
		case "file":
			if matchExt(e.Path, exts) {
				*out = append(*out, FileEntry{Path: e.Path, SHA: e.SHA, Size: e.Size})
			}
		case "dir":
			if err := scanDir(gh, owner, repo, ref, e.Path, exts, out); err != nil {
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
