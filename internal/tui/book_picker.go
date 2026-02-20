package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// bookPickerDelegate renders book items in picker mode
type bookPickerDelegate struct{}

func (d bookPickerDelegate) Height() int  { return 1 }
func (d bookPickerDelegate) Spacing() int { return 0 }
func (d bookPickerDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d bookPickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	bookItem, ok := item.(BookItem)
	if !ok {
		return
	}

	// Build the display string
	idStr := fmt.Sprintf("%-22s", bookItem.Book.ID)
	title := bookItem.Book.Title
	shelfInfo := StyleHelp.Render(fmt.Sprintf("[%s]", bookItem.ShelfName))

	// Check if this item is selected
	isSelected := index == m.Index()

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+idStr+" "+title+" "+shelfInfo))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(idStr)+" "+title+" "+shelfInfo)
	}
}

type bookPickerModel struct {
	list     list.Model
	selected *BookItem
	quitting bool
}

func (m bookPickerModel) Init() tea.Cmd {
	return nil
}

func (m bookPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(BookItem); ok {
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

func (m bookPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return StyleBorder.Render(m.list.View())
}

// RunBookPicker launches an interactive book picker.
// Returns the selected BookItem or error if canceled.
func RunBookPicker(books []BookItem, title string) (BookItem, error) {
	if len(books) == 0 {
		return BookItem{}, fmt.Errorf("no books to display")
	}

	// Convert BookItems to list.Items
	items := make([]list.Item, len(books))
	for i, b := range books {
		items[i] = b
	}

	// Create the list
	delegate := bookPickerDelegate{}
	l := list.New(items, delegate, 0, 0)
	if title != "" {
		l.Title = title
	} else {
		l.Title = "Select a book"
	}
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.PaginationStyle = StyleHelp
	l.Styles.HelpStyle = StyleHelp

	// Set help keybindings
	selectKey := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{selectKey}
	}

	m := bookPickerModel{list: l}

	// Run the program with alt screen
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return BookItem{}, fmt.Errorf("running TUI: %w", err)
	}

	// Check if user selected a book
	if fm, ok := finalModel.(bookPickerModel); ok {
		if fm.selected != nil {
			return *fm.selected, nil
		}
	}

	return BookItem{}, fmt.Errorf("canceled")
}
