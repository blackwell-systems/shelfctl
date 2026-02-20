package github

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized — check your GitHub token")
	ErrForbidden    = errors.New("forbidden — token may lack required scope (needs 'repo')")
	ErrConflict     = errors.New("conflict — resource already exists")
)
