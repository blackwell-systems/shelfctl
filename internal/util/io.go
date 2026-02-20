package util

import (
	"io"
	"os"
	"path/filepath"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func CopyFile(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
