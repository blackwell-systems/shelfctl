# Multi-Select Component for Bubble Tea

A reusable multi-select wrapper for [Bubble Tea](https://github.com/charmbracelet/bubbletea) list components. Adds checkbox-style selection with persistent state across view changes.

## Features

- Checkbox UI for selectable items (`[ ]` / `[✓]`)
- Spacebar to toggle selection
- Selection state persists across list updates (e.g., directory navigation)
- Customizable checkbox appearance
- Selection count in title
- Works with any `list.Item` implementation

## Installation

```go
import "github.com/blackwell-systems/shelfctl/internal/tui/multiselect"
```

## Usage

### 1. Implement the SelectableItem Interface

Your list items must implement the `multiselect.SelectableItem` interface:

```go
type MyItem struct {
    name     string
    selected bool
}

// FilterValue implements list.Item
func (m *MyItem) FilterValue() string {
    return m.name
}

// IsSelected implements multiselect.SelectableItem
func (m *MyItem) IsSelected() bool {
    return m.selected
}

// SetSelected implements multiselect.SelectableItem
func (m *MyItem) SetSelected(selected bool) {
    m.selected = selected
}

// IsSelectable implements multiselect.SelectableItem
// Return false for items that shouldn't have checkboxes
func (m *MyItem) IsSelectable() bool {
    return true
}
```

**Important:** Use pointer receivers for methods so the multiselect component can mutate items.

### 2. Create the Multi-Select Model

```go
// Create a standard bubbles list
baseList := list.New(items, myDelegate{}, 80, 24)
baseList.Title = "Select Items"

// Wrap it with multi-select
ms := multiselect.New(baseList)
ms.SetTitle("My Items") // Base title (count will be appended)
```

### 3. Update Your Delegate for Checkbox Rendering

Your delegate needs access to the multiselect model to render checkboxes:

```go
type myDelegate struct {
    multiSelectModel *multiselect.Model
}

func (d myDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    myItem := item.(*MyItem)

    // Get checkbox prefix from multiselect
    prefix := ""
    if d.multiSelectModel != nil {
        prefix = d.multiSelectModel.CheckboxPrefix(myItem)
    }

    // Render item with checkbox
    fmt.Fprintf(w, "%s%s", prefix, myItem.name)
}
```

### 4. Handle Spacebar in Your Update Function

```go
func (m myModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case " ": // spacebar
            m.multiSelect.Toggle()
            return m, nil
        }
    }

    // Update the multiselect model
    var cmd tea.Cmd
    m.multiSelect, cmd = m.multiSelect.Update(msg)
    return m, cmd
}
```

### 5. Get Selected Items

```go
// Get all selected keys (uses FilterValue as unique key)
selectedKeys := ms.SelectedKeys()

// Or iterate through items
for _, item := range ms.List.Items() {
    if selectableItem, ok := item.(*MyItem); ok && selectableItem.IsSelected() {
        // Process selected item
    }
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/blackwell-systems/shelfctl/internal/tui/multiselect"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type item struct {
    title    string
    selected bool
}

func (i *item) FilterValue() string { return i.title }
func (i *item) IsSelected() bool    { return i.selected }
func (i *item) SetSelected(s bool)  { i.selected = s }
func (i *item) IsSelectable() bool  { return true }

type itemDelegate struct {
    ms *multiselect.Model
}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
    i := listItem.(*item)
    prefix := "  "
    if d.ms != nil {
        prefix = d.ms.CheckboxPrefix(i)
    }
    fmt.Fprintf(w, "%s%s", prefix, i.title)
}

type model struct {
    multiSelect multiselect.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        case " ":
            m.multiSelect.Toggle()
            return m, nil
        case "enter":
            // Print selected items
            for _, key := range m.multiSelect.SelectedKeys() {
                fmt.Println("Selected:", key)
            }
            return m, tea.Quit
        }
    }

    var cmd tea.Cmd
    m.multiSelect, cmd = m.multiSelect.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.multiSelect.View()
}

func main() {
    items := []list.Item{
        &item{title: "Item 1"},
        &item{title: "Item 2"},
        &item{title: "Item 3"},
    }

    l := list.New(items, itemDelegate{}, 80, 10)
    ms := multiselect.New(l)
    ms.SetTitle("Select Items")

    // Update delegate with multiselect reference
    delegate := itemDelegate{ms: &ms}
    ms.List.SetDelegate(delegate)

    p := tea.NewProgram(model{multiSelect: ms})
    if _, err := p.Run(); err != nil {
        panic(err)
    }
}
```

## API Reference

### Model Methods

- `New(l list.Model) Model` - Create a new multi-select model
- `Toggle() bool` - Toggle selection of current item, returns true if toggled
- `Select(key string)` - Select an item by its FilterValue key
- `Deselect(key string)` - Deselect an item
- `ClearSelection()` - Clear all selections
- `SelectedKeys() []string` - Get all selected item keys
- `SelectedCount() int` - Get number of selected items
- `SetTitle(title string)` - Set base title (without count)
- `SetCheckboxStyle(checked, empty string)` - Customize checkbox appearance
- `SetShowCount(show bool)` - Toggle selection count in title
- `RestoreSelectionState()` - Restore selection after items change
- `CheckboxPrefix(item SelectableItem) string` - Get checkbox for delegate rendering

### SelectableItem Interface

```go
type SelectableItem interface {
    list.Item                    // Must implement list.Item
    IsSelected() bool            // Current selection state
    SetSelected(bool)           // Update selection state
    IsSelectable() bool         // Whether item can be selected
}
```

## Design Notes

- **Pointer receivers required:** Items must use pointer receivers for the interface methods so the multiselect component can mutate them
- **FilterValue as key:** The item's `FilterValue()` is used as a unique key for tracking selection across list updates
- **State persistence:** The component maintains a map of selected keys, which persists even when the list items are replaced (useful for directory navigation)
- **Flexibility:** Only items with `IsSelectable() == true` show checkboxes and can be toggled

## License

See project root for license information.
