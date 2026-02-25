# TUI Architecture Guide

This document explains the architectural patterns used in the shelfctl TUI components and how to use the base components effectively.

## Core Components

### 1. Standard Keys (`keys.go`)

Provides pre-configured key binding sets for common TUI patterns:

- **PickerKeys** - Basic selection (quit, select)
- **NavigablePickerKeys** - Selection with navigation (quit, select, back)
- **MultiSelectPickerKeys** - Multi-selection (quit, select, toggle, back)
- **FormKeys** - Form input (quit, submit, next, prev)

**Benefits:**
- Consistent key bindings across all components
- Easy to extend with additional keys
- Built-in help text

### 2. Base Picker (`picker/base.go`)

Generic picker foundation that handles:
- Standard key bindings (quit, select)
- Window resize handling
- Border rendering
- Error handling
- Custom behavior via handlers

**Benefits:**
- Reduces boilerplate by 60-70%
- Consistent behavior across pickers
- Easy to customize via configuration
- Testable in isolation

### 3. Multi-Select (`multiselect/multiselect.go`)

Reusable multi-selection wrapper for any list:
- Checkbox UI with state persistence
- Works with any `list.Item` implementation
- Customizable appearance

**Benefits:**
- Reusable across multiple pickers
- Can be used in other projects
- Well-documented and tested

## Component Hierarchy

```
TUI Components
├── Base Components (reusable)
│   ├── keys.go - Standard key bindings
│   ├── picker/base.go - Base picker
│   └── multiselect/multiselect.go - Multi-select wrapper
│
├── Styling (reusable)
│   └── common.go - Colors and styles
│
├── Pickers (specific implementations)
│   ├── shelf_picker.go - Select shelf
│   ├── book_picker.go - Select book
│   └── file_picker.go - Select file(s)
│
└── Other Components
    ├── shelve_form.go - Book metadata form
    ├── edit_form.go - Edit book form
    ├── hub.go - Main menu
    └── progress.go - Upload progress
```

## Installation

The base components have been extracted to an external package:

```bash
go get github.com/blackwell-systems/bubbletea-picker
go get github.com/blackwell-systems/bubbletea-multiselect
go get github.com/blackwell-systems/bubbletea-millercolumns
```

Import the components you need:

```go
import (
    "github.com/blackwell-systems/bubbletea-picker"
    "github.com/blackwell-systems/bubbletea-multiselect"
    "github.com/blackwell-systems/bubbletea-millercolumns"
)
```

## Migration Guide

### Refactoring an Existing Picker

#### Before: Traditional Implementation

```go
type myPickerModel struct {
    list     list.Model
    quitting bool
    selected string
    err      error
}

type myKeys struct {
    quit   key.Binding
    select key.Binding
}

var keys = myKeys{
    quit: key.NewBinding(
        key.WithKeys("q", "esc", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
    select: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "select"),
    ),
}

func (m myPickerModel) Init() tea.Cmd {
    return nil
}

func (m myPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.list.FilterState() == list.Filtering {
            break
        }

        switch {
        case key.Matches(msg, keys.quit):
            m.quitting = true
            m.err = fmt.Errorf("canceled by user")
            return m, tea.Quit

        case key.Matches(msg, keys.select):
            if item, ok := m.list.SelectedItem().(MyItem); ok {
                m.selected = item.name
                m.quitting = true
                return m, tea.Quit
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

func (m myPickerModel) View() string {
    if m.quitting {
        return ""
    }
    return StyleBorder.Render(m.list.View())
}
```

**Lines of code:** ~65
**Boilerplate:** High
**Testability:** Moderate

#### After: Using Base Picker

```go
type myPickerModel struct {
    base     *picker.Base
    selected string
}

func (m myPickerModel) Init() tea.Cmd {
    return nil
}

func (m myPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmd := m.base.Update(msg)

    if m.base.IsQuitting() && m.base.Error() == nil {
        if item, ok := m.base.SelectedItem().(MyItem); ok {
            m.selected = item.name
        }
    }

    return m, cmd
}

func (m myPickerModel) View() string {
    return m.base.View()
}

func newMyPicker(items []list.Item) myPickerModel {
    keys := tui.NewPickerKeys()
    l := list.New(items, myDelegate{}, 80, 20)
    l.Title = "Select Item"

    base := picker.New(picker.Config{
        List:        l,
        QuitKeys:    keys.Quit,
        SelectKeys:  keys.Select,
        ShowBorder:  true,
        BorderStyle: tui.StyleBorder,
        OnSelect: func(item list.Item) bool {
            return true // Quit after selection
        },
    })

    return myPickerModel{base: base}
}
```

**Lines of code:** ~40
**Boilerplate:** Low
**Testability:** High (can test base separately)

**Savings:** 38% less code, 70% less boilerplate

### Step-by-Step Migration

1. **Import the new packages**
   ```go
   import (
       "github.com/blackwell-systems/bubbletea-picker"
       "github.com/blackwell-systems/shelfctl/internal/tui"
   )
   ```

2. **Replace model fields**
   ```go
   // Before
   type myModel struct {
       list     list.Model
       quitting bool
       err      error
       selected string
   }

   // After
   type myModel struct {
       base     *picker.Base
       selected string
   }
   ```

3. **Use standard keys**
   ```go
   // Before
   var keys = myKeys{
       quit: key.NewBinding(...),
       select: key.NewBinding(...),
   }

   // After
   keys := tui.NewPickerKeys()
   ```

4. **Create base picker in constructor**
   ```go
   base := picker.New(picker.Config{
       List:        l,
       QuitKeys:    keys.Quit,
       SelectKeys:  keys.Select,
       ShowBorder:  true,
       BorderStyle: tui.StyleBorder,
       OnSelect: func(item list.Item) bool {
           // Handle selection
           return true // Quit after selection
       },
   })
   ```

5. **Simplify Update method**
   ```go
   func (m myModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       cmd := m.base.Update(msg)

       // Only handle picker-specific logic
       if m.base.IsQuitting() && m.base.Error() == nil {
           // Extract selection
       }

       return m, cmd
   }
   ```

6. **Simplify View method**
   ```go
   func (m myModel) View() string {
       return m.base.View()
   }
   ```

## Adding Custom Behavior

### Custom Key Handling

```go
base := picker.New(picker.Config{
    // ... standard config ...
    OnKeyPress: func(msg tea.KeyMsg) (bool, tea.Cmd) {
        switch msg.String() {
        case "d":
            // Custom action
            return true, someCmd // Handled
        case "i":
            // Another custom action
            return true, nil
        }
        return false, nil // Not handled, use default
    },
})
```

### Navigation Logic

```go
keys := tui.NewNavigablePickerKeys()

base := picker.New(picker.Config{
    List:       l,
    QuitKeys:   keys.Quit,
    SelectKeys: keys.Select,
    ShowBorder: true,
    BorderStyle: tui.StyleBorder,

    OnKeyPress: func(msg tea.KeyMsg) (bool, tea.Cmd) {
        if key.Matches(msg, keys.Back) {
            // Navigate back (e.g., to parent directory)
            return true, navigateBackCmd()
        }
        return false, nil
    },

    OnSelect: func(item list.Item) bool {
        myItem := item.(MyItem)
        if myItem.IsDirectory {
            // Navigate into directory, don't quit
            navigateInto(myItem)
            return false
        }
        // File selected, quit
        return true
    },
})
```

### Custom Window Sizing

```go
base := picker.New(picker.Config{
    // ... standard config ...
    OnWindowSize: func(width, height int) {
        // Custom sizing with header/footer
        headerHeight := 3
        footerHeight := 2
        availableHeight := height - headerHeight - footerHeight

        h, v := tui.StyleBorder.GetFrameSize()
        list.SetSize(width-h, availableHeight-v)
    },
})
```

## Best Practices

### 1. Always Use Standard Keys

```go
// Good
keys := tui.NewPickerKeys()

// Avoid
customKeys := myCustomKeys{...}
```

### 2. Keep Selection Logic in OnSelect

```go
// Good
OnSelect: func(item list.Item) bool {
    myItem := item.(MyItem)
    if myItem.IsNavigable {
        navigate(myItem)
        return false
    }
    return true
}

// Avoid checking in Update()
```

### 3. Use OnKeyPress for Custom Keys Only

```go
// Good - only custom keys
OnKeyPress: func(msg tea.KeyMsg) (bool, tea.Cmd) {
    switch msg.String() {
    case "d":
        deleteItem()
        return true, nil
    }
    return false, nil
}

// Avoid - reimplementing standard keys
OnKeyPress: func(msg tea.KeyMsg) (bool, tea.Cmd) {
    switch msg.String() {
    case "q":  // Don't do this
        return true, tea.Quit
    }
    return false, nil
}
```

### 4. Extract Selection in Update, Not OnSelect

```go
// Good
OnSelect: func(item list.Item) bool {
    return true
}

func (m myModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    cmd := m.base.Update(msg)
    if m.base.IsQuitting() && m.base.Error() == nil {
        m.selected = m.base.SelectedItem().(MyItem)
    }
    return m, cmd
}

// Avoid - side effects in OnSelect
OnSelect: func(item list.Item) bool {
    m.selected = item.(MyItem)  // Don't do this
    return true
}
```

## Testing

Base picker enables better testing:

```go
func TestMyPicker(t *testing.T) {
    items := []list.Item{
        MyItem{name: "test"},
    }

    keys := tui.NewPickerKeys()
    l := list.New(items, myDelegate{}, 80, 20)

    var selected list.Item
    base := picker.New(picker.Config{
        List:       l,
        QuitKeys:   keys.Quit,
        SelectKeys: keys.Select,
        OnSelect: func(item list.Item) bool {
            selected = item
            return true
        },
    })

    // Test key handling
    base.Update(tea.KeyMsg{Type: tea.KeyEnter})

    // Verify selection
    if selected == nil {
        t.Error("Expected item to be selected")
    }
}
```

## Future Enhancements

Potential future additions:

1. **Base Form Component** - Similar to base picker, for form inputs
2. **Navigation Manager** - Handle screen transitions and history
3. **Validation Framework** - Reusable form validators
4. **Base Delegate** - Reduce delegate boilerplate

## Questions?

See the README files in each component directory:
- `picker/README.md` - Base picker documentation
- `multiselect/README.md` - Multi-select documentation

Or check the example implementations:
- `shelf_picker_v2.go` - Refactored shelf picker example
