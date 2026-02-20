package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// FileItem represents a file or directory in the picker.
type FileItem struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// FilterValue implements list.Item
func (f FileItem) FilterValue() string {
	return f.Name
}

// fileDelegate renders file items
type fileDelegate struct{}

func (d fileDelegate) Height() int  { return 1 }
func (d fileDelegate) Spacing() int { return 0 }
func (d fileDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d fileDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	fileItem, ok := item.(FileItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Format display
	icon := "  "
	if fileItem.IsDir {
		icon = "📁"
	} else {
		icon = "📄"
	}

	display := fmt.Sprintf("%s %s", icon, fileItem.Name)

	if isSelected {
		fmt.Fprint(w, StyleHighlight.Render("› "+display))
	} else {
		fmt.Fprint(w, "  "+StyleNormal.Render(display))
	}
}

type filePickerModel struct {
	list        list.Model
	currentPath string
	quitting    bool
	selected    string
	err         error
}

// filePickerKeys defines keyboard shortcuts
type filePickerKeys struct {
	quit   key.Binding
	select_ key.Binding
	parent key.Binding
}

var fileKeys = filePickerKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "cancel"),
	),
	select_: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/open"),
	),
	parent: key.NewBinding(
		key.WithKeys("backspace", "h"),
		key.WithHelp("backspace", "parent dir"),
	),
}

func (m filePickerModel) Init() tea.Cmd {
	return nil
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, fileKeys.quit):
			m.quitting = true
			m.err = fmt.Errorf("canceled by user")
			return m, tea.Quit

		case key.Matches(msg, fileKeys.select_):
			if item, ok := m.list.SelectedItem().(FileItem); ok {
				if item.IsDir {
					// Navigate into directory
					return m.loadDirectory(item.Path)
				}
				// Select file
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
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m filePickerModel) View() string {
	if m.quitting {
		return ""
	}
	return StyleBorder.Render(m.list.View())
}

// loadDirectory loads files from the given directory
func (m filePickerModel) loadDirectory(path string) (tea.Model, tea.Cmd) {
	entries, err := os.ReadDir(path)
	if err != nil {
		m.err = fmt.Errorf("reading directory: %w", err)
		m.quitting = true
		return m, tea.Quit
	}

	// Build file items
	var items []list.Item

	// Add parent directory entry if not at root
	parent := filepath.Dir(path)
	if parent != path {
		items = append(items, FileItem{
			Name:  "..",
			Path:  parent,
			IsDir: true,
		})
	}

	// Add directories and files separately, then sort
	var dirs []FileItem
	var files []FileItem

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

		item := FileItem{
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

	// Update list
	m.currentPath = path
	m.list.SetItems(items)
	m.list.Title = "Select File: " + path
	m.list.ResetSelected()

	return m, nil
}

// RunFilePicker launches an interactive file browser.
// Returns the selected file path, or error if canceled.
func RunFilePicker(startPath string) (string, error) {
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

	// Create list
	delegate := fileDelegate{}
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Select File"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = StyleHeader
	l.Styles.HelpStyle = StyleHelp

	// Set help keybindings
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{fileKeys.parent}
	}

	m := filePickerModel{
		list:        l,
		currentPath: startPath,
	}

	// Load initial directory
	initialModel, _ := m.loadDirectory(startPath)
	m = initialModel.(filePickerModel)

	// Run the program
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running file picker: %w", err)
	}

	fm, ok := finalModel.(filePickerModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if fm.err != nil {
		return "", fm.err
	}

	return fm.selected, nil
}
