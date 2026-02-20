package cache

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/util"
)

// VerifyFile checks the sha256 of the file at path against expected.
// Returns nil if they match or expected is empty (skip check).
func VerifyFile(path, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return nil
	}
	got, err := util.SHA256File(path)
	if err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}
	if got != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, got)
	}
	return nil
}
