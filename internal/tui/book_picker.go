package tui

import (
	"fmt"
	"io"

	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/blackwell-systems/shelfctl/internal/tui/picker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// renderBookPickerItem renders a book item in picker mode
func renderBookPickerItem(w io.Writer, m list.Model, index int, item list.Item) {
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
	base     *picker.Base
	selected *BookItem
}

func (m bookPickerModel) Init() tea.Cmd {
	return nil
}

func (m bookPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.base.Update(msg)

	// Extract selection when quitting without error
	if m.base.IsQuitting() && m.base.Error() == nil {
		if item, ok := m.base.SelectedItem().(BookItem); ok {
			m.selected = &item
		}
	}

	return m, cmd
}

func (m bookPickerModel) View() string {
	return m.base.View()
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

	// Create the list with base delegate
	d := delegate.New(renderBookPickerItem)
	l := list.New(items, d, 0, 0)
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

	// Use standard picker keys
	keys := NewPickerKeys()

	// Create base picker
	base := picker.New(picker.Config{
		List:        l,
		QuitKeys:    keys.Quit,
		SelectKeys:  keys.Select,
		ShowBorder:  true,
		BorderStyle: StyleBorder,
		OnSelect: func(item list.Item) bool {
			return true // Quit after selection
		},
	})

	m := bookPickerModel{base: base}

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

	if fm, ok := finalModel.(bookPickerModel); ok && fm.base.Error() != nil {
		return BookItem{}, fm.base.Error()
	}

	return BookItem{}, fmt.Errorf("canceled")
}
