package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
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

// shelfDelegate renders shelf options
type shelfDelegate struct{}

func (d shelfDelegate) Height() int  { return 1 }
func (d shelfDelegate) Spacing() int { return 0 }
func (d shelfDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d shelfDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	shelfItem, ok := item.(ShelfOption)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	display := fmt.Sprintf("%s (%s)", shelfItem.Name, StyleHelp.Render(shelfItem.Repo))

	if isSelected {
		fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

type shelfPickerModel struct {
	list     list.Model
	quitting bool
	selected string
	err      error
}

type shelfPickerKeys struct {
	quit   key.Binding
	select_ key.Binding
}

var shelfKeys = shelfPickerKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "cancel"),
	),
	select_: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}

func (m shelfPickerModel) Init() tea.Cmd {
	return nil
}

func (m shelfPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, shelfKeys.quit):
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case key.Matches(msg, shelfKeys.select_):
			if item, ok := m.list.SelectedItem().(ShelfOption); ok {
				m.selected = item.Name
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

func (m shelfPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return StyleBorder.Render(m.list.View())
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

	// Create list
	delegate := shelfDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Shelf"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	m := shelfPickerModel{
		list: l,
	}

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

	if fm.err != nil {
		return "", fm.err
	}

	return fm.selected, nil
}
