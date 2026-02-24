package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BookItem represents a book in the list with metadata.
type BookItem struct {
	Book        catalog.Book
	ShelfName   string
	Cached      bool
	HasCover    bool
	CoverPath   string
	Owner       string
	Repo        string
	Release     string // Release tag for this book
	CatalogPath string // Path to catalog.yml in repo
	selected    bool   // For multi-select mode
}

// FilterValue returns a string used for filtering in the list
func (b BookItem) FilterValue() string {
	// Include ID, title, tags, and shelf name in filter
	tags := strings.Join(b.Book.Tags, " ")
	return fmt.Sprintf("%s %s %s %s", b.Book.ID, b.Book.Title, tags, b.ShelfName)
}

// truncateText truncates a string to maxWidth with ellipsis
func truncateText(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return s[:maxWidth-1] + "…"
}

// formatBytes formats bytes as human-readable size
func formatBytes(n int64) string {
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

// IsSelected implements multiselect.SelectableItem
func (b BookItem) IsSelected() bool {
	return b.selected
}

// SetSelected implements multiselect.SelectableItem
func (b *BookItem) SetSelected(selected bool) {
	b.selected = selected
}

// IsSelectable implements multiselect.SelectableItem
// All books are selectable
func (b BookItem) IsSelectable() bool {
	return true
}

// browserTitle derives a title from the books' shelf names.
// If all books belong to one shelf, returns "Shelf: <name>".
// If books span multiple shelves, returns "All Shelves".
func browserTitle(books []BookItem) string {
	shelves := make(map[string]struct{})
	for _, b := range books {
		shelves[b.ShelfName] = struct{}{}
	}
	if len(shelves) == 1 {
		for name := range shelves {
			return "Shelf: " + name
		}
	}
	return "All Shelves"
}

// Column width constraints
const (
	minTitleWidth  = 12
	maxTitleWidth  = 48
	minAuthorWidth = 8
	maxAuthorWidth = 45
	minTagWidth    = 6
	minShelfWidth  = 5
	maxShelfWidth  = 16
	minCachedWidth = 7
	maxCachedWidth = 10
	columnGap      = 1
)

// computeColumnWidths distributes available width proportionally across columns.
func computeColumnWidths(totalWidth int) (titleW, authorW, tagW, shelfW, cachedW int) {
	// Reserve space for prefix ("› " or "  " or "[✓] ") and gaps between columns
	prefix := 2
	gaps := columnGap * 4 // 4 gaps between 5 columns
	usable := totalWidth - prefix - gaps
	if usable < minTitleWidth+minAuthorWidth+minTagWidth+minShelfWidth+minCachedWidth {
		return minTitleWidth, minAuthorWidth, minTagWidth, minShelfWidth, minCachedWidth
	}
	titleW = usable * 45 / 100
	if titleW > maxTitleWidth {
		titleW = maxTitleWidth
	}
	remaining := usable - titleW
	authorW = remaining * 35 / 100
	if authorW > maxAuthorWidth {
		authorW = maxAuthorWidth
	}
	shelfW = remaining * 20 / 100
	if shelfW > maxShelfWidth {
		shelfW = maxShelfWidth
	}
	tagW = remaining * 25 / 100
	cachedW = remaining - authorW - tagW - shelfW // remainder
	if cachedW > maxCachedWidth {
		cachedW = maxCachedWidth
	}

	// Enforce minimums
	if titleW < minTitleWidth {
		titleW = minTitleWidth
	}
	if authorW < minAuthorWidth {
		authorW = minAuthorWidth
	}
	if tagW < minTagWidth {
		tagW = minTagWidth
	}
	if shelfW < minShelfWidth {
		shelfW = minShelfWidth
	}
	if cachedW < minCachedWidth {
		cachedW = minCachedWidth
	}
	return
}

// padOrTruncate pads s to exactly width, truncating with "…" if necessary.
func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) > width {
		if width <= 1 {
			return "…"
		}
		return s[:width-1] + "…"
	}
	if len(s) < width {
		return s + strings.Repeat(" ", width-len(s))
	}
	return s
}

// renderBookItem renders a book in the browser list with fixed-width columns.
func renderBookItem(w io.Writer, m list.Model, index int, item list.Item) {
	bookItem, ok := item.(BookItem)
	if !ok {
		return
	}

	listWidth := m.Width()
	if listWidth <= 0 {
		listWidth = 80
	}
	titleW, authorW, tagW, shelfW, cachedW := computeColumnWidths(listWidth)

	gap := strings.Repeat(" ", columnGap)

	// Cursor / selection prefix
	isCursor := index == m.Index()
	prefix := "  "
	if isCursor {
		prefix = lipgloss.NewStyle().Foreground(ColorOrange).Render("›") + " "
	}
	if bookItem.selected {
		prefix = lipgloss.NewStyle().Foreground(ColorTealLight).Bold(true).Render("✓") + "  "
		titleW--
		if titleW < minTitleWidth {
			titleW = minTitleWidth
		}
	}

	// Build column content
	titleCol := padOrTruncate(bookItem.Book.Title, titleW)
	authorCol := padOrTruncate(bookItem.Book.Author, authorW)

	tagStr := strings.Join(bookItem.Book.Tags, " · ")
	tagCol := padOrTruncate(tagStr, tagW)

	shelfCol := padOrTruncate(bookItem.ShelfName, shelfW)

	cachedStr := ""
	if bookItem.Cached {
		cachedStr = "✓ local"
	}
	cachedCol := padOrTruncate(cachedStr, cachedW)

	isCursorSelected := isCursor

	// Style each column
	var titleStyled, authorStyled, tagStyled, shelfStyled, cachedStyled string
	if isCursorSelected {
		titleStyled = StyleHighlight.Render(titleCol)
		authorStyled = lipgloss.NewStyle().Foreground(ColorOrange).Faint(true).Render(authorCol)
		tagStyled = lipgloss.NewStyle().Foreground(ColorTealLight).Render(tagCol)
		shelfStyled = lipgloss.NewStyle().Foreground(ColorOrange).Faint(true).Render(shelfCol)
		cachedStyled = StyleHighlight.Render(cachedCol)
	} else {
		titleStyled = StyleNormal.Render(titleCol)
		authorStyled = StyleHelp.Render(authorCol)
		tagStyled = StyleTag.Render(tagCol)
		shelfStyled = StyleHelp.Render(shelfCol)
		if bookItem.Cached {
			cachedStyled = StyleCached.Render(cachedCol)
		} else {
			cachedStyled = cachedCol
		}
	}

	line := prefix + titleStyled + gap + authorStyled + gap + tagStyled + gap + shelfStyled + gap + cachedStyled
	_, _ = fmt.Fprint(w, line)
}

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

func (m BrowserModel) renderDetailsPane() string {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return ""
	}

	bookItem, ok := selectedItem.(BookItem)
	if !ok {
		return ""
	}

	// Calculate details pane width (40% of screen, accounting for divider and master border)
	detailsWidth := ((m.width - 2) * 4) / 10
	if detailsWidth < 30 {
		detailsWidth = 30 // Minimum width for readability
	}

	// Calculate max text width: panel width minus padding (2 chars)
	// Account for label widths (e.g., "Repository: " is 12 chars)
	const labelWidth = 12 // longest label
	maxTextWidth := detailsWidth - 2 - labelWidth
	if maxTextWidth < 10 {
		maxTextWidth = 10
	}

	// Style for the details content area
	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(0, 1)

	var s strings.Builder

	// Show cover image if available and terminal supports it
	if bookItem.HasCover {
		protocol := DetectImageProtocol()
		if protocol != ProtocolNone {
			if img := RenderInlineImage(bookItem.CoverPath, protocol); img != "" {
				s.WriteString(img)
				s.WriteString("\n\n")
			}
		}
	}

	// Title
	s.WriteString(StyleHeader.Render("Book Details"))
	s.WriteString("\n\n")

	// Title
	s.WriteString(StyleHighlight.Render("Title: "))
	s.WriteString(truncateText(bookItem.Book.Title, maxTextWidth))
	s.WriteString("\n\n")

	// Author
	if bookItem.Book.Author != "" {
		s.WriteString(StyleHighlight.Render("Author: "))
		s.WriteString(truncateText(bookItem.Book.Author, maxTextWidth))
		s.WriteString("\n\n")
	}

	// Year
	if bookItem.Book.Year > 0 {
		s.WriteString(StyleHighlight.Render("Year: "))
		fmt.Fprintf(&s, "%d", bookItem.Book.Year)
		s.WriteString("\n\n")
	}

	// Tags
	if len(bookItem.Book.Tags) > 0 {
		s.WriteString(StyleHighlight.Render("Tags: "))
		s.WriteString("\n")
		for _, t := range bookItem.Book.Tags {
			pill := lipgloss.NewStyle().
				Background(ColorTealDim).Foreground(ColorTealLight).
				Padding(0, 1).Render(t)
			s.WriteString(pill + " ")
		}
		s.WriteString("\n\n")
	}

	// Shelf
	s.WriteString(StyleHighlight.Render("Shelf: "))
	s.WriteString(truncateText(bookItem.ShelfName, maxTextWidth))
	s.WriteString("\n\n")

	// Repository
	s.WriteString(StyleHighlight.Render("Repository: "))
	repoText := fmt.Sprintf("%s/%s", bookItem.Owner, bookItem.Repo)
	s.WriteString(truncateText(repoText, maxTextWidth))
	s.WriteString("\n\n")

	// Cache status
	s.WriteString(StyleHighlight.Render("Cached: "))
	if bookItem.Cached {
		s.WriteString(StyleCached.Render("✓ Yes"))
	} else {
		s.WriteString("No")
	}
	s.WriteString("\n\n")

	// Size
	if bookItem.Book.SizeBytes > 0 {
		s.WriteString(StyleHighlight.Render("Size: "))
		s.WriteString(formatBytes(bookItem.Book.SizeBytes))
		s.WriteString("\n\n")
	}

	// Format
	s.WriteString(StyleHighlight.Render("Format: "))
	s.WriteString(truncateText(bookItem.Book.Format, maxTextWidth))
	s.WriteString("\n")

	// Apply details panel styling
	return detailsStyle.Render(s.String())
}

// renderFooter creates a footer with all available keyboard shortcuts.
// The shortcut matching activeCmd is rendered with StyleHighlight.
func (m BrowserModel) renderFooter() string {
	return RenderFooterBar([]ShortcutEntry{
		{Key: "", Label: "↑/↓ navigate"},
		{Key: "/", Label: "/ filter"},
		{Key: "", Label: "enter action"},
		{Key: "o", Label: "o open"},
		{Key: "g", Label: "g download"},
		{Key: "x", Label: "x uncache"},
		{Key: "s", Label: "s sync"},
		{Key: "e", Label: "e edit"},
		{Key: " ", Label: "space select"},
		{Key: "c", Label: "c clear"},
		{Key: "tab", Label: "tab detail toggle"},
		{Key: "", Label: "q quit"},
	}, m.activeCmd)
}

func (m BrowserModel) View() string {
	if m.quitting {
		return ""
	}

	// Outer container for centering - adds margin around the entire box
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

	// Inner content box with border
	masterStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTeal).
		Padding(0)

	// Calculate dimensions for inner box
	// Subtract outer padding (2*2 vertical, 4*2 horizontal) and border (2 each side)
	if m.width > 0 && m.height > 0 {
		innerWidth := m.width - (4 * 2) - 2   // outer padding + border
		innerHeight := m.height - (2 * 2) - 2 // outer padding + border

		// Ensure minimum size
		if innerWidth < 60 {
			innerWidth = 60
		}
		if innerHeight < 10 {
			innerHeight = 10
		}

		masterStyle = masterStyle.
			Width(innerWidth).
			Height(innerHeight)
	}

	var mainContent string
	if m.showDetails {
		// Split-panel layout: compose panels then wrap
		// Add border on right side of list to create solid divider
		listStyle := lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorTeal)
		listView := listStyle.Render(m.list.View())
		detailsView := m.renderDetailsPane()

		// Join horizontally: list (with border) + details
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			detailsView,
		)
	} else {
		// Single panel: list only
		mainContent = m.list.View()
	}

	// Create footer with divider
	// Calculate divider width based on content width
	dividerWidth := m.width - (4 * 2) - 2 // outer padding + border
	if dividerWidth < 40 {
		dividerWidth = 40
	}
	divider := lipgloss.NewStyle().
		Foreground(ColorTeal).
		Width(dividerWidth).
		Render(strings.Repeat("─", dividerWidth))
	footer := m.renderFooter()

	// Compose: main content + divider + footer
	content := lipgloss.JoinVertical(lipgloss.Left, mainContent, divider, footer)

	// Apply inner box border
	boxed := masterStyle.Render(content)

	// Add download progress if active
	if m.downloading && m.currentDownload != nil {
		progressBar := m.progress.ViewAs(m.downloadPct)
		label := fmt.Sprintf("Downloading %s", m.currentDownload.Book.ID)
		if len(m.downloadQueue) > 0 {
			label = fmt.Sprintf("[%d remaining] %s", len(m.downloadQueue)+1, label)
		}

		progressView := lipgloss.NewStyle().
			Foreground(ColorYellow).
			Render(label + "\n" + progressBar)

		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", progressView)
	} else if m.downloadErr != "" {
		errorView := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(m.downloadErr)
		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", errorView)
	}

	// Apply outer container for floating effect
	return outerStyle.Render(boxed)
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
