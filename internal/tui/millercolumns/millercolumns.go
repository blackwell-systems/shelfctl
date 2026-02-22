package millercolumns

import (
	"github.com/charmbracelet/lipgloss"
)

// Model manages a stack of columns for hierarchical navigation.
// Each column represents one level in the hierarchy (e.g., directory level, menu level).
type Model struct {
	columns              []Column
	focusedCol           int
	width                int
	height               int
	maxVisibleColumns    int
	focusedBorderColor   lipgloss.Color
	unfocusedBorderColor lipgloss.Color
	borderStyle          lipgloss.Style
}

// Column represents a single level in the Miller columns view.
type Column struct {
	ID   string // Unique identifier (e.g., file path, menu ID)
	List any    // The model for this column (should have Update and View methods)
}

// Config configures a Miller columns model.
type Config struct {
	// MaxVisibleColumns is the maximum number of columns to show at once (default: 3)
	MaxVisibleColumns int

	// FocusedBorderColor is the border color for the focused column (default: cyan)
	FocusedBorderColor lipgloss.Color

	// UnfocusedBorderColor is the border color for unfocused columns (default: gray)
	UnfocusedBorderColor lipgloss.Color

	// BorderStyle is the base border style to use (required - should include border and padding)
	BorderStyle lipgloss.Style
}

// New creates a new Miller columns model.
func New(config Config) Model {
	if config.MaxVisibleColumns == 0 {
		config.MaxVisibleColumns = 3
	}
	if config.FocusedBorderColor == "" {
		config.FocusedBorderColor = lipgloss.Color("6") // Cyan
	}
	if config.UnfocusedBorderColor == "" {
		config.UnfocusedBorderColor = lipgloss.Color("240") // Gray
	}

	// Use provided border style (assumed to be set by caller)
	borderStyle := config.BorderStyle

	return Model{
		columns:              []Column{},
		focusedCol:           0,
		maxVisibleColumns:    config.MaxVisibleColumns,
		focusedBorderColor:   config.FocusedBorderColor,
		unfocusedBorderColor: config.UnfocusedBorderColor,
		borderStyle:          borderStyle,
	}
}

// PushColumn adds a new column to the right.
// The new column becomes focused.
// The model parameter should have Update() and View() methods (typically list.Model or multiselect.Model).
func (m *Model) PushColumn(id string, model any) {
	m.columns = append(m.columns, Column{
		ID:   id,
		List: model,
	})
	m.focusedCol = len(m.columns) - 1
	m.resizeColumns()
}

// PopColumn removes the rightmost column.
// Focus moves to the new rightmost column.
// Returns true if a column was removed, false if only one column remains.
func (m *Model) PopColumn() bool {
	if len(m.columns) <= 1 {
		return false
	}
	m.columns = m.columns[:len(m.columns)-1]
	m.focusedCol = len(m.columns) - 1
	return true
}

// ReplaceColumn replaces the column at the given index.
// The model parameter should have Update() and View() methods (typically list.Model or multiselect.Model).
func (m *Model) ReplaceColumn(index int, id string, model any) {
	if index >= 0 && index < len(m.columns) {
		m.columns[index] = Column{
			ID:   id,
			List: model,
		}
		m.resizeColumns()
	}
}

// FocusNext moves focus to the next column (to the right).
// Returns true if focus moved, false if already at the last column.
func (m *Model) FocusNext() bool {
	if m.focusedCol < len(m.columns)-1 {
		m.focusedCol++
		return true
	}
	return false
}

// FocusPrev moves focus to the previous column (to the left).
// Returns true if focus moved, false if already at the first column.
func (m *Model) FocusPrev() bool {
	if m.focusedCol > 0 {
		m.focusedCol--
		return true
	}
	return false
}

// FocusedColumn returns the currently focused column, or nil if no columns exist.
func (m *Model) FocusedColumn() *Column {
	if len(m.columns) == 0 {
		return nil
	}
	return &m.columns[m.focusedCol]
}

// Columns returns all columns.
func (m *Model) Columns() []Column {
	return m.columns
}

// FocusedIndex returns the index of the focused column.
func (m *Model) FocusedIndex() int {
	return m.focusedCol
}

// SetSize handles window resize and resizes all columns proportionally.
// Call this when you receive tea.WindowSizeMsg.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.resizeColumns()
}

// UpdateFocusedColumn updates the model in the focused column.
// This is a convenience method for updating the column model from the parent.
func (m *Model) UpdateFocusedColumn(newModel any) {
	if m.focusedCol >= 0 && m.focusedCol < len(m.columns) {
		m.columns[m.focusedCol].List = newModel
	}
}

// GetColumn returns a pointer to the column at the given index.
// This allows the parent to modify the column's model in place.
// Returns nil if index is out of bounds.
func (m *Model) GetColumn(index int) *Column {
	if index >= 0 && index < len(m.columns) {
		return &m.columns[index]
	}
	return nil
}

// View renders the Miller columns.
func (m Model) View() string {
	if len(m.columns) == 0 {
		return ""
	}

	// Determine which columns to show (last N that fit on screen)
	startCol := 0
	if len(m.columns) > m.maxVisibleColumns {
		startCol = len(m.columns) - m.maxVisibleColumns
	}

	// Render visible columns
	visibleCols := m.columns[startCol:]
	columnViews := make([]string, len(visibleCols))

	for i, col := range visibleCols {
		actualIndex := startCol + i

		// Style based on focus
		style := m.borderStyle
		if actualIndex == m.focusedCol {
			style = style.BorderForeground(m.focusedBorderColor)
		} else {
			style = style.BorderForeground(m.unfocusedBorderColor)
		}

		// Get the view from the model
		var view string
		if v, ok := col.List.(interface{ View() string }); ok {
			view = v.View()
		}

		columnViews[i] = style.Render(view)
	}

	// Join columns horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, columnViews...)
}

// resizeColumns adjusts column widths based on terminal size.
func (m *Model) resizeColumns() {
	if len(m.columns) == 0 || m.width == 0 || m.height == 0 {
		return
	}

	// Calculate visible columns
	numVisible := len(m.columns)
	if numVisible > m.maxVisibleColumns {
		numVisible = m.maxVisibleColumns
	}

	// Note: Actual resizing of column models is handled by the parent
	// via GetColumn() to access and resize each model's inner list.
	// This is necessary because we store models as `any` and can't
	// access their fields directly here.
}
