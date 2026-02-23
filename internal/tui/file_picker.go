package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/blackwell-systems/shelfctl/internal/tui/millercolumns"
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
	// Use name for filtering (easier to search), but include path for uniqueness
	return f.Name + " " + f.Path
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

type filePickerModel struct {
	mc            millercolumns.Model
	quitting      bool
	err           error
	selectedMulti []string // Return value for multi-select
	width         int
	height        int
	showHidden    bool // Whether to show hidden files/directories
}

// filePickerKeys defines keyboard shortcuts
type filePickerKeys struct {
	quit         key.Binding
	selectItem   key.Binding
	openDir      key.Binding
	parent       key.Binding
	toggle       key.Binding
	toggleHidden key.Binding
	navRight     key.Binding
	navLeft      key.Binding
}

var fileKeys = filePickerKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "cancel"),
	),
	selectItem: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/open"),
	),
	openDir: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("right", "open dir"),
	),
	parent: key.NewBinding(
		key.WithKeys("backspace", "left", "h"),
		key.WithHelp("backspace", "parent dir"),
	),
	toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	toggleHidden: key.NewBinding(
		key.WithKeys("."),
		key.WithHelp(".", "show hidden"),
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
	col := m.mc.FocusedColumn()
	if col == nil {
		return m, nil
	}

	// Get the multiselect model from the column
	ms, ok := col.List.(*multiselect.Model)
	if !ok {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store dimensions and handle resize
		m.width = msg.Width
		m.height = msg.Height
		m.mc.SetSize(msg.Width, msg.Height)
		m.resizeAllColumns()
		return m, nil

	case tea.KeyMsg:
		// Don't handle keys when filtering
		if ms.List.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, fileKeys.quit):
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case key.Matches(msg, fileKeys.toggle):
			// Toggle checkbox in focused column
			ms.Toggle()
			return m, nil

		case key.Matches(msg, fileKeys.toggleHidden):
			// Toggle hidden files/directories visibility
			m.showHidden = !m.showHidden
			// Rebuild all visible columns with new showHidden setting
			return m.rebuildAllColumns()

		case key.Matches(msg, fileKeys.openDir):
			// Right arrow / 'l' - only navigates into directories
			if item, ok := ms.List.SelectedItem().(*FileItem); ok {
				if item.IsDir {
					// Navigate into directory - push new column
					return m.pushColumn(item.Path)
				}
				// Ignore if not a directory
			}
			return m, nil

		case key.Matches(msg, fileKeys.selectItem):
			// Enter - selects files or opens directories
			if item, ok := ms.List.SelectedItem().(*FileItem); ok {
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
			if !m.mc.PopColumn() {
				// If only one column remains, navigate to parent directory
				col := m.mc.FocusedColumn()
				if col != nil {
					parent := filepath.Dir(col.ID)
					if parent != col.ID {
						return m.replaceColumn(0, parent)
					}
				}
			}
			return m, nil

		case key.Matches(msg, fileKeys.navRight):
			m.mc.FocusNext()
			return m, nil

		case key.Matches(msg, fileKeys.navLeft):
			m.mc.FocusPrev()
			return m, nil
		}
	}

	// Update the focused column's multiselect model
	var cmd tea.Cmd
	updatedMs, msCmd := ms.Update(msg)
	m.mc.UpdateFocusedColumn(&updatedMs)
	cmd = msCmd

	return m, cmd
}

func (m filePickerModel) View() string {
	if m.quitting {
		return ""
	}
	return m.mc.View()
}

// buildDirectoryItems creates list items for the given directory
func buildDirectoryItems(path string, showHidden bool) ([]list.Item, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var items []list.Item
	var dirs []*FileItem
	var files []*FileItem

	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files unless showHidden is true
		if !showHidden && strings.HasPrefix(name, ".") {
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

// createListForPath creates a multiselect list model for the given path
func createListForPath(path string, showHidden bool) (*multiselect.Model, error) {
	items, err := buildDirectoryItems(path, showHidden)
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
		return []key.Binding{fileKeys.toggle, fileKeys.toggleHidden, fileKeys.parent}
	}

	// Restore selection state
	ms.RestoreSelectionState()

	return &ms, nil
}

// pushColumn adds a new column for the given directory
func (m filePickerModel) pushColumn(path string) (tea.Model, tea.Cmd) {
	listModel, err := createListForPath(path, m.showHidden)
	if err != nil {
		// On error, show message in current column but don't crash
		col := m.mc.FocusedColumn()
		if col != nil {
			if ms, ok := col.List.(*multiselect.Model); ok {
				ms.SetTitle(path + " (Permission Denied)")
			}
		}
		return m, nil
	}

	m.mc.PushColumn(path, listModel)

	// Resize all columns after pushing
	m.resizeAllColumns()

	return m, nil
}

// replaceColumn replaces the column at the given index with a new path
func (m filePickerModel) replaceColumn(index int, path string) (tea.Model, tea.Cmd) {
	listModel, err := createListForPath(path, m.showHidden)
	if err != nil {
		return m, nil
	}

	m.mc.ReplaceColumn(index, path, listModel)
	m.resizeAllColumns()

	return m, nil
}

// rebuildAllColumns rebuilds all visible columns with current showHidden setting
func (m filePickerModel) rebuildAllColumns() (tea.Model, tea.Cmd) {
	// Get all current column paths
	columns := m.mc.Columns()
	if len(columns) == 0 {
		return m, nil
	}

	// Store the focused column index and selected item index
	focusedIdx := m.mc.FocusedIndex()

	// Get the selected index in the focused column before rebuilding
	var selectedIdx int
	if focusedCol := m.mc.FocusedColumn(); focusedCol != nil {
		if ms, ok := focusedCol.List.(*multiselect.Model); ok {
			selectedIdx = ms.List.Index()
		}
	}

	// Rebuild each column
	for i, col := range columns {
		listModel, err := createListForPath(col.ID, m.showHidden)
		if err != nil {
			continue
		}

		// Restore selection state for this column
		m.mc.ReplaceColumn(i, col.ID, listModel)
	}

	// Restore focus to the same column index
	// FocusedIndex returns 0-based index, so we need to focus that many times from start
	for i := 0; i < focusedIdx; i++ {
		m.mc.FocusNext()
	}

	// Restore selected index in focused column
	if focusedCol := m.mc.FocusedColumn(); focusedCol != nil {
		if ms, ok := focusedCol.List.(*multiselect.Model); ok {
			// Try to restore the same index, but don't exceed new list length
			if selectedIdx < len(ms.List.Items()) {
				ms.List.Select(selectedIdx)
			}
		}
	}

	m.resizeAllColumns()

	return m, nil
}

// collectSelectedFiles gathers all selected files across all columns
func (m *filePickerModel) collectSelectedFiles() []string {
	var selected []string
	for _, col := range m.mc.Columns() {
		if ms, ok := col.List.(*multiselect.Model); ok {
			items := ms.List.Items()
			for _, item := range items {
				if fileItem, ok := item.(*FileItem); ok && fileItem.IsSelected() {
					selected = append(selected, fileItem.Path)
				}
			}
		}
	}
	return selected
}

// resizeAllColumns resizes all multiselect column lists to fit the terminal.
// Should be called after pushing/replacing columns or on window resize.
func (m *filePickerModel) resizeAllColumns() {
	if m.width == 0 || m.height == 0 {
		return // No size set yet, wait for WindowSizeMsg
	}

	numVisible := len(m.mc.Columns())
	if numVisible > 3 {
		numVisible = 3
	}
	if numVisible == 0 {
		return
	}

	h, v := StyleBorder.GetFrameSize()
	colWidth := (m.width / numVisible) - h - 1
	colHeight := m.height - v

	// Resize all multiselect models
	for i := 0; i < len(m.mc.Columns()); i++ {
		col := m.mc.GetColumn(i)
		if col != nil {
			if ms, ok := col.List.(*multiselect.Model); ok {
				ms.List.SetSize(colWidth, colHeight)
			}
		}
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

	// Create file picker model first (needed for showHidden flag)
	m := filePickerModel{
		showHidden: false, // Hidden files are hidden by default
	}

	// Create initial list for start path
	initialList, err := createListForPath(startPath, m.showHidden)
	if err != nil {
		return nil, fmt.Errorf("loading start path: %w", err)
	}

	// Create miller columns model with custom styling
	mc := millercolumns.New(millercolumns.Config{
		MaxVisibleColumns:    3,
		FocusedBorderColor:   lipgloss.Color("6"),   // Cyan
		UnfocusedBorderColor: lipgloss.Color("240"), // Gray
		BorderStyle:          StyleBorder,
	})
	mc.PushColumn(startPath, initialList)

	// Assign miller columns to model
	m.mc = mc

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
