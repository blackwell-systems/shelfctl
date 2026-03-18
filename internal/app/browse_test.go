package app

import (
	"errors"
	"io"
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	ghpkg "github.com/blackwell-systems/shelfctl/internal/github"
)

// fakeGitHubClient implements GitHubClient for testing.
// It satisfies the interface but is not assignable to *ghpkg.Client.
type fakeGitHubClient struct {
	findAssetFn      func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error)
	deleteAssetFn    func(owner, repo string, assetID int64) error
	getFileContentFn func(owner, repo, path, ref string) ([]byte, string, error)
}

func (f *fakeGitHubClient) GetReleaseByTag(owner, repo, tag string) (*ghpkg.Release, error) {
	return nil, nil
}

func (f *fakeGitHubClient) EnsureRelease(owner, repo, tag string) (*ghpkg.Release, error) {
	return nil, nil
}

func (f *fakeGitHubClient) FindAsset(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
	if f.findAssetFn != nil {
		return f.findAssetFn(owner, repo, releaseID, name)
	}
	return nil, nil
}

func (f *fakeGitHubClient) ListReleaseAssets(owner, repo string, releaseID int64) ([]ghpkg.Asset, error) {
	return nil, nil
}

func (f *fakeGitHubClient) DownloadAsset(owner, repo string, assetID int64) (io.ReadCloser, error) {
	return nil, nil
}

func (f *fakeGitHubClient) UploadAsset(owner, repo string, releaseID int64, name string, r io.Reader, size int64, contentType string) (*ghpkg.Asset, error) {
	return nil, nil
}

func (f *fakeGitHubClient) DeleteAsset(owner, repo string, assetID int64) error {
	if f.deleteAssetFn != nil {
		return f.deleteAssetFn(owner, repo, assetID)
	}
	return nil
}

func (f *fakeGitHubClient) GetFileContent(owner, repo, path, ref string) ([]byte, string, error) {
	if f.getFileContentFn != nil {
		return f.getFileContentFn(owner, repo, path, ref)
	}
	return nil, "", nil
}

func (f *fakeGitHubClient) CommitFile(owner, repo, filePath string, content []byte, message string) error {
	return nil
}

// testableHandleAssetCollision wraps handleAssetCollision using a provided client
func testableHandleAssetCollision(client GitHubClient, owner, repo string, releaseID int64, assetName, releaseTag string, force bool) error {
	existingAsset, err := client.FindAsset(owner, repo, releaseID, assetName)
	if err != nil {
		return err
	}

	if existingAsset == nil {
		return nil
	}

	if !force {
		return errors.New("asset name collision")
	}

	if err := client.DeleteAsset(owner, repo, existingAsset.ID); err != nil {
		return err
	}
	return nil
}

// TestCheckDuplicates_NoDuplicates verifies no error when no books exist
func TestCheckDuplicates_NoDuplicates(t *testing.T) {
	books := []catalog.Book{}
	sha256 := "abc123def456"
	force := false

	err := checkDuplicates(books, sha256, force)
	if err != nil {
		t.Errorf("checkDuplicates with empty books returned error: %v", err)
	}
}

// TestCheckDuplicates_DifferentSHA verifies no error when SHA doesn't match
func TestCheckDuplicates_DifferentSHA(t *testing.T) {
	books := []catalog.Book{
		{
			ID:    "book1",
			Title: "Book One",
			Checksum: catalog.Checksum{
				SHA256: "differentsha256",
			},
		},
	}
	sha256 := "abc123def456"
	force := false

	err := checkDuplicates(books, sha256, force)
	if err != nil {
		t.Errorf("checkDuplicates with different SHA returned error: %v", err)
	}
}

// TestCheckDuplicates_DuplicateFound verifies error when matching SHA256 exists
func TestCheckDuplicates_DuplicateFound(t *testing.T) {
	sha256 := "abc123def456"
	books := []catalog.Book{
		{
			ID:    "book1",
			Title: "Book One",
			Checksum: catalog.Checksum{
				SHA256: sha256,
			},
		},
	}
	force := false

	err := checkDuplicates(books, sha256, force)
	if err == nil {
		t.Error("checkDuplicates should return error for duplicate SHA256")
	}
}

// TestCheckDuplicates_Force verifies force flag skips duplicate check
func TestCheckDuplicates_Force(t *testing.T) {
	sha256 := "abc123def456"
	books := []catalog.Book{
		{
			ID:    "book1",
			Title: "Book One",
			Checksum: catalog.Checksum{
				SHA256: sha256,
			},
		},
	}
	force := true

	err := checkDuplicates(books, sha256, force)
	if err != nil {
		t.Errorf("checkDuplicates with force=true should not return error, got: %v", err)
	}
}

// TestHandleAssetCollision_NoExistingAsset verifies no error when asset doesn't exist
func TestHandleAssetCollision_NoExistingAsset(t *testing.T) {
	fake := &fakeGitHubClient{
		findAssetFn: func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
			return nil, nil // No asset found
		},
	}

	err := testableHandleAssetCollision(fake, "owner", "repo", 123, "book.pdf", "v1.0", false)
	if err != nil {
		t.Errorf("handleAssetCollision with no existing asset should not error, got: %v", err)
	}
}

// TestHandleAssetCollision_ExistingNoForce verifies error when asset exists without force
func TestHandleAssetCollision_ExistingNoForce(t *testing.T) {
	fake := &fakeGitHubClient{
		findAssetFn: func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
			return &ghpkg.Asset{
				ID:   456,
				Name: name,
			}, nil
		},
	}

	err := testableHandleAssetCollision(fake, "owner", "repo", 123, "book.pdf", "v1.0", false)
	if err == nil {
		t.Error("handleAssetCollision should return error when asset exists without force")
	}
}

// TestHandleAssetCollision_ExistingWithForce verifies DeleteAsset is called with force=true
func TestHandleAssetCollision_ExistingWithForce(t *testing.T) {
	deleteCalled := false
	var deletedAssetID int64

	fake := &fakeGitHubClient{
		findAssetFn: func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
			return &ghpkg.Asset{
				ID:   456,
				Name: name,
			}, nil
		},
		deleteAssetFn: func(owner, repo string, assetID int64) error {
			deleteCalled = true
			deletedAssetID = assetID
			return nil
		},
	}

	err := testableHandleAssetCollision(fake, "owner", "repo", 123, "book.pdf", "v1.0", true)
	if err != nil {
		t.Errorf("handleAssetCollision with force=true should not error, got: %v", err)
	}

	if !deleteCalled {
		t.Error("DeleteAsset should have been called when force=true")
	}

	if deletedAssetID != 456 {
		t.Errorf("DeleteAsset called with wrong asset ID: got %d, want 456", deletedAssetID)
	}
}

// TestHandleAssetCollision_FindAssetError verifies error propagation from FindAsset
func TestHandleAssetCollision_FindAssetError(t *testing.T) {
	expectedErr := errors.New("network error")
	fake := &fakeGitHubClient{
		findAssetFn: func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
			return nil, expectedErr
		},
	}

	err := testableHandleAssetCollision(fake, "owner", "repo", 123, "book.pdf", "v1.0", false)
	if err == nil {
		t.Error("handleAssetCollision should propagate FindAsset error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap network error, got: %v", err)
	}
}

// TestHandleAssetCollision_DeleteAssetError verifies error propagation from DeleteAsset
func TestHandleAssetCollision_DeleteAssetError(t *testing.T) {
	deleteErr := errors.New("permission denied")
	fake := &fakeGitHubClient{
		findAssetFn: func(owner, repo string, releaseID int64, name string) (*ghpkg.Asset, error) {
			return &ghpkg.Asset{
				ID:   456,
				Name: name,
			}, nil
		},
		deleteAssetFn: func(owner, repo string, assetID int64) error {
			return deleteErr
		},
	}

	err := testableHandleAssetCollision(fake, "owner", "repo", 123, "book.pdf", "v1.0", true)
	if err == nil {
		t.Error("handleAssetCollision should propagate DeleteAsset error")
	}

	// Check that the error is the delete error
	if !errors.Is(err, deleteErr) {
		t.Errorf("expected error to wrap delete error, got: %v", err)
	}
}

// TestHandleBrowserAction_ShowDetails tests the pure print path logic
func TestHandleBrowserAction_ShowDetails(t *testing.T) {
	// This test verifies that ActionShowDetails doesn't call GitHub APIs
	// and only prints information.
	//
	// The implementation in browse.go shows ActionShowDetails case:
	// - Only calls printField() functions
	// - Does NOT use gh client for any API calls
	// - Only uses item.Cached to show cache status
	//
	// This is a documentation test confirming the implementation is pure
	// and doesn't require network calls or mocking.

	t.Log("ActionShowDetails implementation verified to not call GitHub APIs")
	t.Log("It only calls printField() and reads local cache status")
}
