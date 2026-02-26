package unified

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current active view
type View string

const (
	ViewHub         View = "hub"
	ViewBrowse      View = "browse"
	ViewShelve      View = "shelve"
	ViewEdit        View = "edit"
	ViewMove        View = "move"
	ViewDelete      View = "delete"
	ViewCacheClear  View = "cache-clear"
	ViewCacheInfo   View = "cache-info"
	ViewCreateShelf View = "create-shelf"
	ViewDeleteShelf View = "delete-shelf"
	ViewImportShelf View = "import-shelf"
	ViewImportRepo  View = "import-repo"
	ViewShelves     View = "shelves"
)

// Model is the unified TUI orchestrator that manages view switching
type Model struct {
	currentView View
	width       int
	height      int

	// View models
	hub         HubModel
	browse      BrowseModel
	createShelf CreateShelfModel
	cacheClear  CacheClearModel
	cacheInfo   CacheInfoModel
	deleteBook  DeleteBookModel
	deleteShelf DeleteShelfModel
	editBook    EditBookModel
	shelve      ShelveModel
	moveBook    MoveBookModel
	importShelf ImportShelfModel
	importRepo  ImportRepoModel
	shelves     ShelvesModel

	// Context passed between views
	hubContext tui.HubContext

	// Dependencies needed for view initialization
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager

	// Pending action (used when TUI needs to exit to perform action)
	pendingAction  *ActionRequestMsg
	pendingCommand *CommandRequestMsg
	shouldRestart  bool
	restartAtView  View
}

// New creates a new unified model starting at the hub
func New(ctx tui.HubContext, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) Model {
	return NewAtView(ctx, gh, cfg, cacheMgr, ViewHub)
}

// NewAtView creates a new unified model starting at a specific view
func NewAtView(ctx tui.HubContext, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager, startView View) Model {
	m := Model{
		currentView: startView,
		hubContext:  ctx,
		gh:          gh,
		cfg:         cfg,
		cacheMgr:    cacheMgr,
	}

	// Initialize the starting view
	switch startView {
	case ViewHub:
		m.hub = NewHubModel(ctx)
	case ViewBrowse:
		books := m.collectBooks()
		m.browse = NewBrowseModel(books, gh, cacheMgr)
	case ViewCreateShelf:
		m.createShelf = NewCreateShelfModel(gh, cfg)
	// Add other views as they're implemented
	default:
		// Default to hub
		m.currentView = ViewHub
		m.hub = NewHubModel(ctx)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	switch m.currentView {
	case ViewHub:
		return m.hub.Init()
	case ViewBrowse:
		return m.browse.Init()
	case ViewCreateShelf:
		return m.createShelf.Init()
	default:
		return m.hub.Init()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to current view
		return m.updateCurrentView(msg)

	case NavigateMsg:
		return m.handleNavigation(msg)

	case QuitAppMsg:
		return m, tea.Quit

	case ActionRequestMsg:
		// Store action and exit TUI to perform it
		m.pendingAction = &msg
		m.shouldRestart = true
		// Map ReturnTo string to View
		switch msg.ReturnTo {
		case "hub":
			m.restartAtView = ViewHub
		case "browse":
			m.restartAtView = ViewBrowse
		default:
			m.restartAtView = ViewHub
		}
		return m, tea.Quit

	case CommandRequestMsg:
		// Store command request and exit TUI to perform it
		m.pendingCommand = &msg
		m.shouldRestart = true
		switch msg.ReturnTo {
		case "browse":
			m.restartAtView = ViewBrowse
		case "hub":
			m.restartAtView = ViewHub
		default:
			m.restartAtView = ViewHub
		}
		return m, tea.Quit

	default:
		// Forward to current view
		return m.updateCurrentView(msg)
	}
}

func (m Model) View() string {
	var content string
	switch m.currentView {
	case ViewHub:
		content = m.hub.View()
	case ViewBrowse:
		content = m.browse.View()
	case ViewCreateShelf:
		content = m.createShelf.View()
	case ViewCacheClear:
		content = m.cacheClear.View()
	case ViewCacheInfo:
		content = m.cacheInfo.View()
	case ViewDelete:
		content = m.deleteBook.View()
	case ViewDeleteShelf:
		content = m.deleteShelf.View()
	case ViewEdit:
		content = m.editBook.View()
	case ViewShelve:
		content = m.shelve.View()
	case ViewMove:
		content = m.moveBook.View()
	case ViewImportShelf:
		content = m.importShelf.View()
	case ViewImportRepo:
		content = m.importRepo.View()
	case ViewShelves:
		content = m.shelves.View()
	default:
		content = "Unknown view"
	}
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
	}
	return content
}

func (m Model) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewHub:
		var hubModel tea.Model
		hubModel, cmd = m.hub.Update(msg)
		m.hub = hubModel.(HubModel)
		// Check if command palette produced a navigation message
		// (handled inline to avoid one-frame flash of hub without palette)
		if m.hub.pendingNavMsg != nil {
			navMsg := m.hub.pendingNavMsg
			m.hub.pendingNavMsg = nil
			switch navMsg := navMsg.(type) {
			case NavigateMsg:
				return m.handleNavigation(navMsg)
			case QuitAppMsg:
				return m, tea.Quit
			}
		}
	case ViewBrowse:
		var browseModel tea.Model
		browseModel, cmd = m.browse.Update(msg)
		m.browse = browseModel.(BrowseModel)
	case ViewCreateShelf:
		var createShelfModel CreateShelfModel
		createShelfModel, cmd = m.createShelf.Update(msg)
		m.createShelf = createShelfModel
	case ViewCacheClear:
		var cacheClearModel CacheClearModel
		cacheClearModel, cmd = m.cacheClear.Update(msg)
		m.cacheClear = cacheClearModel
	case ViewCacheInfo:
		var cacheInfoModel CacheInfoModel
		cacheInfoModel, cmd = m.cacheInfo.Update(msg)
		m.cacheInfo = cacheInfoModel
	case ViewDelete:
		var deleteBookModel DeleteBookModel
		deleteBookModel, cmd = m.deleteBook.Update(msg)
		m.deleteBook = deleteBookModel
	case ViewDeleteShelf:
		var deleteShelfModel DeleteShelfModel
		deleteShelfModel, cmd = m.deleteShelf.Update(msg)
		m.deleteShelf = deleteShelfModel
	case ViewEdit:
		var editBookModel EditBookModel
		editBookModel, cmd = m.editBook.Update(msg)
		m.editBook = editBookModel
	case ViewShelve:
		var shelveModel ShelveModel
		shelveModel, cmd = m.shelve.Update(msg)
		m.shelve = shelveModel
	case ViewMove:
		var moveBookModel MoveBookModel
		moveBookModel, cmd = m.moveBook.Update(msg)
		m.moveBook = moveBookModel
	case ViewImportShelf:
		var importShelfModel ImportShelfModel
		importShelfModel, cmd = m.importShelf.Update(msg)
		m.importShelf = importShelfModel
	case ViewImportRepo:
		var importRepoModel ImportRepoModel
		importRepoModel, cmd = m.importRepo.Update(msg)
		m.importRepo = importRepoModel
	case ViewShelves:
		var shelvesModel ShelvesModel
		shelvesModel, cmd = m.shelves.Update(msg)
		m.shelves = shelvesModel
	}

	return m, cmd
}

// collectBooks gathers all books from all shelves for the browse view
// This replicates the logic from internal/app/browse.go
func (m Model) collectBooks() []tui.BookItem {
	var allItems []tui.BookItem

	for i := range m.cfg.Shelves {
		shelf := &m.cfg.Shelves[i]
		owner := shelf.EffectiveOwner(m.cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()
		releaseTag := shelf.EffectiveRelease(m.cfg.Defaults.Release)

		data, _, err := m.gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
		if err != nil {
			// Skip shelves with errors
			continue
		}

		books, err := catalog.Parse(data)
		if err != nil {
			// Skip shelves with parse errors
			continue
		}

		for _, b := range books {
			cached := m.cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)

			// Download catalog cover if specified and not already cached
			if b.Cover != "" && !m.cacheMgr.HasCatalogCover(shelf.Repo, b.ID) {
				if coverData, _, err := m.gh.GetFileContent(owner, shelf.Repo, b.Cover, ""); err == nil {
					_ = m.cacheMgr.StoreCatalogCover(shelf.Repo, b.ID, strings.NewReader(string(coverData)))
				}
			}

			// Get best available cover (catalog > extracted > none)
			coverPath := m.cacheMgr.GetCoverPath(shelf.Repo, b.ID)
			hasCover := coverPath != ""

			allItems = append(allItems, tui.BookItem{
				Book:        b,
				ShelfName:   shelf.Name,
				Cached:      cached,
				HasCover:    hasCover,
				CoverPath:   coverPath,
				Owner:       owner,
				Repo:        shelf.Repo,
				Release:     releaseTag,
				CatalogPath: catalogPath,
			})
		}
	}

	return allItems
}

func (m Model) handleNavigation(msg NavigateMsg) (tea.Model, tea.Cmd) {
	switch msg.Target {
	case "browse":
		m.currentView = ViewBrowse
		// Collect books from all shelves (same logic as browse.go)
		books := m.collectBooks()
		m.browse = NewBrowseModel(books, m.gh, m.cacheMgr)
		// Batch init command with window size message
		return m, tea.Batch(
			m.browse.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "shelve":
		// Shelve as unified view (no terminal drop)
		m.currentView = ViewShelve
		m.shelve = NewShelveModel(m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.shelve.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "edit-book":
		// Edit as unified view (no terminal drop)
		m.currentView = ViewEdit
		books := m.collectBooks()
		m.editBook = NewEditBookModel(books, m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.editBook.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "edit-book-single":
		// Edit a single book directly (from browse view), return to browse
		m.currentView = ViewEdit
		m.editBook = NewEditBookModelSingle(msg.BookItem, m.gh, m.cfg, m.cacheMgr, "browse")
		return m, tea.Batch(
			m.editBook.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "move":
		// Move as unified view (no terminal drop)
		m.currentView = ViewMove
		books := m.collectBooks()
		m.moveBook = NewMoveBookModel(books, m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.moveBook.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "delete-book":
		// Delete as unified view (no terminal drop)
		m.currentView = ViewDelete
		books := m.collectBooks()
		m.deleteBook = NewDeleteBookModel(books, m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.deleteBook.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "cache-clear":
		// Cache clear as unified view (no terminal drop)
		m.currentView = ViewCacheClear
		books := m.collectBooks()
		m.cacheClear = NewCacheClearModel(books, m.cacheMgr)
		return m, tea.Batch(
			m.cacheClear.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "hub":
		// Refresh hub context and return to hub
		m.currentView = ViewHub
		// Rebuild context to reflect any cache/catalog changes from browse
		m.hubContext = BuildContext(m.gh, m.cfg, m.cacheMgr)
		m.hub = NewHubModel(m.hubContext)
		// Batch init command with window size message
		return m, tea.Batch(
			m.hub.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "shelves":
		m.currentView = ViewShelves
		m.shelves = NewShelvesModel(m.gh, m.cfg)
		return m, tea.Batch(m.shelves.Init(), func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		})

	case "index":
		// Non-TUI command - just run command and return
		return m, func() tea.Msg {
			return CommandRequestMsg{
				Command:  "index",
				ReturnTo: "hub",
			}
		}

	case "cache-info":
		m.currentView = ViewCacheInfo
		books := m.collectBooks()
		m.cacheInfo = NewCacheInfoModel(books, m.cacheMgr)
		return m, tea.Batch(
			m.cacheInfo.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "shelve-url":
		m.currentView = ViewShelve
		m.shelve = NewShelveModelWithURL(m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.shelve.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "import-shelf":
		m.currentView = ViewImportShelf
		m.importShelf = NewImportShelfModel(m.gh, m.cfg)
		return m, tea.Batch(
			m.importShelf.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "import-repo":
		m.currentView = ViewImportRepo
		m.importRepo = NewImportRepoModel(m.gh, m.cfg)
		return m, tea.Batch(
			m.importRepo.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "create-shelf":
		// Create-shelf form as unified view
		m.currentView = ViewCreateShelf
		m.createShelf = NewCreateShelfModel(m.gh, m.cfg)
		return m, tea.Batch(
			m.createShelf.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "delete-shelf":
		m.currentView = ViewDeleteShelf
		m.deleteShelf = NewDeleteShelfModel(m.gh, m.cfg)
		return m, tea.Batch(
			m.deleteShelf.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	default:
		// Unknown target, stay on current view
		return m, nil
	}
}

// handleOpenBook downloads (if needed) and opens a book file
func (m Model) handleOpenBook(item *tui.BookItem) error {
	if item == nil {
		return fmt.Errorf("no book selected")
	}

	b := &item.Book

	// Download if not cached
	if !item.Cached {
		// Get release
		rel, err := m.gh.GetReleaseByTag(item.Owner, item.Repo, b.Source.Release)
		if err != nil {
			return fmt.Errorf("release %q: %w", b.Source.Release, err)
		}

		// Find asset
		asset, err := m.gh.FindAsset(item.Owner, item.Repo, rel.ID, b.Source.Asset)
		if err != nil {
			return fmt.Errorf("finding asset: %w", err)
		}
		if asset == nil {
			return fmt.Errorf("asset %q not found", b.Source.Asset)
		}

		// Download
		rc, err := m.gh.DownloadAsset(item.Owner, item.Repo, asset.ID)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		defer func() { _ = rc.Close() }()

		// Use progress bar with TUI
		progressCh := make(chan int64, 50)
		errCh := make(chan error, 1)

		// Show connecting message
		fmt.Printf("Connecting to GitHub...\n")

		// Start download in goroutine
		go func() {
			pr := tui.NewProgressReader(rc, asset.Size, progressCh)
			_, err := m.cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, pr, b.Checksum.SHA256)
			close(progressCh)
			errCh <- err
		}()

		// Show progress UI (TUI-based progress bar)
		label := fmt.Sprintf("Downloading %s (%s)", b.ID, humanBytes(asset.Size))
		if err := tui.ShowProgress(label, asset.Size, progressCh); err != nil {
			return err // User cancelled
		}

		// Get result
		if err := <-errCh; err != nil {
			return fmt.Errorf("cache: %w", err)
		}

		fmt.Println("✓ Cached")
	}

	// Open the file
	path := m.cacheMgr.Path(item.Owner, item.Repo, b.ID, b.Source.Asset)
	return openFile(path, "")
}

// handleEditBook opens the edit form and updates book metadata
func (m Model) handleEditBook(item *tui.BookItem) error {
	if item == nil {
		return fmt.Errorf("no book selected")
	}

	b := &item.Book
	shelf := m.cfg.ShelfByName(item.ShelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", item.ShelfName)
	}
	owner := shelf.EffectiveOwner(m.cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()

	// Show edit form
	defaults := tui.EditFormDefaults{
		BookID: b.ID,
		Title:  b.Title,
		Author: b.Author,
		Year:   b.Year,
		Tags:   b.Tags,
	}

	formData, err := tui.RunEditForm(defaults)
	if err != nil {
		return err
	}

	// Parse tags
	tags := []string{}
	if formData.Tags != "" {
		for _, t := range strings.Split(formData.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Load catalog
	data, _, err := m.gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing catalog: %w", err)
	}

	// Build updated book
	updatedBook := *b
	updatedBook.Title = formData.Title
	updatedBook.Author = formData.Author
	updatedBook.Year = formData.Year
	updatedBook.Tags = tags

	// Update book in catalog
	books = catalog.Append(books, updatedBook)

	// Commit catalog
	mgr := catalog.NewManager(m.gh, owner, shelf.Repo, catalogPath)
	commitMsg := fmt.Sprintf("edit: update %s metadata", b.ID)
	if err := mgr.Save(books, commitMsg); err != nil {
		return fmt.Errorf("committing catalog: %w", err)
	}

	fmt.Printf("\n✓ Book successfully updated: %s\n", b.ID)
	fmt.Println("\nPress Enter to return to browse...")
	fmt.Scanln() //nolint:errcheck

	return nil
}

// HasPendingAction returns true if there's a pending action to perform
func (m Model) HasPendingAction() bool {
	return m.pendingAction != nil
}

// GetPendingAction returns the pending action and clears it
func (m *Model) GetPendingAction() *ActionRequestMsg {
	action := m.pendingAction
	m.pendingAction = nil
	return action
}

// HasPendingCommand returns true if there's a pending command request
func (m Model) HasPendingCommand() bool {
	return m.pendingCommand != nil
}

// GetPendingCommand returns the pending command request and clears it
func (m *Model) GetPendingCommand() *CommandRequestMsg {
	cmd := m.pendingCommand
	m.pendingCommand = nil
	return cmd
}

// ShouldRestart returns true if the TUI should restart after an action
func (m Model) ShouldRestart() bool {
	return m.shouldRestart
}

// GetRestartView returns the view to restart at
func (m Model) GetRestartView() View {
	return m.restartAtView
}

// humanBytes formats bytes as human-readable size
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// openFile opens a file with the system default application
func openFile(path, app string) error {
	var cmdName string
	var args []string

	if app != "" {
		cmdName = app
		args = []string{path}
	} else {
		switch runtime.GOOS {
		case "darwin":
			cmdName = "open"
			args = []string{path}
		case "windows":
			cmdName = "cmd"
			args = []string{"/c", "start", "", path}
		default: // linux, freebsd, etc.
			cmdName = "xdg-open"
			args = []string{path}
		}
	}

	c := exec.Command(cmdName, args...)
	if err := c.Start(); err != nil {
		return fmt.Errorf("opening file with %q: %w", cmdName, err)
	}
	return nil
}

// PerformPendingAction executes a pending action outside the TUI
// This should be called after the TUI has exited
func PerformPendingAction(action *ActionRequestMsg, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) error {
	// Create a temporary model with dependencies
	m := Model{
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
	}

	switch action.Action {
	case tui.ActionOpen:
		return m.handleOpenBook(action.BookItem)
	case tui.ActionEdit:
		return m.handleEditBook(action.BookItem)
	default:
		return nil
	}
}

// BuildContext creates a fresh hub context by scanning all shelves and cache
// This is an expensive operation - only call when context may have changed
func BuildContext(gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) tui.HubContext {
	ctx := tui.HubContext{
		ShelfCount: len(cfg.Shelves),
	}

	// Collect shelf details for inline display
	var shelfDetails []tui.ShelfStatus
	for _, shelf := range cfg.Shelves {
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()
		release := shelf.EffectiveRelease(cfg.Defaults.Release)

		status := tui.ShelfStatus{
			Name:      shelf.Name,
			Repo:      shelf.Repo,
			Owner:     owner,
			BookCount: 0,
			Status:    "✓ Healthy",
		}

		// Check repo exists
		exists, err := gh.RepoExists(owner, shelf.Repo)
		if err != nil || !exists {
			status.Status = "✗ Repo not found"
			shelfDetails = append(shelfDetails, status)
			continue
		}

		// Load catalog and count books
		if data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, ""); err == nil {
			if books, err := catalog.Parse(data); err == nil {
				status.BookCount = len(books)
				ctx.BookCount += len(books)
			}
		} else {
			status.Status = "⚠ Catalog missing"
		}

		// Check release exists
		if _, err := gh.GetReleaseByTag(owner, shelf.Repo, release); err != nil {
			status.Status = "⚠ Release missing"
		}

		shelfDetails = append(shelfDetails, status)
	}

	ctx.ShelfDetails = shelfDetails

	// Calculate cache stats
	cachedCount := 0
	modifiedCount := 0
	var cacheSize int64
	var modifiedBooks []tui.ModifiedBook

	for _, shelf := range cfg.Shelves {
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()

		data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
		if err != nil {
			continue
		}
		books, err := catalog.Parse(data)
		if err != nil {
			continue
		}

		for i := range books {
			b := &books[i]
			if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				cachedCount++
				path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
				if info, err := os.Stat(path); err == nil {
					cacheSize += info.Size()
				}

				// Check if modified
				if cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
					modifiedCount++
					modifiedBooks = append(modifiedBooks, tui.ModifiedBook{
						ID:    b.ID,
						Title: b.Title,
					})
				}
			}
		}
	}

	ctx.CachedCount = cachedCount
	ctx.ModifiedCount = modifiedCount
	ctx.ModifiedBooks = modifiedBooks
	ctx.CacheSize = cacheSize
	if ctx.BookCount > 0 {
		// Get cache dir from any path
		ctx.CacheDir = cacheMgr.Path("", "", "", "")
		if ctx.CacheDir != "" {
			ctx.CacheDir = filepath.Dir(ctx.CacheDir)
		}
	}

	return ctx
}
