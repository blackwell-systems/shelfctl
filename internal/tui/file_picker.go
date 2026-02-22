package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/blackwell-systems/shelfctl/internal/tui/multiselect"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileItem represents a file or directory in the picker.
type FileItem struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	selected bool // For multi-select mode
}

// FilterValue implements list.Item
func (f *FileItem) FilterValue() string {
	return f.Path // Use path as unique key for multi-select
}

// IsSelected implements multiselect.SelectableItem
func (f *FileItem) IsSelected() bool {
	return f.selected
}

// SetSelected implements multiselect.SelectableItem
func (f *FileItem) SetSelected(selected bool) {
	f.selected = selected
}

// IsSelectable implements multiselect.SelectableItem
// Only files are selectable, not directories
func (f *FileItem) IsSelectable() bool {
	return !f.IsDir
}

// fileDelegate renders file items with optional multi-select support
type fileDelegate struct {
	delegate.Base
	multiSelectModel *multiselect.Model
}

// newFileDelegate creates a file delegate with optional multi-select support
func newFileDelegate(ms *multiselect.Model) fileDelegate {
	return fileDelegate{
		Base: delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {
			fileItem, ok := item.(*FileItem)
			if !ok {
				return
			}

			isSelected := index == m.Index()

			// Format display
			var icon string
			var prefix string
			if fileItem.IsDir {
				icon = "📁"
				prefix = "  " // No checkbox for directories
			} else {
				icon = "📄"
				// Use multi-select checkbox if available
				if ms != nil {
					prefix = ms.CheckboxPrefix(fileItem)
				} else {
					prefix = "  "
				}
			}

			display := fmt.Sprintf("%s%s %s", prefix, icon, fileItem.Name)

			if isSelected {
				_, _ = fmt.Fprint(w, StyleHighlight.Render("› "+display))
			} else {
				_, _ = fmt.Fprint(w, "  "+StyleNormal.Render(display))
			}
		}),
		multiSelectModel: ms,
	}
}

// column represents a single directory level in Miller columns view
type column struct {
	path string
	list *multiselect.Model
}

type filePickerModel struct {
	columns       []column // Stack of directory levels
	focusedCol    int      // Which column is currently focused
	quitting      bool
	err           error
	selectedMulti []string // Return value for multi-select
	width         int
	height        int
}

// filePickerKeys defines keyboard shortcuts
type filePickerKeys struct {
	quit       key.Binding
	selectItem key.Binding
	parent     key.Binding
	toggle     key.Binding
	navRight   key.Binding
	navLeft    key.Binding
}

var fileKeys = filePickerKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "cancel"),
	),
	selectItem: key.NewBinding(
		key.WithKeys("enter", "right", "l"),
		key.WithHelp("enter", "select/open"),
	),
	parent: key.NewBinding(
		key.WithKeys("backspace", "left", "h"),
		key.WithHelp("backspace", "parent dir"),
	),
	toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	navRight: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next column"),
	),
	navLeft: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev column"),
	),
}

func (m filePickerModel) Init() tea.Cmd {
	return nil
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.columns) == 0 {
		return m, nil
	}

	focusedList := &m.columns[m.focusedCol].list.List

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if focusedList.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, fileKeys.quit):
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case key.Matches(msg, fileKeys.toggle):
			// Toggle checkbox in focused column
			m.columns[m.focusedCol].list.Toggle()
			return m, nil

		case key.Matches(msg, fileKeys.selectItem):
			if item, ok := focusedList.SelectedItem().(*FileItem); ok {
				if item.IsDir {
					// Navigate into directory - push new column
					return m.pushColumn(item.Path)
				}

				// File selected - collect all checked files across all columns
				m.selectedMulti = m.collectSelectedFiles()
				// Fallback: if nothing checked, select current file
				if len(m.selectedMulti) == 0 {
					m.selectedMulti = []string{item.Path}
				}
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, fileKeys.parent):
			// Go up one level - pop column
			if len(m.columns) > 1 {
				m.columns = m.columns[:len(m.columns)-1]
				m.focusedCol = len(m.columns) - 1
				return m, nil
			}
			// If only one column, go to parent directory
			parent := filepath.Dir(m.columns[0].path)
			if parent != m.columns[0].path {
				return m.replaceColumn(0, parent)
			}

		case key.Matches(msg, fileKeys.navRight):
			// Move focus to next column if it exists
			if m.focusedCol < len(m.columns)-1 {
				m.focusedCol++
				return m, nil
			}

		case key.Matches(msg, fileKeys.navLeft):
			// Move focus to previous column
			if m.focusedCol > 0 {
				m.focusedCol--
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeColumns()
		return m, nil
	}

	// Update focused column's list
	var cmd tea.Cmd
	*m.columns[m.focusedCol].list, cmd = m.columns[m.focusedCol].list.Update(msg)
	return m, cmd
}

func (m filePickerModel) View() string {
	if m.quitting {
		return ""
	}
	if len(m.columns) == 0 {
		return ""
	}

	// Determine which columns to show (last N that fit on screen)
	maxVisibleCols := 3
	startCol := 0
	if len(m.columns) > maxVisibleCols {
		startCol = len(m.columns) - maxVisibleCols
	}

	// Render visible columns
	visibleCols := m.columns[startCol:]
	columnViews := make([]string, len(visibleCols))

	for i, col := range visibleCols {
		actualIndex := startCol + i

		// Add visual indicator for focused column
		style := StyleBorder
		if actualIndex == m.focusedCol {
			style = style.BorderForeground(lipgloss.Color("6")) // Cyan border for focused
		} else {
			style = style.BorderForeground(lipgloss.Color("240")) // Dim border for unfocused
		}
		columnViews[i] = style.Render(col.list.View())
	}

	// Join columns horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, columnViews...)
}

// buildDirectoryItems creates list items for the given directory
func buildDirectoryItems(path string) ([]list.Item, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var items []list.Item
	var dirs []*FileItem
	var files []*FileItem

	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(path, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := &FileItem{
			Name:  name,
			Path:  fullPath,
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		}

		if item.IsDir {
			dirs = append(dirs, item)
		} else {
			// Only show document files
			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".pdf" || ext == ".epub" || ext == ".mobi" || ext == ".djvu" {
				files = append(files, item)
			}
		}
	}

	// Sort directories and files alphabetically
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// Combine: dirs first, then files
	for _, d := range dirs {
		items = append(items, d)
	}
	for _, f := range files {
		items = append(items, f)
	}

	return items, nil
}

// createColumn creates a new column for the given path
func (m *filePickerModel) createColumn(path string) (*column, error) {
	items, err := buildDirectoryItems(path)
	if err != nil {
		return nil, err
	}

	// Create base list with temporary delegate
	tempDelegate := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {})
	l := list.New(items, tempDelegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	// Wrap with multi-select
	ms := multiselect.New(l)
	ms.SetTitle(filepath.Base(path))

	// Create proper delegate with multi-select support
	d := newFileDelegate(&ms)
	ms.List.SetDelegate(d)

	// Set help keybindings
	ms.List.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{fileKeys.toggle, fileKeys.parent}
	}

	// Restore selection state
	ms.RestoreSelectionState()

	col := &column{
		path: path,
		list: &ms,
	}

	return col, nil
}

// pushColumn adds a new column for the given directory
func (m filePickerModel) pushColumn(path string) (tea.Model, tea.Cmd) {
	col, err := m.createColumn(path)
	if err != nil {
		// On error, show message but don't crash
		if len(m.columns) > 0 {
			m.columns[m.focusedCol].list.SetTitle(path + " (Permission Denied)")
		}
		return m, nil
	}

	m.columns = append(m.columns, *col)
	m.focusedCol = len(m.columns) - 1
	m.resizeColumns()

	return m, nil
}

// replaceColumn replaces the column at the given index with a new path
func (m filePickerModel) replaceColumn(index int, path string) (tea.Model, tea.Cmd) {
	col, err := m.createColumn(path)
	if err != nil {
		return m, nil
	}

	m.columns[index] = *col
	m.resizeColumns()

	return m, nil
}

// collectSelectedFiles gathers all selected files across all columns
func (m *filePickerModel) collectSelectedFiles() []string {
	var selected []string
	for _, col := range m.columns {
		items := col.list.List.Items()
		for _, item := range items {
			if fileItem, ok := item.(*FileItem); ok && fileItem.IsSelected() {
				selected = append(selected, fileItem.Path)
			}
		}
	}
	return selected
}

// resizeColumns adjusts column widths based on terminal size
func (m *filePickerModel) resizeColumns() {
	if len(m.columns) == 0 || m.width == 0 || m.height == 0 {
		return
	}

	// Calculate column width
	// Show up to 3 columns, each getting equal width
	numVisible := len(m.columns)
	if numVisible > 3 {
		numVisible = 3
	}

	h, v := StyleBorder.GetFrameSize()
	colWidth := (m.width / numVisible) - h - 1 // -1 for spacing
	colHeight := m.height - v

	// Resize each column's list
	for i := range m.columns {
		m.columns[i].list.List.SetSize(colWidth, colHeight)
	}
}

// RunFilePicker launches an interactive file browser for selecting a single file.
// Returns the selected file path, or error if canceled.
// This is a convenience wrapper around RunFilePickerMulti that returns a single file.
func RunFilePicker(startPath string) (string, error) {
	files, err := RunFilePickerMulti(startPath)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no file selected")
	}
	return files[0], nil
}

// RunFilePickerMulti launches an interactive file browser with multi-select support.
// Users can toggle checkboxes with spacebar and confirm with enter.
// Uses Miller columns (hierarchical view) for navigation.
// Returns a slice of selected file paths, or error if canceled.
func RunFilePickerMulti(startPath string) ([]string, error) {
	if startPath == "" {
		var err error
		startPath, err = os.Getwd()
		if err != nil {
			startPath = os.Getenv("HOME")
		}
	}

	// Expand ~ to home directory
	if strings.HasPrefix(startPath, "~/") {
		home := os.Getenv("HOME")
		startPath = filepath.Join(home, startPath[2:])
	}

	// Create initial model with empty columns
	m := filePickerModel{
		columns:    []column{},
		focusedCol: 0,
	}

	// Create initial column
	col, err := m.createColumn(startPath)
	if err != nil {
		return nil, fmt.Errorf("loading start path: %w", err)
	}
	m.columns = []column{*col}

	// Run the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running file picker: %w", err)
	}

	fm, ok := finalModel.(filePickerModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	if fm.err != nil {
		return nil, fm.err
	}

	return fm.selectedMulti, nil
}
