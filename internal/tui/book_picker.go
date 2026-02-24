package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/bubbletea-components/multiselect"
	"github.com/blackwell-systems/bubbletea-components/picker"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
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

	// Tags
	tagStr := ""
	if len(bookItem.Book.Tags) > 0 {
		tagsJoined := strings.Join(bookItem.Book.Tags, ",")
		const maxTagWidth = 30
		if len(tagsJoined) > maxTagWidth {
			tagsJoined = tagsJoined[:maxTagWidth-1] + "…"
		}
		tagStr = " " + StyleTag.Render("["+tagsJoined+"]")
	}

	// Cached indicator
	cachedStr := ""
	if bookItem.Cached {
		cachedStr = " " + StyleCached.Render("[local]")
	}

	shelfInfo := " " + StyleHelp.Render("["+bookItem.ShelfName+"]")

	// Check if this item is selected
	isSelected := index == m.Index()

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+idStr+" "+title+tagStr+cachedStr+shelfInfo))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(idStr)+" "+title+tagStr+cachedStr+shelfInfo)
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

// multiBookPickerModel is the model for multi-select book picking
type multiBookPickerModel struct {
	ms            multiselect.Model
	quitting      bool
	err           error
	selectedBooks []BookItem
}

func (m multiBookPickerModel) Init() tea.Cmd {
	return nil
}

func (m multiBookPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window resize - account for border
		h, v := StyleBorder.GetFrameSize()
		m.ms.List.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.ms.List.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case " ":
			// Toggle checkbox
			m.ms.Toggle()
			return m, nil

		case "enter":
			// Collect all checked books
			items := m.ms.List.Items()
			for _, item := range items {
				if bookItem, ok := item.(*BookItem); ok && bookItem.IsSelected() {
					m.selectedBooks = append(m.selectedBooks, *bookItem)
				}
			}
			// Fallback: if nothing checked, select current book
			if len(m.selectedBooks) == 0 {
				if item, ok := m.ms.List.SelectedItem().(*BookItem); ok {
					m.selectedBooks = []BookItem{*item}
				}
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Update the multiselect model
	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m multiBookPickerModel) View() string {
	if m.quitting {
		return ""
	}
	return StyleBorder.Render(m.ms.View())
}

// renderBookPickerItemMulti renders a book item with checkbox for multi-select mode
func renderBookPickerItemMulti(w io.Writer, m list.Model, index int, item list.Item, ms *multiselect.Model) {
	bookItem, ok := item.(*BookItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Get checkbox prefix
	prefix := ms.CheckboxPrefix(bookItem)

	// Build the display string
	idStr := fmt.Sprintf("%-22s", bookItem.Book.ID)
	title := bookItem.Book.Title

	// Tags
	tagStr := ""
	if len(bookItem.Book.Tags) > 0 {
		tagsJoined := strings.Join(bookItem.Book.Tags, ",")
		const maxTagWidth = 30
		if len(tagsJoined) > maxTagWidth {
			tagsJoined = tagsJoined[:maxTagWidth-1] + "…"
		}
		tagStr = " " + StyleTag.Render("["+tagsJoined+"]")
	}

	// Cached indicator
	cachedStr := ""
	if bookItem.Cached {
		cachedStr = " " + StyleCached.Render("[local]")
	}

	shelfInfo := " " + StyleHelp.Render("["+bookItem.ShelfName+"]")

	display := prefix + idStr + " " + title + tagStr + cachedStr + shelfInfo

	if isSelected {
		_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

// RunBookPickerMulti launches an interactive book picker with multi-select support.
// Users can toggle checkboxes with spacebar and confirm with enter.
// Returns a slice of selected BookItems or error if canceled.
// NewBookPickerMultiModel creates a configured multiselect model for book picking
// without running it. Used by unified TUI views that embed the picker.
func NewBookPickerMultiModel(books []BookItem, title string) (multiselect.Model, error) {
	if len(books) == 0 {
		return multiselect.Model{}, fmt.Errorf("no books to display")
	}

	// Convert BookItems to list.Items (as pointers so selection state persists)
	items := make([]list.Item, len(books))
	for i := range books {
		items[i] = &books[i]
	}

	// Create base list with temporary delegate
	tempDelegate := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {})
	l := list.New(items, tempDelegate, 0, 0)
	if title != "" {
		l.Title = title
	} else {
		l.Title = "Select books"
	}
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	// Wrap with multi-select
	ms := multiselect.New(l)
	ms.SetTitle(title)

	// Create proper delegate with multi-select support
	d := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {
		renderBookPickerItemMulti(w, m, index, item, &ms)
	})
	ms.List.SetDelegate(d)

	// Set help keybindings
	toggleKey := key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	)
	selectKey := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)
	ms.List.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{toggleKey, selectKey}
	}

	// Restore selection state
	ms.RestoreSelectionState()

	return ms, nil
}

// CollectSelectedBooks returns the selected BookItems from a multiselect model
func CollectSelectedBooks(ms *multiselect.Model) []BookItem {
	var selected []BookItem
	items := ms.List.Items()
	for _, item := range items {
		if bookItem, ok := item.(*BookItem); ok && bookItem.IsSelected() {
			selected = append(selected, *bookItem)
		}
	}
	// Fallback: if nothing checked, select current item
	if len(selected) == 0 {
		if item, ok := ms.List.SelectedItem().(*BookItem); ok {
			selected = []BookItem{*item}
		}
	}
	return selected
}

func RunBookPickerMulti(books []BookItem, title string) ([]BookItem, error) {
	if len(books) == 0 {
		return nil, fmt.Errorf("no books to display")
	}

	// Convert BookItems to list.Items (as pointers so selection state persists)
	items := make([]list.Item, len(books))
	for i := range books {
		items[i] = &books[i]
	}

	// Create base list with temporary delegate
	tempDelegate := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {})
	l := list.New(items, tempDelegate, 0, 0)
	if title != "" {
		l.Title = title
	} else {
		l.Title = "Select books"
	}
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	// Wrap with multi-select
	ms := multiselect.New(l)
	ms.SetTitle(title)

	// Create proper delegate with multi-select support
	d := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {
		renderBookPickerItemMulti(w, m, index, item, &ms)
	})
	ms.List.SetDelegate(d)

	// Set help keybindings
	toggleKey := key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	)
	selectKey := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)
	ms.List.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{toggleKey, selectKey}
	}

	// Restore selection state
	ms.RestoreSelectionState()

	// Create model with multi-select
	m := multiBookPickerModel{ms: ms}

	// Run the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running TUI: %w", err)
	}

	fm, ok := finalModel.(multiBookPickerModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	if fm.err != nil {
		return nil, fm.err
	}

	return fm.selectedBooks, nil
}
