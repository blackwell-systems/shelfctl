# Miller Columns

A reusable [Bubble Tea](https://github.com/charmbracelet/bubbletea) component for hierarchical navigation using Miller columns (like macOS Finder's column view).

## Features

- **Hierarchical Navigation**: Display multiple levels side-by-side for visual context
- **Generic**: Works with any `list.Item` type and hierarchical data structure
- **Focus Management**: Navigate between columns with keyboard shortcuts
- **Responsive**: Automatically adjusts column widths to fit terminal size
- **Customizable**: Configure colors, borders, and number of visible columns
- **Zero Dependencies**: Only requires `bubbles`, `bubbletea`, and `lipgloss`

## Installation

```bash
go get github.com/blackwell-systems/shelfctl/internal/tui/millercolumns
```

## Usage

### Basic Example

```go
package main

import (
    "github.com/blackwell-systems/shelfctl/internal/tui/millercolumns"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    mc millercolumns.Model
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit

        case "enter", "right":
            // Navigate into selected item
            col := m.mc.FocusedColumn()
            if col != nil {
                selectedItem := col.List.SelectedItem()
                if isNavigable(selectedItem) {
                    // Create new list for child items
                    items := getChildItems(selectedItem)
                    newList := createListModel(items)
                    m.mc.PushColumn(getItemID(selectedItem), newList)
                }
            }

        case "left", "backspace":
            // Go back to parent
            m.mc.PopColumn()

        case "tab":
            // Switch focus to next column
            m.mc.FocusNext()

        case "shift+tab":
            // Switch focus to previous column
            m.mc.FocusPrev()
        }
    }

    // Update the miller columns model
    var cmd tea.Cmd
    m.mc, cmd = m.mc.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.mc.View()
}

func main() {
    // Create initial column
    items := getRootItems()
    initialList := createListModel(items)

    // Create miller columns model
    mc := millercolumns.New(millercolumns.Config{
        MaxVisibleColumns: 3,
    })
    mc.PushColumn("root", initialList)

    // Run the program
    p := tea.NewProgram(model{mc: mc})
    p.Run()
}
```

### File Browser Example

```go
package main

import (
    "os"
    "path/filepath"

    "github.com/blackwell-systems/shelfctl/internal/tui/millercolumns"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// FileItem represents a file or directory
type FileItem struct {
    name  string
    path  string
    isDir bool
}

func (f FileItem) Title() string       { return f.name }
func (f FileItem) Description() string { return "" }
func (f FileItem) FilterValue() string { return f.name }

type browserModel struct {
    mc millercolumns.Model
}

func (m browserModel) Init() tea.Cmd {
    return nil
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit

        case "enter", "right", "l":
            // Navigate into directory
            col := m.mc.FocusedColumn()
            if col != nil {
                if item, ok := col.List.SelectedItem().(FileItem); ok && item.isDir {
                    // Load directory contents
                    items := loadDirectory(item.path)
                    newList := list.New(items, list.NewDefaultDelegate(), 0, 0)
                    newList.Title = filepath.Base(item.path)
                    m.mc.PushColumn(item.path, newList)
                }
            }

        case "left", "backspace", "h":
            m.mc.PopColumn()

        case "tab":
            m.mc.FocusNext()

        case "shift+tab":
            m.mc.FocusPrev()
        }
    }

    var cmd tea.Cmd
    m.mc, cmd = m.mc.Update(msg)
    return m, cmd
}

func (m browserModel) View() string {
    return m.mc.View()
}

func loadDirectory(path string) []list.Item {
    entries, _ := os.ReadDir(path)
    items := make([]list.Item, 0, len(entries))
    for _, entry := range entries {
        items = append(items, FileItem{
            name:  entry.Name(),
            path:  filepath.Join(path, entry.Name()),
            isDir: entry.IsDir(),
        })
    }
    return items
}

func main() {
    // Create initial column for home directory
    home, _ := os.UserHomeDir()
    items := loadDirectory(home)

    initialList := list.New(items, list.NewDefaultDelegate(), 0, 0)
    initialList.Title = filepath.Base(home)

    // Create miller columns with custom styling
    mc := millercolumns.New(millercolumns.Config{
        MaxVisibleColumns:    3,
        FocusedBorderColor:   lipgloss.Color("6"),  // Cyan
        UnfocusedBorderColor: lipgloss.Color("240"), // Gray
    })
    mc.PushColumn(home, initialList)

    p := tea.NewProgram(browserModel{mc: mc}, tea.WithAltScreen())
    p.Run()
}
```

## API Reference

### Creating a Model

```go
mc := millercolumns.New(millercolumns.Config{
    MaxVisibleColumns:    3,                    // Show up to 3 columns
    FocusedBorderColor:   lipgloss.Color("6"),  // Cyan border for focused
    UnfocusedBorderColor: lipgloss.Color("240"), // Gray for unfocused
    BorderStyle:          myCustomBorderStyle,   // Optional custom border
})
```

### Managing Columns

```go
// Add a new column to the right
mc.PushColumn("column-id", listModel)

// Remove rightmost column
removed := mc.PopColumn() // returns false if only one column remains

// Replace a column at specific index
mc.ReplaceColumn(0, "new-id", newListModel)

// Get all columns
columns := mc.Columns()

// Get focused column
col := mc.FocusedColumn() // returns *Column or nil
```

### Focus Management

```go
// Move focus right
moved := mc.FocusNext() // returns true if focus moved

// Move focus left
moved := mc.FocusPrev() // returns true if focus moved

// Get focused column index
index := mc.FocusedIndex()
```

### Integration with Bubble Tea

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle navigation keys yourself, then:
    var cmd tea.Cmd
    m.mc, cmd = m.mc.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.mc.View()
}
```

## Key Patterns

### Navigation Logic

The parent component handles navigation decisions:

1. **Push column**: When user selects a navigable item (directory, menu, etc.)
2. **Pop column**: When user presses back/left
3. **Focus switching**: When user presses tab/shift+tab

```go
case "enter":
    col := m.mc.FocusedColumn()
    item := col.List.SelectedItem()
    if canNavigateInto(item) {
        childItems := loadChildren(item)
        newList := createList(childItems)
        m.mc.PushColumn(getID(item), newList)
    }
```

### Window Resize

The model automatically handles `tea.WindowSizeMsg` and resizes all columns proportionally. Just pass messages through:

```go
m.mc, cmd = m.mc.Update(msg)
```

### Custom Styling

```go
customBorder := lipgloss.NewStyle().
    Border(lipgloss.DoubleBorder()).
    Padding(1, 2)

mc := millercolumns.New(millercolumns.Config{
    BorderStyle:          customBorder,
    FocusedBorderColor:   lipgloss.Color("205"), // Hot pink
    UnfocusedBorderColor: lipgloss.Color("238"), // Dark gray
})
```

## Design Decisions

- **Generic**: Works with any `list.Item` - the parent decides what's navigable
- **No callbacks**: Parent handles navigation logic via key handling
- **Composition**: Each column is a standard `bubbles/list.Model`
- **Stateless navigation**: Parent maintains the navigation state, not the component
- **Responsive**: Automatically adjusts to terminal width changes

## Use Cases

- File browsers (directories and files)
- Menu systems (categories → subcategories → items)
- Configuration editors (sections → settings → values)
- Tree viewers (nodes → children → grandchildren)
- Hierarchical data exploration

## Extracting to Standalone Repository

This package has zero dependencies on `shelfctl` internals. To extract:

1. Copy `millercolumns/` directory to new repo
2. Update import paths in examples
3. Add `go.mod`:
   ```go
   module github.com/yourorg/millercolumns

   require (
       github.com/charmbracelet/bubbles v0.18.0
       github.com/charmbracelet/bubbletea v0.25.0
       github.com/charmbracelet/lipgloss v0.10.0
   )
   ```

No code changes needed!

## License

Same as parent project.
