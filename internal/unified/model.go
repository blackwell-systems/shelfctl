package unified

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// View represents the current active view
type View string

const (
	ViewHub        View = "hub"
	ViewBrowse     View = "browse"
	ViewShelve     View = "shelve"
	ViewEdit       View = "edit"
	ViewMove       View = "move"
	ViewDelete     View = "delete"
	ViewCacheClear View = "cache-clear"
)

// Model is the unified TUI orchestrator that manages view switching
type Model struct {
	currentView View
	width       int
	height      int

	// View models
	hub    HubModel
	browse BrowseModel

	// Context passed between views
	hubContext tui.HubContext

	// Dependencies needed for view initialization
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager
}

// New creates a new unified model starting at the hub
func New(ctx tui.HubContext, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) Model {
	return Model{
		currentView: ViewHub,
		hubContext:  ctx,
		hub:         NewHubModel(ctx),
		gh:          gh,
		cfg:         cfg,
		cacheMgr:    cacheMgr,
	}
}

func (m Model) Init() tea.Cmd {
	return m.hub.Init()
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
		// Handle action that requires suspending TUI
		return m, m.handleAction(msg)

	case actionCompleteMsg:
		// Action completed, navigate back to specified view
		if msg.err != nil {
			// TODO: Show error message
			// For now, just return to the view
		}
		return m, func() tea.Msg {
			return NavigateMsg{Target: msg.returnTo}
		}

	default:
		// Forward to current view
		return m.updateCurrentView(msg)
	}
}

func (m Model) View() string {
	switch m.currentView {
	case ViewHub:
		return m.hub.View()
	case ViewBrowse:
		return m.browse.View()
	case ViewShelve:
		return "Shelve view (not yet implemented)"
	case ViewEdit:
		return "Edit view (not yet implemented)"
	case ViewMove:
		return "Move view (not yet implemented)"
	case ViewDelete:
		return "Delete view (not yet implemented)"
	case ViewCacheClear:
		return "Cache clear view (not yet implemented)"
	default:
		return "Unknown view"
	}
}

func (m Model) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewHub:
		var hubModel tea.Model
		hubModel, cmd = m.hub.Update(msg)
		m.hub = hubModel.(HubModel)
	case ViewBrowse:
		var browseModel tea.Model
		browseModel, cmd = m.browse.Update(msg)
		m.browse = browseModel.(BrowseModel)
	case ViewShelve:
		// TODO: forward to shelve model
	case ViewEdit:
		// TODO: forward to edit model
	case ViewMove:
		// TODO: forward to move model
	case ViewDelete:
		// TODO: forward to delete model
	case ViewCacheClear:
		// TODO: forward to cache clear model
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
		m.currentView = ViewShelve
		// TODO: initialize shelve model
		return m, nil

	case "edit-book":
		m.currentView = ViewEdit
		// TODO: initialize edit model
		return m, nil

	case "move":
		m.currentView = ViewMove
		// TODO: initialize move model
		return m, nil

	case "delete-book":
		m.currentView = ViewDelete
		// TODO: initialize delete model
		return m, nil

	case "cache-clear":
		m.currentView = ViewCacheClear
		// TODO: initialize cache clear model
		return m, nil

	case "hub":
		// Refresh hub context and return to hub
		m.currentView = ViewHub
		// TODO: refresh hub context from app state
		m.hub = NewHubModel(m.hubContext)
		// Batch init command with window size message
		return m, tea.Batch(
			m.hub.Init(),
			func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			},
		)

	case "shelves":
		// Non-TUI command - these still need to run outside unified mode
		// For now, just stay on hub
		// TODO: handle non-TUI commands
		return m, nil

	case "index":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "cache-info":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "shelve-url":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "import-repo":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "delete-shelf":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	default:
		// Unknown target, stay on current view
		return m, nil
	}
}

// actionCompleteMsg is sent when an action completes
type actionCompleteMsg struct {
	err      error
	returnTo string
}

// handleAction performs an action
// Note: For now, actions run in a goroutine. Downloads happen silently.
// TODO: Implement in-TUI progress indicators for better UX
func (m Model) handleAction(msg ActionRequestMsg) tea.Cmd {
	returnTarget := msg.ReturnTo
	if returnTarget == "" {
		returnTarget = "hub"
	}

	// For ActionOpen, if already cached, we can open immediately
	// If not cached, download will happen silently in background (not ideal UX)
	// For ActionEdit, we need terminal access, so this won't work well yet
	// TODO: Implement proper suspend/resume or in-TUI forms

	return func() tea.Msg {
		var err error
		switch msg.Action {
		case tui.ActionOpen:
			err = m.handleOpenBook(msg.BookItem)
		case tui.ActionEdit:
			err = m.handleEditBook(msg.BookItem)
		}

		return actionCompleteMsg{
			err:      err,
			returnTo: returnTarget,
		}
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
		fmt.Printf("Downloading %s...\n", b.ID)

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

		// Store in cache
		_, err = m.cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
		if err != nil {
			return fmt.Errorf("cache: %w", err)
		}

		fmt.Printf("Downloaded %s\n", b.ID)
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
	fmt.Scanln()

	return nil
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
