package github

import "errors"

// Common GitHub API errors.
var (
	// ErrNotFound is returned when a resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized — check your GitHub token")
	// ErrForbidden is returned when authorization fails.
	ErrForbidden = errors.New("forbidden — token may lack required scope (needs 'repo')")
	// ErrConflict is returned when a resource already exists.
	ErrConflict = errors.New("conflict — resource already exists")
)
