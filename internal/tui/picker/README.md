# Base Picker Component

A reusable foundation for building Bubble Tea picker components. Reduces boilerplate and provides consistent behavior across all pickers.

## Features

- Standard key handling (quit, select)
- Window resize handling
- Border rendering
- Error handling
- Custom key bindings and handlers
- Extensible for specific picker needs

## Usage

### Basic Picker

```go
package main

import (
    "fmt"
    "github.com/blackwell-systems/shelfctl/internal/tui"
    "github.com/blackwell-systems/shelfctl/internal/tui/picker"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type MyItem struct {
    name string
}

func (m MyItem) FilterValue() string { return m.name }
func (m MyItem) Title() string       { return m.name }
func (m MyItem) Description() string { return "" }

type myPickerModel struct {
    base     *picker.Base
    selected *MyItem
}

func newMyPicker(items []list.Item) myPickerModel {
    keys := tui.NewPickerKeys()
    l := list.New(items, list.NewDefaultDelegate(), 80, 20)
    l.Title = "Select an item"
    l.SetShowStatusBar(false)

    base := picker.New(picker.Config{
        List:       l,
        QuitKeys:   keys.Quit,
        SelectKeys: keys.Select,
        ShowBorder: true,
        BorderStyle: tui.StyleBorder,
        OnSelect: func(item list.Item) bool {
            // Return true to quit after selection
            return true
        },
    })

    return myPickerModel{base: base}
}

func (m myPickerModel) Init() tea.Cmd {
    return nil
}

func (m myPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmd := m.base.Update(msg)

    // Check if selection was made
    if m.base.IsQuitting() && m.base.Error() == nil {
        if item, ok := m.base.SelectedItem().(MyItem); ok {
            m.selected = &item
        }
    }

    return m, cmd
}

func (m myPickerModel) View() string {
    return m.base.View()
}

func main() {
    items := []list.Item{
        MyItem{name: "Option 1"},
        MyItem{name: "Option 2"},
        MyItem{name: "Option 3"},
    }

    p := tea.NewProgram(newMyPicker(items))
    if _, err := p.Run(); err != nil {
        panic(err)
    }
}
```

### Picker with Custom Keys

```go
type advancedPickerModel struct {
    base *picker.Base
}

func newAdvancedPicker(items []list.Item) advancedPickerModel {
    keys := tui.NewNavigablePickerKeys()
    l := list.New(items, list.NewDefaultDelegate(), 80, 20)

    base := picker.New(picker.Config{
        List:       l,
        QuitKeys:   keys.Quit,
        SelectKeys: keys.Select,
        ShowBorder: true,
        BorderStyle: tui.StyleBorder,

        // Custom key handler
        OnKeyPress: func(msg tea.KeyMsg) (bool, tea.Cmd) {
            if key.Matches(msg, keys.Back) {
                // Handle back navigation
                fmt.Println("Going back...")
                return true, tea.Quit
            }
            return false, nil
        },

        // Custom select handler
        OnSelect: func(item list.Item) bool {
            myItem := item.(MyItem)
            fmt.Printf("Selected: %s\n", myItem.name)
            return true // Quit after selection
        },
    })

    return advancedPickerModel{base: base}
}
```

### Picker with Custom Window Sizing

```go
base := picker.New(picker.Config{
    List:       l,
    QuitKeys:   keys.Quit,
    SelectKeys: keys.Select,
    ShowBorder: true,
    BorderStyle: tui.StyleBorder,

    OnWindowSize: func(width, height int) {
        // Custom sizing logic
        headerHeight := 3
        footerHeight := 2
        availableHeight := height - headerHeight - footerHeight

        h, v := tui.StyleBorder.GetFrameSize()
        l.SetSize(width-h, availableHeight-v)
    },

    OnSelect: func(item list.Item) bool {
        return true
    },
})
```

## API Reference

### Config

```go
type Config struct {
    // List is the underlying bubbles list.Model (required)
    List list.Model

    // Keys are the key bindings (required)
    QuitKeys   key.Binding
    SelectKeys key.Binding

    // Handlers
    OnSelect     SelectHandler // Called when SelectKeys is pressed
    OnKeyPress   KeyHandler    // Optional: custom key handling
    OnWindowSize func(width, height int) // Optional: custom window sizing

    // Styling
    BorderStyle lipgloss.Style
    ShowBorder  bool
}
```

### SelectHandler

```go
type SelectHandler func(selectedItem list.Item) bool
```

Return `true` to quit the picker after selection, `false` to continue.

### KeyHandler

```go
type KeyHandler func(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
```

Return `true` if the key was handled (prevents default handling), `false` to pass through.

### Base Methods

- `List() *list.Model` - Access the underlying list
- `IsQuitting() bool` - Check if picker is quitting
- `Error() error` - Get any error
- `SetError(err error)` - Set error and quit
- `Quit()` - Quit without error
- `Update(msg tea.Msg) tea.Cmd` - Handle updates
- `View() string` - Render the picker
- `SelectedItem() list.Item` - Get selected item
- `Items() []list.Item` - Get all items
- `SetItems(items []list.Item)` - Set items
- `Title() string` - Get title
- `SetTitle(title string)` - Set title

## Standard Keys

The package provides several pre-configured key binding sets:

### PickerKeys

Basic picker keys (quit, select):

```go
keys := tui.NewPickerKeys()
// keys.Quit:   q, esc, ctrl+c
// keys.Select: enter
```

### NavigablePickerKeys

For pickers with navigation (quit, select, back):

```go
keys := tui.NewNavigablePickerKeys()
// keys.Quit:   q, esc, ctrl+c
// keys.Select: enter
// keys.Back:   backspace, h
```

### MultiSelectPickerKeys

For pickers with multi-selection (quit, select, toggle, back):

```go
keys := tui.NewMultiSelectPickerKeys()
// keys.Quit:   q, esc, ctrl+c
// keys.Select: enter
// keys.Toggle: space
// keys.Back:   backspace, h
```

### FormKeys

For form components (quit, submit, next, prev):

```go
keys := tui.NewFormKeys()
// keys.Quit:   q, esc, ctrl+c
// keys.Submit: enter
// keys.Next:   tab, down
// keys.Prev:   shift+tab, up
```

## Integration with Existing Code

To migrate an existing picker to use the base:

1. Replace individual fields with `*picker.Base`
2. Move key bindings to use standard keys
3. Move common Update logic into Config handlers
4. Call `base.Update()` and `base.View()`

### Before

```go
type myPickerModel struct {
    list     list.Model
    quitting bool
    selected string
    err      error
}

func (m myPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.list.FilterState() == list.Filtering {
            break
        }
        switch msg.String() {
        case "q", "esc", "ctrl+c":
            m.quitting = true
            m.err = fmt.Errorf("canceled")
            return m, tea.Quit
        case "enter":
            item := m.list.SelectedItem().(MyItem)
            m.selected = item.name
            m.quitting = true
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        h, v := StyleBorder.GetFrameSize()
        m.list.SetSize(msg.Width-h, msg.Height-v)
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}
```

### After

```go
type myPickerModel struct {
    base     *picker.Base
    selected string
}

func (m myPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmd := m.base.Update(msg)

    if m.base.IsQuitting() && m.base.Error() == nil {
        item := m.base.SelectedItem().(MyItem)
        m.selected = item.name
    }

    return m, cmd
}
```

## Design Philosophy

The base picker follows these principles:

1. **Composition over inheritance** - Embed Base, don't extend it
2. **Sensible defaults** - Standard behavior works out of the box
3. **Easy customization** - Override handlers for custom behavior
4. **No magic** - Clear, explicit configuration
5. **Type safety** - Strong typing with interfaces where needed

## License

See project root for license information.
