package multiselect

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// SelectableItem extends list.Item with selection state.
// Items that implement this interface can be selected/deselected.
type SelectableItem interface {
	list.Item
	IsSelected() bool
	SetSelected(bool)
	// IsSelectable returns false for items that shouldn't show checkboxes (e.g., directories)
	IsSelectable() bool
}

// Model wraps a bubbles/list.Model with multi-select capabilities.
type Model struct {
	List            list.Model
	selectedPaths   map[string]bool // Persists selection state by unique key
	showCount       bool            // Show selection count in title
	originalTitle   string          // Title without count suffix
	checkboxChecked string          // Checkbox appearance when checked (default: "[✓] ")
	checkboxEmpty   string          // Checkbox appearance when unchecked (default: "[ ] ")
}

// New creates a new multi-select model wrapping the given list.
func New(l list.Model) Model {
	return Model{
		List:            l,
		selectedPaths:   make(map[string]bool),
		showCount:       true,
		originalTitle:   l.Title,
		checkboxChecked: "[✓] ",
		checkboxEmpty:   "[ ] ",
	}
}

// SetCheckboxStyle customizes the checkbox appearance.
func (m *Model) SetCheckboxStyle(checked, empty string) {
	m.checkboxChecked = checked
	m.checkboxEmpty = empty
}

// SetShowCount controls whether selection count appears in title.
func (m *Model) SetShowCount(show bool) {
	m.showCount = show
}

// Toggle toggles the selection state of the current item.
// Returns true if the item was toggled, false if it's not selectable.
func (m *Model) Toggle() bool {
	item, ok := m.List.SelectedItem().(SelectableItem)
	if !ok || !item.IsSelectable() {
		return false
	}

	// Toggle selection state
	key := item.FilterValue() // Use FilterValue as unique key
	m.selectedPaths[key] = !m.selectedPaths[key]

	// Rebuild list items with updated selection
	m.rebuildItems()
	m.updateTitle()

	return true
}

// Select marks an item as selected by its key.
func (m *Model) Select(key string) {
	m.selectedPaths[key] = true
	m.rebuildItems()
	m.updateTitle()
}

// Deselect marks an item as deselected by its key.
func (m *Model) Deselect(key string) {
	delete(m.selectedPaths, key)
	m.rebuildItems()
	m.updateTitle()
}

// ClearSelection removes all selections.
func (m *Model) ClearSelection() {
	m.selectedPaths = make(map[string]bool)
	m.rebuildItems()
	m.updateTitle()
}

// SelectedKeys returns the keys of all selected items.
func (m *Model) SelectedKeys() []string {
	var keys []string
	for key, selected := range m.selectedPaths {
		if selected {
			keys = append(keys, key)
		}
	}
	return keys
}

// SelectedCount returns the number of selected items.
func (m *Model) SelectedCount() int {
	count := 0
	for _, selected := range m.selectedPaths {
		if selected {
			count++
		}
	}
	return count
}

// rebuildItems updates all list items with current selection state.
// This creates new items with updated selection state since items may be passed by value.
func (m *Model) rebuildItems() {
	items := m.List.Items()
	newItems := make([]list.Item, len(items))

	for i, item := range items {
		if selectableItem, ok := item.(SelectableItem); ok {
			key := selectableItem.FilterValue()
			// Create a copy with updated selection state
			selectableItem.SetSelected(m.selectedPaths[key])
			newItems[i] = selectableItem
		} else {
			newItems[i] = item
		}
	}
	m.List.SetItems(newItems)
}

// updateTitle updates the list title with selection count if enabled.
func (m *Model) updateTitle() {
	if m.showCount {
		count := m.SelectedCount()
		m.List.Title = fmt.Sprintf("%s (%d selected)", m.originalTitle, count)
	}
}

// SetTitle updates the base title (without count).
func (m *Model) SetTitle(title string) {
	m.originalTitle = title
	m.updateTitle()
}

// RestoreSelectionState restores selection state for items (e.g., after directory navigation).
// This is called when items are replaced in the list.
func (m *Model) RestoreSelectionState() {
	m.rebuildItems()
	m.updateTitle()
}

// CheckboxPrefix returns the appropriate checkbox prefix for an item.
// This is meant to be used by custom item delegates.
func (m *Model) CheckboxPrefix(item SelectableItem) string {
	if !item.IsSelectable() {
		return "  " // No checkbox for non-selectable items
	}
	if item.IsSelected() {
		return m.checkboxChecked
	}
	return m.checkboxEmpty
}

// Update handles messages for the multi-select model.
// Pass key messages through to this before handling other updates.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

// View renders the list.
func (m Model) View() string {
	return m.List.View()
}
