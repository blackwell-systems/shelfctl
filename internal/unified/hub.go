package unified

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HubModel is the unified-mode version of the hub menu
type HubModel struct {
	list        list.Model
	context     tui.HubContext
	width       int
	height      int
	shelfData   string
	showDetails bool
	detailsType string
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

// NewHubModel creates a new hub model for unified mode
func NewHubModel(ctx tui.HubContext) HubModel {
	// Filter menu items based on context
	menuItems := tui.GetMenuItems()
	var items []list.Item
	for _, item := range menuItems {
		// Hide shelf-related actions if no shelves configured
		if ctx.ShelfCount == 0 {
			key := item.GetKey()
			if key == "browse" || key == "shelves" || key == "shelve" ||
				key == "edit-book" || key == "delete-book" || key == "delete-shelf" {
				continue
			}
		}
		// Hide browse, edit-book, move, and delete-book if there are no books
		if ctx.BookCount == 0 {
			key := item.GetKey()
			if key == "browse" || key == "edit-book" || key == "move" || key == "delete-book" {
				continue
			}
		}
		items = append(items, item)
	}

	// Create list with custom delegate
	d := delegate.New(renderHubMenuItem)
	l := list.New(items, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.HelpStyle = tui.StyleHelp

	// Set help
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{hubKeyMap.selectItem}
	}

	return HubModel{
		list:    l,
		context: ctx,
	}
}

func (m HubModel) Init() tea.Cmd {
	return nil
}

func (m HubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, hubKeyMap.quit):
			// In unified mode, quitting hub means quitting the app
			return m, func() tea.Msg { return QuitAppMsg{} }

		case key.Matches(msg, hubKeyMap.selectItem):
			if item, ok := m.list.SelectedItem().(tui.MenuItem); ok {
				itemKey := item.GetKey()
				if itemKey == "quit" {
					return m, func() tea.Msg { return QuitAppMsg{} }
				}
				// Emit navigation message to switch views
				return m, func() tea.Msg {
					return NavigateMsg{Target: itemKey}
				}
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
	if item, ok := m.list.SelectedItem().(tui.MenuItem); ok {
		itemKey := item.GetKey()
		if itemKey == "shelves" || itemKey == "cache-info" {
			m.showDetails = true
			m.detailsType = itemKey
		} else {
			m.showDetails = false
			m.detailsType = ""
		}
		m.updateListSize()
	}

	return m, cmd
}

func (m HubModel) View() string {
	// Outer container for centering
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4)

	// Create header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1).
		Render("shelfctl - Personal Library Manager")

	// Create status bar if we have context
	var statusBar string
	if m.context.ShelfCount > 0 {
		status := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("  %d shelves", m.context.ShelfCount))
		if m.context.BookCount > 0 {
			status = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(fmt.Sprintf("  %d shelves · %d books", m.context.ShelfCount, m.context.BookCount))
		}
		statusBar = status
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
			BorderForeground(tui.ColorGray)
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

	// Add padding inside the border
	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1)

	return outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(content)))
}

func (m *HubModel) updateListSize() {
	// Account for outer padding, inner padding, border, and header content
	const outerPaddingH = 4 * 2
	const outerPaddingV = 2 * 2
	const innerPaddingH = 1 + 2
	const headerLines = 4
	const borderWidth = 1
	h, v := tui.StyleBorder.GetFrameSize()

	availableWidth := m.width - outerPaddingH - innerPaddingH - h
	listHeight := m.height - outerPaddingV - v - headerLines

	if m.showDetails {
		const detailsPanelWidth = 45 + 2
		listWidth := availableWidth - detailsPanelWidth - borderWidth
		if listWidth < 30 {
			listWidth = 30
		}
		m.list.SetSize(listWidth, listHeight)
	} else {
		if availableWidth < 40 {
			availableWidth = 40
		}
		m.list.SetSize(availableWidth, listHeight)
	}

	if listHeight < 5 {
		m.list.SetSize(availableWidth, 5)
	}
}

func (m HubModel) renderDetailsPane() string {
	switch m.detailsType {
	case "shelves":
		return m.renderShelvesDetails()
	case "cache-info":
		return m.renderCacheDetails()
	default:
		return ""
	}
}

func (m HubModel) renderShelvesDetails() string {
	if len(m.context.ShelfDetails) == 0 {
		return ""
	}

	const detailsWidth = 45
	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(1, 1)

	var s strings.Builder
	s.WriteString(tui.StyleHeader.Render("Configured Shelves"))
	s.WriteString("\n\n")

	for _, shelf := range m.context.ShelfDetails {
		s.WriteString(tui.StyleHighlight.Render(shelf.Name))
		s.WriteString("\n")
		fmt.Fprintf(&s, "  Repo: %s/%s\n", shelf.Owner, shelf.Repo)
		fmt.Fprintf(&s, "  Books: %d\n", shelf.BookCount)
		fmt.Fprintf(&s, "  Status: %s\n", shelf.Status)
		s.WriteString("\n")
	}

	return detailsStyle.Render(s.String())
}

func (m HubModel) renderCacheDetails() string {
	const detailsWidth = 45
	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(1, 1)

	var s strings.Builder
	s.WriteString(tui.StyleHeader.Render("Cache Statistics"))
	s.WriteString("\n\n")

	// Total books
	s.WriteString(tui.StyleHighlight.Render("Total Books: "))
	fmt.Fprintf(&s, "%d\n", m.context.BookCount)

	// Cached count
	s.WriteString(tui.StyleHighlight.Render("Cached: "))
	if m.context.CachedCount > 0 {
		fmt.Fprintf(&s, "%s (%d books)\n", tui.StyleCached.Render(fmt.Sprintf("✓ %d", m.context.CachedCount)), m.context.CachedCount)
	} else {
		s.WriteString("0\n")
	}

	// Uncached count
	uncached := m.context.BookCount - m.context.CachedCount
	if uncached > 0 {
		s.WriteString(tui.StyleHighlight.Render("Not Cached: "))
		fmt.Fprintf(&s, "%d\n", uncached)
	}

	// Modified count and list
	if m.context.ModifiedCount > 0 {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Modified Books:"))
		s.WriteString("\n")
		s.WriteString(tui.StyleHelp.Render(fmt.Sprintf("  %d books with local changes", m.context.ModifiedCount)))
		s.WriteString("\n\n")

		displayCount := len(m.context.ModifiedBooks)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			book := m.context.ModifiedBooks[i]
			s.WriteString("  • ")
			s.WriteString(tui.StyleHighlight.Render(book.ID))
			s.WriteString("\n")
		}

		if len(m.context.ModifiedBooks) > 10 {
			s.WriteString(tui.StyleHelp.Render(fmt.Sprintf("  ... and %d more", len(m.context.ModifiedBooks)-10)))
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(tui.StyleHelp.Render("  Press 's' in browse or run 'sync --all'"))
		s.WriteString("\n")
	}

	// Cache size
	if m.context.CacheSize > 0 {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Disk Usage: "))
		fmt.Fprintf(&s, "%s\n", formatBytes(m.context.CacheSize))
	}

	// Cache directory
	if m.context.CacheDir != "" {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Location: "))
		s.WriteString(tui.StyleHelp.Render(m.context.CacheDir))
		s.WriteString("\n")
	}

	return detailsStyle.Render(s.String())
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func renderHubMenuItem(w io.Writer, m list.Model, index int, item list.Item) {
	menuItem, ok := item.(tui.MenuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Format label and description
	label := menuItem.GetLabel()
	desc := tui.StyleHelp.Render(menuItem.GetDescription())

	display := fmt.Sprintf("%-20s   %s", label, desc)

	if isSelected {
		_, _ = fmt.Fprint(w, tui.StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+tui.StyleNormal.Render(display))
	}
}
