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
	Icon        string
	Available   bool // whether this feature is implemented
}

// MenuSeparator is a non-interactive section header between menu groups
type MenuSeparator struct {
	Title string
}

func (s MenuSeparator) FilterValue() string { return "" }

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

// menuSection groups related menu items under a title
type menuSection struct {
	Title string
	Items []MenuItem
}

// menuSections defines the menu grouped by category
var menuSections = []menuSection{
	{Title: "Library", Items: []MenuItem{
		{Key: "browse", Icon: "▶", Label: "Browse", Description: "View and search your books", Available: true},
		{Key: "shelve", Icon: "+", Label: "Add Book", Description: "Add a new book to your library", Available: true},
		{Key: "shelve-url", Icon: "↓", Label: "Add from URL", Description: "Download and add a book from URL", Available: true},
		{Key: "edit-book", Icon: "~", Label: "Edit Book", Description: "Update metadata for a book", Available: true},
	}},
	{Title: "Organize", Items: []MenuItem{
		{Key: "move", Icon: "→", Label: "Move Book", Description: "Transfer a book to another shelf or release", Available: true},
		{Key: "delete-book", Icon: "✕", Label: "Delete Book", Description: "Remove a book from your library", Available: true},
	}},
	{Title: "Shelves", Items: []MenuItem{
		{Key: "shelves", Icon: "≡", Label: "View Shelves", Description: "Show all configured shelves and book counts", Available: true},
		{Key: "create-shelf", Icon: "+", Label: "Create Shelf", Description: "Add a new shelf repository to your library", Available: true},
		{Key: "delete-shelf", Icon: "✕", Label: "Delete Shelf", Description: "Remove a shelf from configuration", Available: true},
	}},
	{Title: "Tools", Items: []MenuItem{
		{Key: "import-shelf", Icon: "↻", Label: "Import from Shelf", Description: "Import books from another shelfctl shelf", Available: true},
		{Key: "import-repo", Icon: "↻", Label: "Import from Repo", Description: "Migrate files from any GitHub repo", Available: true},
		{Key: "index", Icon: "#", Label: "HTML Index", Description: "Create web page for local browsing", Available: true},
	}},
	{Title: "Cache", Items: []MenuItem{
		{Key: "cache-info", Icon: "i", Label: "Cache Info", Description: "View cache statistics and disk usage", Available: true},
		{Key: "cache-clear", Icon: "⊗", Label: "Clear Cache", Description: "Remove books from local cache", Available: true},
	}},
}

// GetMenuItems returns all menu items flattened for backward compatibility
func GetMenuItems() []MenuItem {
	var items []MenuItem
	for _, section := range menuSections {
		items = append(items, section.Items...)
	}
	return items
}

// BuildFilteredMenuItems returns a []list.Item with MenuSeparator headers,
// filtered based on context (no shelves, no books).
func BuildFilteredMenuItems(ctx HubContext) []list.Item {
	var result []list.Item
	for _, section := range menuSections {
		var sectionItems []list.Item
		for _, item := range section.Items {
			// Hide shelf-dependent actions when no shelves configured
			if ctx.ShelfCount == 0 {
				switch item.Key {
				case "browse", "shelves", "shelve", "shelve-url", "edit-book", "delete-book", "delete-shelf":
					continue
				}
			}
			// Hide book-dependent actions when no books exist
			if ctx.BookCount == 0 {
				switch item.Key {
				case "browse", "edit-book", "move", "delete-book":
					continue
				}
			}
			sectionItems = append(sectionItems, item)
		}
		if len(sectionItems) > 0 {
			result = append(result, MenuSeparator{Title: section.Title})
			result = append(result, sectionItems...)
		}
	}
	return result
}

// renderMenuItem renders a menu item or section separator in the hub
func renderMenuItem(w io.Writer, m list.Model, index int, item list.Item) {
	// Render section separator
	if sep, ok := item.(MenuSeparator); ok {
		line := lipgloss.NewStyle().Foreground(ColorGray).Render("─── " + sep.Title + " ───")
		_, _ = fmt.Fprint(w, "  "+line)
		return
	}

	menuItem, ok := item.(MenuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

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
	// Track navigation direction so we can skip separators after list update
	navDir := 0
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.list.FilterState() != list.Filtering {
		switch keyMsg.String() {
		case "up", "k":
			navDir = -1
		case "down", "j":
			navDir = 1
		}
	}

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

	// Skip over separators: if navigation landed on one, keep stepping.
	// If no real item exists in that direction, stay put (no wrap past edges).
	if navDir != 0 {
		if _, isSep := m.list.SelectedItem().(MenuSeparator); isSep {
			items := m.list.Items()
			idx := m.list.Index()
			found := false
			for {
				idx += navDir
				if idx < 0 || idx >= len(items) {
					break
				}
				if _, isSep := items[idx].(MenuSeparator); !isSep {
					m.list.Select(idx)
					found = true
					break
				}
			}
			if !found {
				// Restore to nearest real item in opposite direction
				idx = m.list.Index()
				for {
					idx -= navDir
					if idx < 0 || idx >= len(items) {
						break
					}
					if _, isSep := items[idx].(MenuSeparator); !isSep {
						m.list.Select(idx)
						break
					}
				}
			}
		}
	}

	// Check if we should show details panel
	if _, isSep := m.list.SelectedItem().(MenuSeparator); isSep {
		m.showDetails = false
		m.detailsType = ""
		m.updateListSize()
	} else if item, ok := m.list.SelectedItem().(MenuItem); ok {
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
	wordmark := lipgloss.NewStyle().Bold(true).Foreground(ColorOrange).Render("shelf") + lipgloss.NewStyle().Bold(true).Foreground(ColorTealLight).Render("ctl")
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
	items := BuildFilteredMenuItems(ctx)

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

	// Start cursor on first real item (index 0 is a separator)
	for i, item := range items {
		if _, isSep := item.(MenuSeparator); !isSep {
			l.Select(i)
			break
		}
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
