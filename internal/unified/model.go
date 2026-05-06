package unified

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
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
	ViewIndex       View = "index"
	ViewRenameShelf View = "rename-shelf"
	ViewSyncAll     View = "sync-modified"
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
	renameShelf RenameShelfModel
	syncAll     SyncAllModel
	editBook    EditBookModel
	shelve      ShelveModel
	moveBook    MoveBookModel
	importShelf ImportShelfModel
	importRepo  ImportRepoModel
	shelves     ShelvesModel
	index       IndexModel

	// Context passed between views
	hubContext tui.HubContext

	// Dependencies needed for view initialization
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager

	// In-session catalog cache — avoids re-fetching unchanged catalogs across views.
	// Keyed by "owner/repo". Populated by the async context load and getCatalog().
	// Automatically reset on TUI restart (new Model), so stale data after writes is
	// not a concern.
	catalogCache map[string][]catalog.Book

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
		currentView:  startView,
		hubContext:   ctx,
		gh:           gh,
		cfg:          cfg,
		cacheMgr:     cacheMgr,
		catalogCache: make(map[string][]catalog.Book),
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
		return tea.Batch(m.hub.Init(), m.loadHubContextAsync(), hubModifiedRefreshTick())
	case ViewBrowse:
		return m.browse.Init()
	case ViewCreateShelf:
		return m.createShelf.Init()
	default:
		return tea.Batch(m.hub.Init(), m.loadHubContextAsync(), hubModifiedRefreshTick())
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

	case hubContextLoadedMsg:
		// Preserve transient auto-sync display fields that loadHubContextAsync
		// does not set — they are owned by the hub orchestrator, not the loader.
		msg.ctx.AutoSyncInProgress = m.hubContext.AutoSyncInProgress
		msg.ctx.LastAutoSyncAt = m.hubContext.LastAutoSyncAt
		msg.ctx.LastAutoSyncCount = m.hubContext.LastAutoSyncCount
		msg.ctx.LastAutoSyncErrors = m.hubContext.LastAutoSyncErrors
		msg.ctx.LastAutoSyncErrorMsg = m.hubContext.LastAutoSyncErrorMsg
		m.hubContext = msg.ctx
		m.hub.UpdateContext(m.hubContext)
		// Seed catalog cache so browse/index navigation skips re-fetching
		for k, v := range msg.catalogs {
			m.catalogCache[k] = v
		}
		// Immediately check for auto-sync candidates if enabled
		if m.cfg.Sync.AutoSync && msg.ctx.ModifiedCount > 0 {
			return m, m.refreshModifiedStatusCmd()
		}
		return m, nil

	case modifiedRefreshTickMsg:
		// Only continue periodic scanning while on the hub and catalog data is available.
		if m.currentView == ViewHub && len(m.catalogCache) > 0 {
			return m, tea.Batch(m.refreshModifiedStatusCmd(), hubModifiedRefreshTick())
		}
		return m, nil

	case modifiedStatusRefreshedMsg:
		if m.currentView == ViewHub {
			m.hubContext.ModifiedCount = msg.count
			m.hubContext.ModifiedBooks = msg.books
			m.hub.UpdateContext(m.hubContext)
			if len(msg.pendingAutoSync) > 0 && !m.syncAll.autoMode {
				m.syncAll = NewSyncAllModelAuto(m.gh, m.cfg, m.cacheMgr, msg.pendingAutoSync)
				m.hubContext.AutoSyncInProgress = true
				m.hub.UpdateContext(m.hubContext)
				return m, m.syncAll.Init()
			}
		}
		return m, nil

	case autoSyncDoneMsg:
		now := time.Now()
		m.hubContext.LastAutoSyncAt = &now
		m.hubContext.LastAutoSyncCount = msg.synced
		m.hubContext.LastAutoSyncErrors = msg.errors
		if len(msg.errorMsgs) > 0 {
			m.hubContext.LastAutoSyncErrorMsg = msg.errorMsgs[0]
		}
		m.hubContext.AutoSyncInProgress = false
		if m.currentView == ViewHub {
			m.hub.UpdateContext(m.hubContext)
		}
		// When books were successfully synced, reload catalog data from GitHub so
		// the in-memory cache reflects the new SHAs. Without this, HasBeenModified
		// would still see the old SHA and re-queue the same book, causing a
		// "nothing to commit" error on the next catalog save.
		// Failed-only runs skip this — failed books retry on the next 20-second tick.
		if msg.synced > 0 {
			return m, m.loadHubContextAsync()
		}
		return m, nil

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
		// Forward upload/progress messages to headless auto-sync model when active
		if m.syncAll.autoMode {
			switch msg.(type) {
			case syncUploadReadyMsg, syncUploadTickMsg, syncUploadDoneMsg,
				syncProgressMsg, spinner.TickMsg, progress.FrameMsg:
				var cmd tea.Cmd
				m.syncAll, cmd = m.syncAll.Update(msg)
				return m, cmd
			}
		}
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
	case ViewRenameShelf:
		content = m.renameShelf.View()
	case ViewSyncAll:
		content = m.syncAll.View()
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
	case ViewIndex:
		content = m.index.View()
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
	case ViewRenameShelf:
		var renameShelfModel RenameShelfModel
		renameShelfModel, cmd = m.renameShelf.Update(msg)
		m.renameShelf = renameShelfModel
	case ViewSyncAll:
		var syncAllModel SyncAllModel
		syncAllModel, cmd = m.syncAll.Update(msg)
		m.syncAll = syncAllModel
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
	case ViewIndex:
		var indexModel IndexModel
		indexModel, cmd = m.index.Update(msg)
		m.index = indexModel
	}

	return m, cmd
}

// getCatalog returns the parsed catalog for a shelf, using the in-session cache.
// On a cache miss it fetches from GitHub and caches the result.
// Since catalogCache is a map (reference type), writes are visible even through
// value-receiver copies as long as the map was initialized in NewAtView.
func (m Model) getCatalog(owner, repo, catalogPath string) ([]catalog.Book, error) {
	key := owner + "/" + repo
	if books, ok := m.catalogCache[key]; ok {
		return books, nil
	}
	data, _, err := m.gh.GetFileContent(owner, repo, catalogPath, "")
	if err != nil {
		return nil, err
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return nil, err
	}
	m.catalogCache[key] = books
	return books, nil
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

		books, err := m.getCatalog(owner, shelf.Repo, catalogPath)
		if err != nil {
			continue
		}

		for _, b := range books {
			cached := m.cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)

			// Download catalog cover if specified and not already cached
			if b.Cover != "" && !m.cacheMgr.HasCatalogCover(owner, shelf.Repo, b.ID) {
				if coverData, _, err := m.gh.GetFileContent(owner, shelf.Repo, b.Cover, ""); err == nil {
					_ = m.cacheMgr.StoreCatalogCover(owner, shelf.Repo, b.ID, strings.NewReader(string(coverData)))
				}
			}

			allItems = append(allItems, tui.BookItem{
				Book:        b,
				ShelfName:   shelf.Name,
				Cached:      cached,
				Owner:       owner,
				Repo:        shelf.Repo,
				Release:     releaseTag,
				CatalogPath: catalogPath,
			})
		}
	}

	return allItems
}

// collectIndexBooks gathers books in the cache.IndexBook format needed by IndexModel.
func (m Model) collectIndexBooks() []cache.IndexBook {
	var result []cache.IndexBook

	for i := range m.cfg.Shelves {
		shelf := &m.cfg.Shelves[i]
		owner := shelf.EffectiveOwner(m.cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()

		books, err := m.getCatalog(owner, shelf.Repo, catalogPath)
		if err != nil {
			continue
		}

		for _, b := range books {
			isCached := m.cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)
			var filePath string
			if isCached {
				filePath = m.cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			}
			coverPath := m.cacheMgr.GetCoverPath(owner, shelf.Repo, b.ID)
			result = append(result, cache.IndexBook{
				Book:      b,
				ShelfName: shelf.Name,
				Repo:      shelf.Repo,
				FilePath:  filePath,
				CoverPath: coverPath,
				HasCover:  coverPath != "",
				IsCached:  isCached,
			})
		}
	}

	return result
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
		// Return to hub with fast local context; async load will refresh counts.
		// Reload config from disk to pick up any changes written by operations
		// (e.g., create shelf, delete shelf) that modify the config file.
		if newCfg, err := config.Load(); err == nil {
			m.cfg = newCfg
			m.catalogCache = make(map[string][]catalog.Book) // stale after config change
		}
		m.currentView = ViewHub
		fast := BuildContextFast(m.cfg)
		fast.Version = m.hubContext.Version // preserve — fast context doesn't set it
		m.hubContext = fast
		m.hub = NewHubModel(m.hubContext)
		return m, tea.Batch(
			m.hub.Init(),
			m.loadHubContextAsync(),
			hubModifiedRefreshTick(),
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
		m.currentView = ViewIndex
		books := m.collectIndexBooks()
		m.index = NewIndexModel(books, m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.index.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

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

	case "rename-shelf":
		m.currentView = ViewRenameShelf
		m.renameShelf = NewRenameShelfModel(m.gh, m.cfg, m.cacheMgr.BaseDir())
		return m, tea.Batch(
			m.renameShelf.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "sync-modified":
		m.currentView = ViewSyncAll
		m.syncAll = NewSyncAllModel(m.gh, m.cfg, m.cacheMgr)
		return m, tea.Batch(
			m.syncAll.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "auto-sync":
		m.cfg.Sync.AutoSync = !m.cfg.Sync.AutoSync
		_ = config.Save(m.cfg)
		m.hubContext.AutoSyncEnabled = m.cfg.Sync.AutoSync
		m.hub.UpdateContext(m.hubContext)
		return m, nil

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
		label := fmt.Sprintf("Downloading %s (%s)", b.ID, util.HumanBytes(asset.Size))
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
	return util.OpenFile(path, "")
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

// hubContextLoadedMsg carries the result of an async context load.
// catalogs is the fetched catalog data, used to seed the in-session cache.
type hubContextLoadedMsg struct {
	ctx      tui.HubContext
	catalogs map[string][]catalog.Book
}

// modifiedRefreshTickMsg drives periodic local-only rescans of modified books.
type modifiedRefreshTickMsg time.Time

// modifiedStatusRefreshedMsg carries updated modified-book counts from a local-only rescan.
type modifiedStatusRefreshedMsg struct {
	count           int
	books           []tui.ModifiedBook
	pendingAutoSync []syncEntry // non-nil when auto-sync should be triggered
}

// autoSyncDoneMsg is emitted when a background auto-sync run finishes.
type autoSyncDoneMsg struct {
	synced    int
	errors    int
	errorMsgs []string
}

// hubModifiedRefreshTick schedules the next periodic modified-status scan.
func hubModifiedRefreshTick() tea.Cmd {
	return tea.Tick(20*time.Second, func(t time.Time) tea.Msg {
		return modifiedRefreshTickMsg(t)
	})
}

// withinDebounce returns true if the file at path was last modified within
// debounceMins minutes of now. Used to skip files still being written to.
// Returns false (do not skip) if debounceMins <= 0 or the file cannot be stat'd.
func withinDebounce(path string, debounceMins int) bool {
	if debounceMins <= 0 {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < time.Duration(debounceMins)*time.Minute
}

// refreshModifiedStatusCmd rescans local cache files against catalogCache checksums.
// No network calls — uses already-cached catalog data from the initial async load.
// When cfg.Sync.AutoSync is true, also collects debounce-passing entries for auto-sync.
func (m Model) refreshModifiedStatusCmd() tea.Cmd {
	cacheMgr := m.cacheMgr
	cfg := m.cfg
	autoSync := cfg.Sync.AutoSync
	debounceMins := cfg.Sync.DebounceMinutes
	catalogs := make(map[string][]catalog.Book, len(m.catalogCache))
	for k, v := range m.catalogCache {
		catalogs[k] = v
	}
	return func() tea.Msg {
		var count int
		var books []tui.ModifiedBook
		var pending []syncEntry
		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			key := owner + "/" + shelf.Repo
			shelfBooks, ok := catalogs[key]
			if !ok {
				continue
			}
			releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)
			catalogPath := shelf.EffectiveCatalogPath()
			shelfCopy := shelf
			for i := range shelfBooks {
				b := &shelfBooks[i]
				if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
					continue
				}
				if !cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
					continue
				}
				count++
				books = append(books, tui.ModifiedBook{ID: b.ID, Title: b.Title})
				if autoSync {
					cachedPath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
					if withinDebounce(cachedPath, debounceMins) {
						continue // too recent; wait for next scan
					}
					newSHA, size, err := util.ComputeFileHash(cachedPath)
					if err != nil {
						continue
					}
					pending = append(pending, syncEntry{
						shelf:       shelfCopy,
						book:        *b,
						cachedPath:  cachedPath,
						size:        size,
						newSHA:      newSHA,
						releaseTag:  releaseTag,
						catalogPath: catalogPath,
						owner:       owner,
					})
				}
			}
		}
		return modifiedStatusRefreshedMsg{count: count, books: books, pendingAutoSync: pending}
	}
}

// BuildContextFast returns a minimal hub context derived from local config only —
// no network calls. Use this to start the TUI immediately, then fire
// loadHubContextAsync to populate network-dependent fields.
func BuildContextFast(cfg *config.Config) tui.HubContext {
	return tui.HubContext{
		ShelfCount:      len(cfg.Shelves),
		AutoSyncEnabled: cfg.Sync.AutoSync,
	}
}

// loadHubContextAsync returns a Tea command that builds the full hub context
// in the background. It fetches each catalog once, reusing the data for both
// shelf details and cache stats (no duplicate requests). Health checks
// (RepoExists, GetReleaseByTag) are skipped here — they appear in ShelvesModel.
func (m Model) loadHubContextAsync() tea.Cmd {
	gh := m.gh
	cfg := m.cfg
	cacheMgr := m.cacheMgr
	version := m.hubContext.Version // capture: async closure has no access to appVersion
	return func() tea.Msg {
		ctx := tui.HubContext{
			ShelfCount:      len(cfg.Shelves),
			AutoSyncEnabled: cfg.Sync.AutoSync,
		}
		catalogs := make(map[string][]catalog.Book)

		// Single pass: fetch each catalog once
		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			catalogPath := shelf.EffectiveCatalogPath()
			key := owner + "/" + shelf.Repo

			data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil {
				continue
			}
			books, err := catalog.Parse(data)
			if err != nil {
				continue
			}
			catalogs[key] = books
			ctx.BookCount += len(books)
		}

		// Build shelf details from cached catalog data (no extra network calls)
		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			key := owner + "/" + shelf.Repo
			books := catalogs[key]
			ctx.ShelfDetails = append(ctx.ShelfDetails, tui.ShelfStatus{
				Name:      shelf.Name,
				Repo:      shelf.Repo,
				Owner:     owner,
				BookCount: len(books),
				Status:    "✓ Healthy",
			})
		}

		// Calculate cache stats using local filesystem only
		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			key := owner + "/" + shelf.Repo
			books := catalogs[key]
			for i := range books {
				b := &books[i]
				if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
					ctx.CachedCount++
					path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
					if info, err := os.Stat(path); err == nil {
						ctx.CacheSize += info.Size()
					}
					if cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
						ctx.ModifiedCount++
						ctx.ModifiedBooks = append(ctx.ModifiedBooks, tui.ModifiedBook{
							ID:    b.ID,
							Title: b.Title,
						})
					}
				}
			}
		}

		if ctx.BookCount > 0 {
			ctx.CacheDir = cacheMgr.BaseDir()
		}

		ctx.Version = version
		return hubContextLoadedMsg{ctx: ctx, catalogs: catalogs}
	}
}
