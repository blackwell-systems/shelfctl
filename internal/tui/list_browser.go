package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
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
func (d bookDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
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
	quit        key.Binding
	enter       key.Binding
	open        key.Binding
	get         key.Binding
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
	h, v := StyleBorder.GetFrameSize()

	if m.showDetails {
		// Split view: list takes 50% width, details pane takes other 50%
		listWidth := (m.width / 2) - h
		m.list.SetSize(listWidth, m.height-v)
	} else {
		// Full width for list
		m.list.SetSize(m.width-h, m.height-v)
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

	var s strings.Builder

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
	s.WriteString(fmt.Sprintf("%s/%s", bookItem.Owner, bookItem.Repo))
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

	return s.String()
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.showDetails {
		// Split-panel layout
		listView := m.list.View()
		detailsView := m.renderDetailsPane()

		// Use lipgloss to join horizontally
		combined := lipgloss.JoinHorizontal(
			lipgloss.Top,
			StyleBorder.Render(listView),
			StyleBorder.Render(detailsView),
		)
		return combined
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
		return []key.Binding{keys.open, keys.get, keys.togglePanel}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.open, keys.get, keys.enter, keys.togglePanel}
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
