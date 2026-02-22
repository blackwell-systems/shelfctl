package tui

import (
	"fmt"
	"io"

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

// HubContext holds optional context info to display in the hub
type HubContext struct {
	ShelfCount int
	BookCount  int
	HasCache   bool
}

// menuItems defines the menu in logical order
var menuItems = []MenuItem{
	// Browse & View
	{Key: "browse", Label: "Browse Library", Description: "View and search your books", Available: true},
	{Key: "shelves", Label: "View Shelves", Description: "Show all configured shelves and book counts", Available: true},
	{Key: "index", Label: "Generate HTML Index", Description: "Create web page for local browsing", Available: true},
	// Add & Import
	{Key: "shelve", Label: "Add Book", Description: "Add a new book to your library", Available: true},
	{Key: "shelve-url", Label: "Add from URL", Description: "Download and add a book from URL", Available: true},
	{Key: "import-repo", Label: "Import from Repository", Description: "Migrate books from another repo", Available: true},
	// Manage
	{Key: "edit-book", Label: "Edit Book", Description: "Update metadata for a book", Available: true},
	{Key: "move", Label: "Move Book", Description: "Transfer a book to another shelf or release", Available: true},
	// Remove
	{Key: "delete-book", Label: "Delete Book", Description: "Remove a book from your library", Available: true},
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

	display := fmt.Sprintf("%-20s %s", label, desc)

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

type hubModel struct {
	list     list.Model
	quitting bool
	action   string // which action was selected
	err      error
	context  HubContext
	width    int
	height   int
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

		// Account for outer padding, inner padding, border, and header content
		const outerPaddingH = 4 * 2 // left/right outer padding
		const outerPaddingV = 2 * 2 // top/bottom outer padding
		const innerPaddingH = 1 + 2 // left (1) + right (2) inner padding
		const headerLines = 4       // header + status + spacing
		h, v := StyleBorder.GetFrameSize()

		listWidth := msg.Width - outerPaddingH - innerPaddingH - h
		listHeight := msg.Height - outerPaddingV - v - headerLines

		if listWidth < 40 {
			listWidth = 40
		}
		if listHeight < 5 {
			listHeight = 5
		}

		m.list.SetSize(listWidth, listHeight)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m hubModel) View() string {
	if m.quitting {
		return ""
	}

	// Outer container for centering - creates margin around the box
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

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

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

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
