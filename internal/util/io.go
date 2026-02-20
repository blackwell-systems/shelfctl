package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDir creates the directory at path, including parent directories.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

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

// CopyFile copies a file from src to dst, creating parent directories as needed.
func CopyFile(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
