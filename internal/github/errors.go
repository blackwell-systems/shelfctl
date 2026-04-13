package github

import "errors"

// Common GitHub API errors.
var (
	// ErrNotFound is returned when a resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("authentication failed: check that SHELFCTL_GITHUB_TOKEN is set and valid")
	// ErrForbidden is returned when authorization fails.
	ErrForbidden = errors.New("permission denied: ensure your token has 'repo' scope for private repos, or 'public_repo' for public")
	// ErrConflict is returned when a resource already exists.
	ErrConflict = errors.New("conflict: resource already exists")
	// ErrRateLimited is returned when the GitHub API rate limit is exceeded.
	ErrRateLimited = errors.New("GitHub API rate limit exceeded: wait and try again, or check usage at api.github.com/rate_limit")
	// ErrServerError is returned for 5xx responses from GitHub.
	ErrServerError = errors.New("GitHub API error: try again later")
)
