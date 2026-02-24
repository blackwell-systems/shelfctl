package tui

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// keyMap defines keyboard shortcuts
type keyMap struct {
	quit         key.Binding
	enter        key.Binding
	open         key.Binding
	get          key.Binding
	uncache      key.Binding
	sync         key.Binding
	edit         key.Binding
	filter       key.Binding
	togglePanel  key.Binding
	toggleSelect key.Binding
	clearSelect  key.Binding
}

var keys = keyMap{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "action"),
	),
	open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	get: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "download"),
	),
	uncache: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "remove cache"),
	),
	sync: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sync"),
	),
	edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	togglePanel: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle details"),
	),
	toggleSelect: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	clearSelect: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear selection"),
	),
}

// BrowserAction represents an action requested from the browser
// BrowserAction represents an action that can be performed in the browser.
type BrowserAction string

// Browser action types.
const (
	ActionNone        BrowserAction = ""
	ActionShowDetails BrowserAction = "details"
	ActionOpen        BrowserAction = "open"
	ActionDownload    BrowserAction = "download"
	ActionEdit        BrowserAction = "edit"
)

// BrowserResult holds the result of a browser session
type BrowserResult struct {
	Action    BrowserAction
	BookItem  *BookItem  // Single book (for open, edit, details)
	BookItems []BookItem // Multiple books (for download)
}

// downloadMsg contains download progress updates
type downloadMsg struct {
	bookID   string
	progress float64 // 0.0 to 1.0
	done     bool
	err      error
}

// Downloader interface abstracts GitHub/cache operations
type Downloader interface {
	Download(owner, repo, bookID, release, asset, sha256 string) (downloaded bool, err error)
	DownloadWithProgress(owner, repo, bookID, release, asset, sha256 string, progressCh chan<- float64) error
	Uncache(owner, repo, bookID, asset string) error
	Sync(owner, repo, bookID, release, asset, catalogPath, catalogSHA256 string) (synced bool, err error)
	HasBeenModified(owner, repo, bookID, asset, catalogSHA256 string) bool
}

// BrowserModel holds the state for the list browser
// Exported for unified TUI integration
type BrowserModel struct {
	list          list.Model
	quitting      bool
	action        BrowserAction
	selected      *BookItem
	selectedBooks []BookItem
	showDetails   bool
	width         int
	height        int

	// Download dependencies
	downloader Downloader

	// Download state
	downloading     bool
	downloadQueue   []BookItem
	currentDownload *BookItem
	downloadPct     float64
	downloadErr     string
	progress        progress.Model
	progressCh      <-chan float64 // Active progress channel for streaming updates

	// Unified mode flag - when true, never returns tea.Quit
	// Instead, sets quitting flag for wrapper to handle
	unifiedMode bool

	// Footer command highlight
	activeCmd string // Key that was just pressed for footer highlight

	// Base title (without column header)
	baseTitle string
}

func (m BrowserModel) Init() tea.Cmd {
	return nil
}

// setActiveCmd sets the footer highlight and returns a tick command to clear it.
func (m *BrowserModel) setActiveCmd(key string) tea.Cmd {
	return SetActiveCmd(&m.activeCmd, key)
}

func (m BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case downloadMsg:
		// Handle download progress
		if msg.err != nil {
			m.downloadErr = fmt.Sprintf("Download failed: %v", msg.err)
			m.downloading = false
			m.currentDownload = nil
			m.progressCh = nil // Clear the channel
			// Continue with next download if any
			if len(m.downloadQueue) > 0 {
				return m, m.startNextDownload()
			}
			return m, nil
		}

		if msg.done {
			// Download complete - mark book as cached
			items := m.list.Items()
			for i, item := range items {
				if bookItem, ok := item.(BookItem); ok {
					if bookItem.Book.ID == msg.bookID {
						bookItem.Cached = true
						// Clear selection on downloaded book
						bookItem.selected = false
						items[i] = bookItem
						break
					}
				}
			}
			m.list.SetItems(items)

			m.downloadPct = 1.0
			m.downloading = false
			m.currentDownload = nil
			m.downloadErr = ""
			m.progressCh = nil // Clear the channel

			// Start next download if any
			if len(m.downloadQueue) > 0 {
				return m, m.startNextDownload()
			}
			return m, nil
		}

		// Intermediate progress update - update progress bar and continue subscription
		m.downloadPct = msg.progress
		cmd := m.progress.SetPercent(msg.progress)

		// Continue subscribing to progress updates
		if m.progressCh != nil {
			return m, tea.Batch(cmd, waitForDownloadProgress(msg.bookID, m.progressCh))
		}
		return m, cmd

	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keys.quit):
			m.quitting = true
			if m.unifiedMode {
				return m, nil
			}
			return m, tea.Quit

		case key.Matches(msg, keys.togglePanel):
			// Toggle details panel
			m.showDetails = !m.showDetails
			m.updateListSize()
			return m, m.setActiveCmd("tab")

		case key.Matches(msg, keys.enter):
			// If details showing, use as action, otherwise toggle details
			if m.showDetails {
				if item, ok := m.list.SelectedItem().(BookItem); ok {
					m.action = ActionShowDetails
					m.selected = &item
					m.quitting = true
					if m.unifiedMode {
						return m, nil
					}
					return m, tea.Quit
				}
			} else {
				m.showDetails = true
				m.updateListSize()
				return m, nil
			}

		case key.Matches(msg, keys.open):
			// Open book
			highlightCmd := m.setActiveCmd("o")
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionOpen
				m.selected = &item
				m.quitting = true
				if m.unifiedMode {
					return m, nil
				}
				return m, tea.Quit
			}
			return m, highlightCmd

		case key.Matches(msg, keys.toggleSelect):
			// Toggle selection on current item (spacebar)
			idx := m.list.Index()
			items := m.list.Items()
			if idx >= 0 && idx < len(items) {
				if bookItem, ok := items[idx].(BookItem); ok {
					bookItem.selected = !bookItem.selected
					items[idx] = bookItem
					m.list.SetItems(items)
				}
			}
			return m, m.setActiveCmd(" ")

		case key.Matches(msg, keys.clearSelect):
			// Clear all selections
			items := m.list.Items()
			for i, item := range items {
				if bookItem, ok := item.(BookItem); ok {
					bookItem.selected = false
					items[i] = bookItem
				}
			}
			m.list.SetItems(items)
			return m, m.setActiveCmd("c")

		case key.Matches(msg, keys.get):
			// Download book(s)
			highlightCmd := m.setActiveCmd("g")

			// Don't start new download if already downloading
			if m.downloading {
				return m, highlightCmd
			}

			// Check if any books are selected for batch download
			items := m.list.Items()
			var booksToDownload []BookItem

			for _, item := range items {
				if bookItem, ok := item.(BookItem); ok && bookItem.selected {
					if !bookItem.Cached {
						booksToDownload = append(booksToDownload, bookItem)
					}
				}
			}

			// Download in background (requires downloader)
			if m.downloader != nil {
				// Batch download selected books
				if len(booksToDownload) > 0 {
					m.downloadQueue = booksToDownload
					return m, tea.Batch(highlightCmd, m.startNextDownload())
				}

				// Single book download
				if item, ok := m.list.SelectedItem().(BookItem); ok {
					if !item.Cached {
						m.downloadQueue = []BookItem{item}
						return m, tea.Batch(highlightCmd, m.startNextDownload())
					}
				}
			}
			return m, highlightCmd

		case key.Matches(msg, keys.uncache):
			// Remove book(s) from cache
			highlightCmd := m.setActiveCmd("x")

			// Only works if downloader available
			if m.downloader == nil {
				return m, highlightCmd
			}

			// Check if any books are selected for batch uncache
			items := m.list.Items()
			var booksToUncache []BookItem

			for _, item := range items {
				if bookItem, ok := item.(BookItem); ok && bookItem.selected {
					if bookItem.Cached {
						booksToUncache = append(booksToUncache, bookItem)
					}
				}
			}

			// Batch uncache selected books
			if len(booksToUncache) > 0 {
				for _, bookItem := range booksToUncache {
					_ = m.downloader.Uncache(bookItem.Owner, bookItem.Repo, bookItem.Book.ID, bookItem.Book.Source.Asset)
					// Update item state
					for i, item := range items {
						if bi, ok := item.(BookItem); ok && bi.Book.ID == bookItem.Book.ID {
							bi.Cached = false
							bi.selected = false
							items[i] = bi
							break
						}
					}
				}
				m.list.SetItems(items)
				return m, highlightCmd
			}

			// Single book uncache
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				if item.Cached {
					_ = m.downloader.Uncache(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
					// Update item state
					idx := m.list.Index()
					if idx >= 0 && idx < len(items) {
						if bookItem, ok := items[idx].(BookItem); ok {
							bookItem.Cached = false
							items[idx] = bookItem
							m.list.SetItems(items)
						}
					}
				}
			}
			return m, highlightCmd

		case key.Matches(msg, keys.sync):
			// Sync modified cached books back to GitHub
			highlightCmd := m.setActiveCmd("s")

			// Only works if downloader available
			if m.downloader == nil {
				return m, highlightCmd
			}

			// Check if any books are selected for batch sync
			items := m.list.Items()
			var booksToSync []BookItem

			for _, item := range items {
				if bookItem, ok := item.(BookItem); ok && bookItem.selected {
					if bookItem.Cached && m.downloader.HasBeenModified(bookItem.Owner, bookItem.Repo, bookItem.Book.ID, bookItem.Book.Source.Asset, bookItem.Book.Checksum.SHA256) {
						booksToSync = append(booksToSync, bookItem)
					}
				}
			}

			// Batch sync selected books
			if len(booksToSync) > 0 {
				for i, bookItem := range booksToSync {
					progressLabel := fmt.Sprintf("[%d/%d] %s", i+1, len(booksToSync), bookItem.Book.ID)
					m.list.NewStatusMessage(progressLabel)
					_, _ = m.downloader.Sync(bookItem.Owner, bookItem.Repo, bookItem.Book.ID, bookItem.Release, bookItem.Book.Source.Asset, bookItem.CatalogPath, bookItem.Book.Checksum.SHA256)
				}
				m.list.NewStatusMessage(fmt.Sprintf("Synced %d books", len(booksToSync)))
				// Clear selections after sync
				for i, item := range items {
					if bi, ok := item.(BookItem); ok {
						bi.selected = false
						items[i] = bi
					}
				}
				m.list.SetItems(items)
				return m, highlightCmd
			}

			// Single book sync
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				if item.Cached && m.downloader.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
					m.list.NewStatusMessage(fmt.Sprintf("Syncing %s...", item.Book.ID))
					_, _ = m.downloader.Sync(item.Owner, item.Repo, item.Book.ID, item.Release, item.Book.Source.Asset, item.CatalogPath, item.Book.Checksum.SHA256)
					m.list.NewStatusMessage(fmt.Sprintf("Synced %s", item.Book.ID))
				} else {
					m.list.NewStatusMessage(fmt.Sprintf("%s: no changes", item.Book.ID))
				}
			}
			return m, highlightCmd

		case key.Matches(msg, keys.edit):
			// Edit book
			highlightCmd := m.setActiveCmd("e")
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionEdit
				m.selected = &item
				m.quitting = true
				if m.unifiedMode {
					return m, nil
				}
				return m, tea.Quit
			}
			return m, highlightCmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateListSize()
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *BrowserModel) updateListSize() {
	// Account for outer container padding, master wrapper border, and footer
	const outerPaddingH = 4 * 2 // left/right padding from outer container
	const outerPaddingV = 2 * 2 // top/bottom padding from outer container
	const masterBorder = 2
	const borderWidth = 1  // right border on list when details shown
	const footerHeight = 2 // divider + footer line

	// Calculate available space after outer padding, border, and footer
	availableWidth := m.width - outerPaddingH - masterBorder
	availableHeight := m.height - outerPaddingV - masterBorder - footerHeight

	if m.showDetails {
		// Split view: list takes ~60% of available width
		listWidth := (availableWidth * 6) / 10

		// Set list size (accounting for right border)
		m.list.SetSize(listWidth-borderWidth, availableHeight-2)
	} else {
		// Full width for list
		m.list.SetSize(availableWidth, availableHeight-2)
	}

	// Update column header in title (rendered by list above items)
	m.list.Title = StyleHeader.Render(m.baseTitle) + "\n" + RenderColumnHeader(m.list.Width())
	m.list.Styles.Title = lipgloss.NewStyle() // Clear title style so inner styles are preserved
}

// waitForDownloadProgress creates a subscription command that waits for the next progress update
func waitForDownloadProgress(bookID string, progressCh <-chan float64) tea.Cmd {
	return func() tea.Msg {
		pct, ok := <-progressCh
		if !ok {
			// Channel closed, download complete
			return downloadMsg{
				bookID:   bookID,
				progress: 1.0,
				done:     true,
			}
		}

		if pct < 0 {
			// Error occurred
			return downloadMsg{
				bookID: bookID,
				done:   true,
				err:    fmt.Errorf("download failed"),
			}
		}

		// Intermediate progress update
		return downloadMsg{
			bookID:   bookID,
			progress: pct,
			done:     false,
		}
	}
}

// startNextDownload pops the next book from queue and starts downloading
func (m *BrowserModel) startNextDownload() tea.Cmd {
	if len(m.downloadQueue) == 0 {
		m.downloading = false
		m.currentDownload = nil
		return nil
	}

	// Pop first book from queue
	book := m.downloadQueue[0]
	m.downloadQueue = m.downloadQueue[1:]

	m.downloading = true
	m.currentDownload = &book
	m.downloadPct = 0.0
	m.downloadErr = ""

	// Start download in background and stream progress
	if m.downloader == nil {
		return func() tea.Msg {
			return downloadMsg{
				bookID: book.Book.ID,
				done:   true,
				err:    fmt.Errorf("downloader not configured"),
			}
		}
	}

	progressCh := make(chan float64, 100)
	m.progressCh = progressCh // Store for continued subscription

	// Download in goroutine
	go func() {
		err := m.downloader.DownloadWithProgress(
			book.Owner,
			book.Repo,
			book.Book.ID,
			book.Book.Source.Release,
			book.Book.Source.Asset,
			book.Book.Checksum.SHA256,
			progressCh,
		)
		// Signal error before closing if download failed
		if err != nil {
			progressCh <- -1.0
		}
		close(progressCh)
	}()

	// Return subscription command that waits for progress updates
	return waitForDownloadProgress(book.Book.ID, progressCh)
}

// RunListBrowser launches an interactive book browser.
// Returns the action and selected book, or error if there was a problem.
// Pass nil downloader to disable background downloads (downloads will exit TUI).
func RunListBrowser(books []BookItem, downloader Downloader) (*BrowserResult, error) {
	if len(books) == 0 {
		return nil, fmt.Errorf("no books to display")
	}

	// Convert BookItems to list.Items
	items := make([]list.Item, len(books))
	for i, b := range books {
		items[i] = b
	}

	// Create the list
	d := delegate.New(renderBookItem)
	l := list.New(items, d, 0, 0)
	title := browserTitle(books)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // Disable built-in help, we'll render custom footer
	l.Styles.Title = StyleHeader
	l.Styles.PaginationStyle = StyleHelp
	l.Styles.HelpStyle = StyleHelp

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	m := BrowserModel{
		list:        l,
		showDetails: false, // Details pane off by default
		downloader:  downloader,
		progress:    prog,
		baseTitle:   title,
	}

	// Run the program with alt screen
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running TUI: %w", err)
	}

	// Return the action result
	if fm, ok := finalModel.(BrowserModel); ok {
		return &BrowserResult{
			Action:    fm.action,
			BookItem:  fm.selected,
			BookItems: fm.selectedBooks,
		}, nil
	}

	return &BrowserResult{Action: ActionNone}, nil
}

// NewBrowserModel creates a BrowserModel for use in unified TUI mode
func NewBrowserModel(books []BookItem, downloader Downloader, unifiedMode bool) BrowserModel {
	// Convert BookItems to list.Items
	items := make([]list.Item, len(books))
	for i, b := range books {
		items[i] = b
	}

	// Create the list
	d := delegate.New(renderBookItem)
	l := list.New(items, d, 0, 0)
	title := browserTitle(books)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // Disable built-in help, we'll render custom footer
	l.Styles.Title = StyleHeader
	l.Styles.PaginationStyle = StyleHelp
	l.Styles.HelpStyle = StyleHelp

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	return BrowserModel{
		list:        l,
		showDetails: false, // Details pane off by default
		downloader:  downloader,
		progress:    prog,
		unifiedMode: unifiedMode,
		baseTitle:   title,
	}
}

// IsQuitting returns true if the browser wants to quit
func (m BrowserModel) IsQuitting() bool {
	return m.quitting
}

// GetAction returns the action that was requested
func (m BrowserModel) GetAction() BrowserAction {
	return m.action
}

// GetSelected returns the selected book item
func (m BrowserModel) GetSelected() *BookItem {
	return m.selected
}

// GetSelectedBooks returns the list of selected books (for multi-select)
func (m BrowserModel) GetSelectedBooks() []BookItem {
	return m.selectedBooks
}
