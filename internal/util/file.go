package util

// HumanBytes formats a byte count as a human-readable string (e.g., "1.5 MiB").
func HumanBytes(n int64) string {
	panic("scaffold stub")
}

// OpenFile opens a file with the system default application or a specified app.
func OpenFile(path, app string) error {
	panic("scaffold stub")
}

// IsPDF checks if the filename indicates a PDF file.
func IsPDF(filename string) bool {
	panic("scaffold stub")
}

// ComputeFileHash computes the SHA256 hash and byte size of a file.
func ComputeFileHash(path string) (string, int64, error) {
	panic("scaffold stub")
}
