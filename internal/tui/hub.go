package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents an action in the hub menu
type MenuItem struct {
	Key         string
	Label       string
	Description string
	Available   bool // whether this feature is implemented
}

// FilterValue implements list.Item
func (m MenuItem) FilterValue() string {
	return m.Label + " " + m.Description
}

// Getters for unified mode
func (m MenuItem) GetKey() string         { return m.Key }
func (m MenuItem) GetLabel() string       { return m.Label }
func (m MenuItem) GetDescription() string { return m.Description }

// ShelfStatus represents the status of a single shelf
type ShelfStatus struct {
	Name      string
	Repo      string
	Owner     string
	BookCount int
	Status    string // "✓ Healthy", "⚠ Warning", "✗ Error"
}

// ModifiedBook represents a book with local changes
type ModifiedBook struct {
	ID    string
	Title string
}

// HubContext holds optional context info to display in the hub
type HubContext struct {
	ShelfCount   int
	BookCount    int
	HasCache     bool
	ShelfDetails []ShelfStatus // for inline display
	// Cache stats
	CachedCount   int
	ModifiedCount int
	ModifiedBooks []ModifiedBook
	CacheSize     int64
	CacheDir      string
}

// GetMenuItems returns the menu items for use in unified mode
func GetMenuItems() []MenuItem {
	return menuItems
}

// menuItems defines the menu in frequency-of-use order
var menuItems = []MenuItem{
	// Most frequent
	{Key: "browse", Label: "Browse Library", Description: "View and search your books", Available: true},
	{Key: "shelve", Label: "Add Book", Description: "Add a new book to your library", Available: true},
	{Key: "edit-book", Label: "Edit Book", Description: "Update metadata for a book", Available: true},
	// Fairly frequent
	{Key: "cache-info", Label: "Cache Info", Description: "View cache statistics and disk usage", Available: true},
	{Key: "cache-clear", Label: "Clear Cache", Description: "Remove books from local cache", Available: true},
	// Occasional
	{Key: "shelves", Label: "View Shelves", Description: "Show all configured shelves and book counts", Available: true},
	{Key: "index", Label: "Generate HTML Index", Description: "Create web page for local browsing", Available: true},
	{Key: "shelve-url", Label: "Add from URL", Description: "Download and add a book from URL", Available: true},
	// Rare
	{Key: "move", Label: "Move Book", Description: "Transfer a book to another shelf or release", Available: true},
	{Key: "delete-book", Label: "Delete Book", Description: "Remove a book from your library", Available: true},
	{Key: "import-repo", Label: "Import from Repository", Description: "Migrate books from another repo", Available: true},
	{Key: "create-shelf", Label: "Create Shelf", Description: "Add a new shelf repository to your library", Available: true},
	{Key: "delete-shelf", Label: "Delete Shelf", Description: "Remove a shelf from configuration", Available: true},
	// Exit
	{Key: "quit", Label: "Quit", Description: "Exit shelfctl", Available: true},
}

// renderMenuItem renders a menu item in the hub
func renderMenuItem(w io.Writer, m list.Model, index int, item list.Item) {
	menuItem, ok := item.(MenuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Format label and description
	label := menuItem.Label
	desc := StyleHelp.Render(menuItem.Description)

	display := fmt.Sprintf("%-25s   %s", label, desc)

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

type hubModel struct {
	list        list.Model
	quitting    bool
	action      string // which action was selected
	err         error
	context     HubContext
	width       int
	height      int
	shelfData   string // rendered shelves table for details panel
	showDetails bool   // whether to show the details panel
	detailsType string // "shelves" or "cache-info"
}

type hubKeys struct {
	quit       key.Binding
	selectItem key.Binding
}

var hubKeyMap = hubKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	selectItem: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}

func (m hubModel) Init() tea.Cmd {
	return nil
}

func (m hubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, hubKeyMap.quit):
			m.quitting = true
			m.action = "quit"
			return m, tea.Quit

		case key.Matches(msg, hubKeyMap.selectItem):
			if item, ok := m.list.SelectedItem().(MenuItem); ok {
				m.action = item.Key
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

	// Check if we should show details panel
	if item, ok := m.list.SelectedItem().(MenuItem); ok {
		if item.Key == "shelves" || item.Key == "cache-info" {
			m.showDetails = true
			m.detailsType = item.Key
		} else {
			m.showDetails = false
			m.detailsType = ""
		}
		m.updateListSize()
	}

	return m, cmd
}

func (m *hubModel) updateListSize() {
	// Account for outer padding, inner padding, border, and header content
	const outerPaddingH = 4 * 2 // left/right outer padding
	const outerPaddingV = 2 * 2 // top/bottom outer padding
	const innerPaddingH = 1 + 2 // left (1) + right (2) inner padding
	const headerLines = 4       // header + status + spacing
	const borderWidth = 1       // right border on list when details shown
	h, v := StyleBorder.GetFrameSize()

	availableWidth := m.width - outerPaddingH - innerPaddingH - h
	listHeight := m.height - outerPaddingV - v - headerLines

	if m.showDetails {
		// Split view: list takes remaining width after fixed details panel
		const detailsPanelWidth = 45 + 2 // panel width + padding
		listWidth := availableWidth - detailsPanelWidth - borderWidth
		if listWidth < 30 {
			listWidth = 30
		}
		m.list.SetSize(listWidth, listHeight)
	} else {
		// Full width for list
		if availableWidth < 40 {
			availableWidth = 40
		}
		m.list.SetSize(availableWidth, listHeight)
	}

	if listHeight < 5 {
		m.list.SetSize(availableWidth, 5)
	}
}

func (m hubModel) renderDetailsPane() string {
	// Route to appropriate details panel based on type
	switch m.detailsType {
	case "shelves":
		return m.renderShelvesDetails()
	case "cache-info":
		return m.renderCacheDetails()
	default:
		return ""
	}
}

func (m hubModel) renderShelvesDetails() string {
	if len(m.context.ShelfDetails) == 0 {
		return ""
	}

	// Fixed width for details panel - just enough for content
	const detailsWidth = 45

	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(1, 1)

	// Render shelves table
	var s strings.Builder
	s.WriteString(StyleHeader.Render("Configured Shelves"))
	s.WriteString("\n\n")

	dim := lipgloss.NewStyle().Foreground(ColorGray)
	num := lipgloss.NewStyle().Foreground(ColorTealLight).Bold(true)
	for _, shelf := range m.context.ShelfDetails {
		s.WriteString(StyleHighlight.Render(shelf.Name))
		s.WriteString("\n")
		s.WriteString(dim.Render(fmt.Sprintf("  %s/%s", shelf.Owner, shelf.Repo)))
		s.WriteString("\n")
		s.WriteString(dim.Render("  ") + num.Render(fmt.Sprintf("%d", shelf.BookCount)) + dim.Render(" books"))
		s.WriteString("\n")
		s.WriteString(dim.Render("  ") + shelf.Status)
		s.WriteString("\n\n")
	}

	return detailsStyle.Render(s.String())
}

func (m hubModel) renderCacheDetails() string {
	// Fixed width for details panel
	const detailsWidth = 45

	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(1, 1)

	var s strings.Builder
	s.WriteString(StyleHeader.Render("Cache Statistics"))
	s.WriteString("\n\n")

	// Total books
	s.WriteString(StyleHighlight.Render("Total Books: "))
	fmt.Fprintf(&s, "%d\n", m.context.BookCount)

	// Cached count
	s.WriteString(StyleHighlight.Render("Cached: "))
	if m.context.CachedCount > 0 {
		fmt.Fprintf(&s, "%s (%d books)\n", StyleCached.Render(fmt.Sprintf("✓ %d", m.context.CachedCount)), m.context.CachedCount)
	} else {
		s.WriteString("0\n")
	}

	// Uncached count
	uncached := m.context.BookCount - m.context.CachedCount
	if uncached > 0 {
		s.WriteString(StyleHighlight.Render("Not Cached: "))
		fmt.Fprintf(&s, "%d\n", uncached)
	}

	// Modified count and list
	if m.context.ModifiedCount > 0 {
		s.WriteString("\n")
		s.WriteString(StyleHighlight.Render("Modified Books:"))
		s.WriteString("\n")
		s.WriteString(StyleHelp.Render(fmt.Sprintf("  %d books with local changes", m.context.ModifiedCount)))
		s.WriteString("\n\n")

		// List modified books (limit to avoid overflow)
		displayCount := len(m.context.ModifiedBooks)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			book := m.context.ModifiedBooks[i]
			s.WriteString("  • ")
			s.WriteString(StyleHighlight.Render(book.ID))
			s.WriteString("\n")
		}

		if len(m.context.ModifiedBooks) > 10 {
			s.WriteString(StyleHelp.Render(fmt.Sprintf("  ... and %d more", len(m.context.ModifiedBooks)-10)))
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(StyleHelp.Render("  Press 's' in browse or run 'sync --all'"))
		s.WriteString("\n")
	}

	// Cache size
	if m.context.CacheSize > 0 {
		s.WriteString("\n")
		s.WriteString(StyleHighlight.Render("Disk Usage: "))
		fmt.Fprintf(&s, "%s\n", formatBytes(m.context.CacheSize))
	}

	// Cache directory
	if m.context.CacheDir != "" {
		s.WriteString("\n")
		s.WriteString(StyleHighlight.Render("Location: "))
		s.WriteString(StyleHelp.Render(m.context.CacheDir))
		s.WriteString("\n")
	}

	return detailsStyle.Render(s.String())
}

func (m hubModel) View() string {
	if m.quitting {
		return ""
	}

	// Outer container for centering - creates margin around the box
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

	// Two-tone wordmark header
	wordmark := lipgloss.NewStyle().Bold(true).Foreground(ColorOrange).Render("shelf") +
		lipgloss.NewStyle().Bold(true).Foreground(ColorTealLight).Render("ctl")
	header := lipgloss.NewStyle().Padding(0, 1).Render(wordmark + "  " +
		lipgloss.NewStyle().Foreground(ColorGray).Render("Personal Library Manager"))

	// Status bar with brand teal accent
	var statusBar string
	if m.context.ShelfCount > 0 {
		num := lipgloss.NewStyle().Foreground(ColorTealLight).Bold(true)
		dim := lipgloss.NewStyle().Foreground(ColorGray)
		stat := "  " + num.Render(fmt.Sprintf("%d", m.context.ShelfCount)) + dim.Render(" shelves")
		if m.context.BookCount > 0 {
			stat += "  " + num.Render(fmt.Sprintf("%d", m.context.BookCount)) + dim.Render(" books")
		}
		if m.context.CachedCount > 0 {
			stat += "  " + num.Render(fmt.Sprintf("%d", m.context.CachedCount)) + dim.Render(" cached")
		}
		statusBar = stat
	}

	// Combine header, status, and list
	parts := []string{header}
	if statusBar != "" {
		parts = append(parts, statusBar)
	}
	parts = append(parts, m.list.View())

	listContent := lipgloss.JoinVertical(lipgloss.Left, parts...)

	var content string
	if m.showDetails {
		// Split-panel layout
		listStyle := lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorTeal)
		listView := listStyle.Render(listContent)
		detailsView := m.renderDetailsPane()

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			detailsView,
		)
	} else {
		// Single panel: menu only
		content = listContent
	}

	// Add padding inside the border (more on right to prevent text bleeding to edge)
	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1) // top, right, bottom, left

	// Apply inner padding, then border, then outer padding for floating effect
	return outerStyle.Render(StyleBorder.Render(innerPadding.Render(content)))
}

// RunHub launches the interactive hub menu
// Returns the selected action key, or error if canceled
func RunHub(ctx HubContext) (string, error) {
	// Filter menu items based on context
	var items []list.Item
	for _, item := range menuItems {
		// Hide shelf-related actions if no shelves configured
		if ctx.ShelfCount == 0 {
			if item.Key == "browse" || item.Key == "shelves" || item.Key == "shelve" || item.Key == "edit-book" || item.Key == "delete-book" || item.Key == "delete-shelf" {
				continue
			}
		}
		// Hide browse, edit-book, move, and delete-book if there are no books
		if ctx.BookCount == 0 && (item.Key == "browse" || item.Key == "edit-book" || item.Key == "move" || item.Key == "delete-book") {
			continue
		}
		items = append(items, item)
	}

	// Create list
	d := delegate.NewWithSpacing(renderMenuItem, 1)
	l := list.New(items, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.HelpStyle = StyleHelp

	// Set help
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{hubKeyMap.selectItem}
	}

	m := hubModel{
		list:    l,
		context: ctx,
	}

	// Run the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running hub: %w", err)
	}

	fm, ok := finalModel.(hubModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if fm.err != nil {
		return "", fm.err
	}

	return fm.action, nil
}
