package app

import (
	"io"
	"time"
)

// CheckResult holds the outcome of a single doctor check.
type CheckResult struct {
	Label   string
	Status  string // "pass", "fail", "warn", "skip"
	Message string
	Detail  string // optional remediation hint
}

// runDoctor executes all health checks and writes results to w.
// Returns true if all non-skipped checks passed (exit 0).
func runDoctor(w io.Writer) (bool, error) {
	_ = time.Now()
	return false, nil
}
