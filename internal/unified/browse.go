package unified

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// BrowseModel wraps the tui.BrowserModel for unified mode
type BrowseModel struct {
	browser tui.BrowserModel
}

// browserDownloader implements tui.Downloader for browse view
// Implementation copied from internal/app/browse.go for feature parity
type browserDownloader struct {
	gh    *github.Client
	cache *cache.Manager
}

func (d *browserDownloader) Download(owner, repo, bookID, release, asset, sha256 string) (bool, error) {
	return d.DownloadWithProgress(owner, repo, bookID, release, asset, sha256, nil) == nil, nil
}

func (d *browserDownloader) DownloadWithProgress(owner, repo, bookID, release, asset, sha256 string, progressCh chan<- float64) error {
	// Get release
	rel, err := d.gh.GetReleaseByTag(owner, repo, release)
	if err != nil {
		return fmt.Errorf("release %q: %w", release, err)
	}

	// Find asset
	assetObj, err := d.gh.FindAsset(owner, repo, rel.ID, asset)
	if err != nil {
		return fmt.Errorf("finding asset: %w", err)
	}
	if assetObj == nil {
		return fmt.Errorf("asset %q not found", asset)
	}

	// Download
	rc, err := d.gh.DownloadAsset(owner, repo, assetObj.ID)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Wrap with progress tracking if channel provided
	var reader io.Reader = rc
	if progressCh != nil {
		reader = &progressReader{
			reader:     rc,
			total:      assetObj.Size,
			progressCh: progressCh,
		}
	}

	// Store in cache
	_, err = d.cache.Store(owner, repo, bookID, asset, reader, sha256)
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}

	return nil
}

func (d *browserDownloader) Uncache(owner, repo, bookID, asset string) error {
	return d.cache.Remove(owner, repo, bookID, asset)
}

func (d *browserDownloader) Sync(owner, repo, bookID, release, asset, catalogPath, catalogSHA256 string) (bool, error) {
	// Check if cached and modified
	if !d.cache.Exists(owner, repo, bookID, asset) {
		return false, fmt.Errorf("not cached")
	}

	if !d.cache.HasBeenModified(owner, repo, bookID, asset, catalogSHA256) {
		return false, nil // No changes
	}

	// Get cached file path, hash, and size
	cachedPath := d.cache.Path(owner, repo, bookID, asset)
	cachedSHA, cachedSize, err := computeFileHash(cachedPath)
	if err != nil {
		return false, fmt.Errorf("computing hash: %w", err)
	}

	// Get release
	rel, err := d.gh.GetReleaseByTag(owner, repo, release)
	if err != nil {
		return false, fmt.Errorf("release %q: %w", release, err)
	}

	// Find and delete old asset
	oldAsset, err := d.gh.FindAsset(owner, repo, rel.ID, asset)
	if err != nil {
		return false, fmt.Errorf("finding asset: %w", err)
	}
	if oldAsset != nil {
		if err := d.gh.DeleteAsset(owner, repo, oldAsset.ID); err != nil {
			return false, fmt.Errorf("deleting old asset: %w", err)
		}
	}

	// Upload modified file
	f, err := os.Open(cachedPath)
	if err != nil {
		return false, fmt.Errorf("opening cached file: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = d.gh.UploadAsset(owner, repo, rel.ID, asset, f, cachedSize, "application/octet-stream")
	if err != nil {
		return false, fmt.Errorf("uploading: %w", err)
	}

	// Update catalog with new SHA256
	mgr := catalog.NewManager(d.gh, owner, repo, catalogPath)
	books, err := mgr.Load()
	if err != nil {
		return false, fmt.Errorf("loading catalog: %w", err)
	}

	// Find and update the book
	bookToUpdate := catalog.ByID(books, bookID)
	if bookToUpdate != nil {
		bookToUpdate.Checksum.SHA256 = cachedSHA
		bookToUpdate.SizeBytes = cachedSize

		commitMsg := fmt.Sprintf("sync: update %s with local changes", bookID)
		if err := mgr.Save(books, commitMsg); err != nil {
			return false, fmt.Errorf("saving catalog: %w", err)
		}
	}

	return true, nil
}

func (d *browserDownloader) HasBeenModified(owner, repo, bookID, asset, catalogSHA256 string) bool {
	return d.cache.HasBeenModified(owner, repo, bookID, asset, catalogSHA256)
}

// progressReader wraps io.Reader to send progress updates
type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	progressCh chan<- float64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	if pr.progressCh != nil && pr.total > 0 {
		pct := float64(pr.read) / float64(pr.total)
		select {
		case pr.progressCh <- pct:
		default:
		}
	}

	return n, err
}

// computeFileHash computes SHA256 hash and size of a file
func computeFileHash(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

// NewBrowseModel creates a new browse model with the full browser
func NewBrowseModel(books []tui.BookItem, gh *github.Client, cacheMgr *cache.Manager) BrowseModel {
	// Create downloader
	dl := &browserDownloader{
		gh:    gh,
		cache: cacheMgr,
	}

	// Create the browser model in unified mode
	browser := tui.NewBrowserModel(books, dl, true) // true = unified mode

	return BrowseModel{
		browser: browser,
	}
}

func (m BrowseModel) Init() tea.Cmd {
	return m.browser.Init()
}

func (m BrowseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Forward to embedded browser
	browserModel, cmd := m.browser.Update(msg)
	m.browser = browserModel.(tui.BrowserModel)

	// Check if browser wants to quit
	if m.browser.IsQuitting() {
		// Check if there was an action requested
		action := m.browser.GetAction()
		selected := m.browser.GetSelected()

		switch action {
		case tui.ActionNone:
			// Simple quit - return to hub
			return m, func() tea.Msg {
				return NavigateMsg{Target: "hub"}
			}
		case tui.ActionShowDetails:
			// Details view - return to hub (could show details panel in future)
			return m, func() tea.Msg {
				return NavigateMsg{Target: "hub"}
			}
		case tui.ActionOpen:
			// Open book - suspend TUI, download if needed, open file
			return m, func() tea.Msg {
				return ActionRequestMsg{
					Action:   tui.ActionOpen,
					BookItem: selected,
					ReturnTo: "browse",
				}
			}
		case tui.ActionEdit:
			// Edit book - suspend TUI, run edit form, commit changes
			return m, func() tea.Msg {
				return ActionRequestMsg{
					Action:   tui.ActionEdit,
					BookItem: selected,
					ReturnTo: "browse",
				}
			}
		}
	}

	return m, cmd
}

func (m BrowseModel) View() string {
	return m.browser.View()
}
