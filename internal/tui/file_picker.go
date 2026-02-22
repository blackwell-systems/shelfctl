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

type filePickerModel struct {
	list          *multiselect.Model // Pointer to allow nil for single-select mode
	currentPath   string
	quitting      bool
	selected      string
	err           error
	selectedMulti []string // Return value for multi-select
}

// filePickerKeys defines keyboard shortcuts
type filePickerKeys struct {
	quit       key.Binding
	selectItem key.Binding
	parent     key.Binding
	toggle     key.Binding
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
	parent: key.NewBinding(
		key.WithKeys("backspace", "h"),
		key.WithHelp("backspace", "parent dir"),
	),
	toggle: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
}

func (m filePickerModel) Init() tea.Cmd {
	return nil
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var currentList list.Model
	if m.list != nil {
		currentList = m.list.List
	} else {
		// Single-select mode - list is stored directly in model (legacy path)
		// This shouldn't happen with the new design, but kept for safety
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if currentList.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, fileKeys.quit):
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case key.Matches(msg, fileKeys.toggle):
			// Toggle checkbox in multi-select mode
			if m.list != nil {
				m.list.Toggle()
			}

		case key.Matches(msg, fileKeys.selectItem):
			if item, ok := currentList.SelectedItem().(*FileItem); ok {
				if item.IsDir {
					// Navigate into directory
					return m.loadDirectory(item.Path)
				}

				// In multi-select mode, collect all checked files
				if m.list != nil {
					items := currentList.Items()
					for _, listItem := range items {
						if fileItem, ok := listItem.(*FileItem); ok && fileItem.IsSelected() {
							m.selectedMulti = append(m.selectedMulti, fileItem.Path)
						}
					}
					// Fallback: if nothing checked, select current file
					if len(m.selectedMulti) == 0 {
						m.selectedMulti = []string{item.Path}
					}
					m.quitting = true
					return m, tea.Quit
				}

				// Single-select mode (shouldn't reach here with new design)
				m.selected = item.Path
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, fileKeys.parent):
			// Go up one directory
			parent := filepath.Dir(m.currentPath)
			if parent != m.currentPath {
				return m.loadDirectory(parent)
			}
		}

	case tea.WindowSizeMsg:
		h, v := StyleBorder.GetFrameSize()
		if m.list != nil {
			m.list.List.SetSize(msg.Width-h, msg.Height-v)
		}
	}

	var cmd tea.Cmd
	if m.list != nil {
		*m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m filePickerModel) View() string {
	if m.quitting {
		return ""
	}
	if m.list != nil {
		return StyleBorder.Render(m.list.View())
	}
	return ""
}

// loadDirectory loads files from the given directory
func (m filePickerModel) loadDirectory(path string) (tea.Model, tea.Cmd) {
	var currentList *list.Model
	if m.list != nil {
		currentList = &m.list.List
	}

	if currentList == nil {
		return m, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		// Don't quit on permission errors - just show error in title and allow going back
		if m.list != nil {
			m.list.SetTitle(path + " (Permission Denied - Press Backspace)")
		}

		// Add parent directory entry so user can navigate back
		parent := filepath.Dir(path)
		items := []list.Item{}
		if parent != path {
			items = append(items, &FileItem{
				Name:  "..",
				Path:  parent,
				IsDir: true,
			})
		}
		currentList.SetItems(items)
		m.currentPath = path
		return m, nil
	}

	// Build file items
	var items []list.Item

	// Add parent directory entry if not at root
	parent := filepath.Dir(path)
	if parent != path {
		items = append(items, &FileItem{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	// Add directories and files separately, then sort
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
			// selection state will be restored by multiselect.RestoreSelectionState()
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

	// Update list
	m.currentPath = path
	currentList.SetItems(items)

	// Restore selection state and update title
	if m.list != nil {
		m.list.SetTitle(path)
		m.list.RestoreSelectionState()
	}

	currentList.ResetSelected()

	return m, nil
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

	// Create base list with temporary delegate (will be replaced)
	// We use a basic render function temporarily since we need the list to create multiselect,
	// but we need multiselect to create the proper delegate
	tempDelegate := delegate.New(func(w io.Writer, m list.Model, index int, item list.Item) {
		// Temporary - will be replaced immediately
	})
	l := list.New([]list.Item{}, tempDelegate, 0, 0)
	l.Title = "Select Files"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	// Wrap with multi-select
	ms := multiselect.New(l)
	ms.SetTitle(startPath)

	// Now create the proper delegate with access to multi-select model
	d := newFileDelegate(&ms)
	ms.List.SetDelegate(d)

	// Set help keybindings for multi-select mode
	ms.List.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{fileKeys.toggle, fileKeys.parent}
	}

	m := filePickerModel{
		list:        &ms,
		currentPath: startPath,
	}

	// Load initial directory
	initialModel, _ := m.loadDirectory(startPath)
	m, _ = initialModel.(filePickerModel)

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
