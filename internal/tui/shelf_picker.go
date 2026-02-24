package tui

import (
	"fmt"
	"io"

	"github.com/blackwell-systems/bubbletea-components/picker"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ShelfOption represents a shelf choice.
type ShelfOption struct {
	Name string
	Repo string
}

// FilterValue implements list.Item
func (s ShelfOption) FilterValue() string {
	return s.Name + " " + s.Repo
}

// renderShelfOption renders a shelf option in the picker list
func renderShelfOption(w io.Writer, m list.Model, index int, item list.Item) {
	shelfItem, ok := item.(ShelfOption)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	display := fmt.Sprintf("%s (%s)", shelfItem.Name, StyleHelp.Render(shelfItem.Repo))

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

type shelfPickerModel struct {
	base     *picker.Base
	selected string
}

func (m shelfPickerModel) Init() tea.Cmd {
	return nil
}

func (m shelfPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.base.Update(msg)

	// Extract selection when quitting without error
	if m.base.IsQuitting() && m.base.Error() == nil {
		if item, ok := m.base.SelectedItem().(ShelfOption); ok {
			m.selected = item.Name
		}
	}

	return m, cmd
}

func (m shelfPickerModel) View() string {
	return m.base.View()
}

// RunShelfPicker launches an interactive shelf selector.
// Returns the selected shelf name, or error if canceled.
func RunShelfPicker(shelves []ShelfOption) (string, error) {
	if len(shelves) == 0 {
		return "", fmt.Errorf("no shelves configured")
	}

	// If only one shelf, just return it
	if len(shelves) == 1 {
		return shelves[0].Name, nil
	}

	// Convert to list items
	items := make([]list.Item, len(shelves))
	for i, s := range shelves {
		items[i] = s
	}

	// Create list with base delegate
	d := delegate.New(renderShelfOption)
	l := list.New(items, d, 0, 0)
	l.Title = "Select Shelf"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

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

	m := shelfPickerModel{base: base}

	// Run the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running shelf picker: %w", err)
	}

	fm, ok := finalModel.(shelfPickerModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if fm.base.Error() != nil {
		return "", fm.base.Error()
	}

	return fm.selected, nil
}
