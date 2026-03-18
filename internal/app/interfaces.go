package app

import (
	"io"

	github "github.com/blackwell-systems/shelfctl/internal/github"
)

// GitHubClient abstracts GitHub API operations for testability.
type GitHubClient interface {
	GetReleaseByTag(owner, repo, tag string) (*github.Release, error)
	EnsureRelease(owner, repo, tag string) (*github.Release, error)
	FindAsset(owner, repo string, releaseID int64, name string) (*github.Asset, error)
	ListReleaseAssets(owner, repo string, releaseID int64) ([]github.Asset, error)
	DownloadAsset(owner, repo string, assetID int64) (io.ReadCloser, error)
	UploadAsset(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*github.Asset, error)
	DeleteAsset(owner, repo string, assetID int64) error
	GetFileContent(owner, repo, path, ref string) ([]byte, string, error)
	CommitFile(owner, repo, filePath string, content []byte, message string) error
}

// Compile-time check that *github.Client satisfies GitHubClient.
var _ GitHubClient = (*github.Client)(nil)
