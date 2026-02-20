package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// BookItem represents a book in the list with metadata.
type BookItem struct {
	Book      catalog.Book
	ShelfName string
	Cached    bool
	Owner     string
	Repo      string
}

// FilterValue returns a string used for filtering in the list
func (b BookItem) FilterValue() string {
	// Include ID, title, tags, and shelf name in filter
	tags := strings.Join(b.Book.Tags, " ")
	return fmt.Sprintf("%s %s %s %s", b.Book.ID, b.Book.Title, tags, b.ShelfName)
}

// Custom list item delegate for rendering books
type bookDelegate struct{}

func (d bookDelegate) Height() int  { return 1 }
func (d bookDelegate) Spacing() int { return 0 }
func (d bookDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d bookDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	bookItem, ok := item.(BookItem)
	if !ok {
		return
	}

	// Build the display string
	var s strings.Builder

	// Book ID (fixed width for alignment)
	idStr := fmt.Sprintf("%-22s", bookItem.Book.ID)

	// Title
	title := bookItem.Book.Title

	// Tags
	tagStr := ""
	if len(bookItem.Book.Tags) > 0 {
		tagStr = " " + StyleTag.Render("["+strings.Join(bookItem.Book.Tags, ",")+"]")
	}

	// Cached indicator
	cachedMark := ""
	if bookItem.Cached {
		cachedMark = " " + StyleCached.Render("✓")
	}

	// Check if this item is selected
	isSelected := index == m.Index()

	if isSelected {
		// Highlight selected item
		s.WriteString(StyleHighlight.Render("› " + idStr + " " + title + tagStr + cachedMark))
	} else {
		// Normal rendering
		s.WriteString("  " + StyleNormal.Render(idStr) + " " + title + tagStr + cachedMark)
	}

	_, _ = fmt.Fprint(w, s.String())
}

// keyMap defines keyboard shortcuts
type keyMap struct {
	quit   key.Binding
	enter  key.Binding
	open   key.Binding
	get    key.Binding
	filter key.Binding
}

var keys = keyMap{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "details"),
	),
	open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	get: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "download"),
	),
	filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
}

// BrowserAction represents an action requested from the browser
type BrowserAction string

const (
	ActionNone        BrowserAction = ""
	ActionShowDetails BrowserAction = "details"
	ActionOpen        BrowserAction = "open"
	ActionDownload    BrowserAction = "download"
)

// BrowserResult holds the result of a browser session
type BrowserResult struct {
	Action   BrowserAction
	BookItem *BookItem
}

// model holds the state for the list browser
type model struct {
	list     list.Model
	quitting bool
	action   BrowserAction
	selected *BookItem
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

		case key.Matches(msg, keys.enter):
			// Show book details
			if item, ok := m.list.SelectedItem().(BookItem); ok {
				m.action = ActionShowDetails
				m.selected = &item
				m.quitting = true
				return m, tea.Quit
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
		}

	case tea.WindowSizeMsg:
		h, v := StyleBorder.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return StyleBorder.Render(m.list.View())
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
	delegate := bookDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Books"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.PaginationStyle = StyleHelp
	l.Styles.HelpStyle = StyleHelp

	// Set help keybindings
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.open, keys.get}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.open, keys.get, keys.enter}
	}

	m := model{list: l}

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
