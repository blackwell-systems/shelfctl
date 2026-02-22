package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BookItem represents a book in the list with metadata.
type BookItem struct {
	Book      catalog.Book
	ShelfName string
	Cached    bool
	HasCover  bool
	CoverPath string
	Owner     string
	Repo      string
	selected  bool // For multi-select mode
}

// FilterValue returns a string used for filtering in the list
func (b BookItem) FilterValue() string {
	// Include ID, title, tags, and shelf name in filter
	tags := strings.Join(b.Book.Tags, " ")
	return fmt.Sprintf("%s %s %s %s", b.Book.ID, b.Book.Title, tags, b.ShelfName)
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

// renderBookItem renders a book in the browser list
func renderBookItem(w io.Writer, m list.Model, index int, item list.Item) {
	bookItem, ok := item.(BookItem)
	if !ok {
		return
	}

	// Build the display string
	var s strings.Builder

	// Cover indicator (camera emoji if cover exists)
	coverMark := ""
	if bookItem.HasCover {
		coverMark = "📷 "
	}

	// Book ID (truncate if too long)
	id := bookItem.Book.ID
	const maxIDWidth = 20
	if len(id) > maxIDWidth {
		id = id[:maxIDWidth-1] + "…"
	}
	idStr := fmt.Sprintf("%-20s", id)

	// Title (truncate based on available width)
	title := bookItem.Book.Title
	const maxTitleWidth = 50
	if len(title) > maxTitleWidth {
		title = title[:maxTitleWidth-1] + "…"
	}

	// Tags (truncate if too long)
	tagStr := ""
	if len(bookItem.Book.Tags) > 0 {
		tagsJoined := strings.Join(bookItem.Book.Tags, ",")
		const maxTagWidth = 30
		if len(tagsJoined) > maxTagWidth {
			tagsJoined = tagsJoined[:maxTagWidth-1] + "…"
		}
		tagStr = " " + StyleTag.Render("["+tagsJoined+"]")
	}

	// Cached indicator
	cachedMark := ""
	if bookItem.Cached {
		cachedMark = " " + StyleCached.Render("[local]")
	}

	// Check if this item is selected
	isSelected := index == m.Index()

	if isSelected {
		// Highlight selected item
		s.WriteString(StyleHighlight.Render("› " + coverMark + idStr + " " + title + tagStr + cachedMark))
	} else {
		// Normal rendering
		s.WriteString("  " + coverMark + StyleNormal.Render(idStr) + " " + title + tagStr + cachedMark)
	}

	_, _ = fmt.Fprint(w, s.String())
}

// keyMap defines keyboard shortcuts
type keyMap struct {
	quit        key.Binding
	enter       key.Binding
	open        key.Binding
	get         key.Binding
	edit        key.Binding
	filter      key.Binding
	togglePanel key.Binding
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
	Action   BrowserAction
	BookItem *BookItem
}

// model holds the state for the list browser
type model struct {
	list        list.Model
	quitting    bool
	action      BrowserAction
	selected    *BookItem
	showDetails bool
	width       int
	height      int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keys.quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.togglePanel):
			// Toggle details panel
			m.showDetails = !m.showDetails
			m.updateListSize()
			return m, nil

		case key.Matches(msg, keys.enter):
			// If details showing, use as action, otherwise toggle details
			if m.showDetails {
				if item, ok := m.list.SelectedItem().(BookItem); ok {
					m.action = ActionShowDetails
					m.selected = &item
					m.quitting = true
					return m, tea.Quit
				}
			} else {
				m.showDetails = true
				m.updateListSize()
				return m, nil
			}

		case key.Matches(msg, keys.open):
			// Open book
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionOpen
				m.selected = &item
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, keys.get):
			// Download book
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionDownload
				m.selected = &item
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, keys.edit):
			// Edit book
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionEdit
				m.selected = &item
				m.quitting = true
				return m, tea.Quit
			}
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

func (m *model) updateListSize() {
	// Account for outer container padding, master wrapper border, and inner padding
	const outerPaddingH = 4 * 2 // left/right padding from outer container
	const outerPaddingV = 2 * 2 // top/bottom padding from outer container
	const masterBorder = 2
	const borderWidth = 1 // right border on list when details shown

	// Calculate available space after outer padding and border
	availableWidth := m.width - outerPaddingH - masterBorder
	availableHeight := m.height - outerPaddingV - masterBorder

	if m.showDetails {
		// Split view: list takes ~60% of available width
		listWidth := (availableWidth * 6) / 10

		// Set list size (accounting for right border)
		m.list.SetSize(listWidth-borderWidth, availableHeight-2)
	} else {
		// Full width for list
		m.list.SetSize(availableWidth, availableHeight-2)
	}
}

func (m model) renderDetailsPane() string {
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

	// ID
	s.WriteString(StyleHighlight.Render("ID: "))
	s.WriteString(bookItem.Book.ID)
	s.WriteString("\n\n")

	// Title
	s.WriteString(StyleHighlight.Render("Title: "))
	s.WriteString(bookItem.Book.Title)
	s.WriteString("\n\n")

	// Author
	if bookItem.Book.Author != "" {
		s.WriteString(StyleHighlight.Render("Author: "))
		s.WriteString(bookItem.Book.Author)
		s.WriteString("\n\n")
	}

	// Tags
	if len(bookItem.Book.Tags) > 0 {
		s.WriteString(StyleHighlight.Render("Tags: "))
		s.WriteString(StyleTag.Render(strings.Join(bookItem.Book.Tags, ", ")))
		s.WriteString("\n\n")
	}

	// Shelf
	s.WriteString(StyleHighlight.Render("Shelf: "))
	s.WriteString(bookItem.ShelfName)
	s.WriteString("\n\n")

	// Repository
	s.WriteString(StyleHighlight.Render("Repository: "))
	fmt.Fprintf(&s, "%s/%s", bookItem.Owner, bookItem.Repo)
	s.WriteString("\n\n")

	// Cache status
	s.WriteString(StyleHighlight.Render("Cached: "))
	if bookItem.Cached {
		s.WriteString(StyleCached.Render("✓ Yes"))
	} else {
		s.WriteString("No")
	}
	s.WriteString("\n\n")

	// Asset info
	s.WriteString(StyleHighlight.Render("Format: "))
	s.WriteString(bookItem.Book.Format)
	s.WriteString("\n")

	s.WriteString(StyleHighlight.Render("Asset: "))
	s.WriteString(bookItem.Book.Source.Asset)
	s.WriteString("\n")

	// Apply details panel styling
	return detailsStyle.Render(s.String())
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Outer container for centering - adds margin around the entire box
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

	// Inner content box with border
	masterStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray).
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

	var content string
	if m.showDetails {
		// Split-panel layout: compose panels then wrap
		// Add border on right side of list to create solid divider
		listStyle := lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorGray)
		listView := listStyle.Render(m.list.View())
		detailsView := m.renderDetailsPane()

		// Join horizontally: list (with border) + details
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			detailsView,
		)
	} else {
		// Single panel: list only
		content = m.list.View()
	}

	// Apply inner box border, then outer container for floating effect
	return outerStyle.Render(masterStyle.Render(content))
}

// RunListBrowser launches an interactive book browser.
// Returns the action and selected book, or error if there was a problem.
func RunListBrowser(books []BookItem) (*BrowserResult, error) {
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
	l.Title = "Books"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.PaginationStyle = StyleHelp
	l.Styles.HelpStyle = StyleHelp

	// Set help keybindings
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.open, keys.get, keys.edit, keys.togglePanel}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.open, keys.get, keys.edit, keys.enter, keys.togglePanel}
	}

	m := model{
		list:        l,
		showDetails: true, // Show details pane by default
	}

	// Run the program with alt screen
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running TUI: %w", err)
	}

	// Return the action result
	if fm, ok := finalModel.(model); ok {
		return &BrowserResult{
			Action:   fm.action,
			BookItem: fm.selected,
		}, nil
	}

	return &BrowserResult{Action: ActionNone}, nil
}
