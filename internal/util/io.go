package util

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome expands a leading ~/ to the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // fallback to unexpanded path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
