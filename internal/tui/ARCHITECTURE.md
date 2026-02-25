# TUI Architecture

## Unified Single-Program Design

The TUI runs as a single persistent Bubble Tea program for the entire session. The orchestrator in `internal/unified/model.go` coordinates all views, routes messages, and handles view switching without screen flicker.

```
Orchestrator (model.go)
├── Hub          # Main menu with command palette
├── Browse       # Book browser with multi-select and actions
├── Shelve       # File picker → metadata form → upload
├── Edit         # Metadata editor (carousel for batch)
├── Move         # Destination selector
├── Delete       # Confirmation dialog
├── CacheClear   # Multi-select cache picker
└── CreateShelf  # Shelf creation form
```

## Message System

All view transitions use typed messages defined in `internal/unified/messages.go`:

| Message | Purpose |
|---------|---------|
| `NavigateMsg` | Switch between views |
| `QuitAppMsg` | Exit the application |
| `ActionRequestMsg` | Operations needing terminal (open, import) |
| `CommandRequestMsg` | Non-TUI commands (shelves, index, cache-info) |

### Suspend-Resume Pattern

Operations that need normal terminal output (e.g., `open` launching a PDF viewer):

```
TUI (alt screen) → ActionRequestMsg
  → Suspend (exit alt screen)
  → Run command (normal terminal output)
  → Resume (re-enter alt screen, return to same or different view)
```

### pendingNavMsg Pattern

The command palette returns navigation results synchronously to avoid a one-frame flash through the hub:

```go
// hub.go: store result instead of returning as Cmd
case commandpalette.ActionSelectedMsg:
    m.pendingNavMsg = msg.Action.Run()
    return m, nil

// model.go: orchestrator checks inline in same Update cycle
if m.hub.pendingNavMsg != nil {
    navMsg := m.hub.pendingNavMsg
    m.hub.pendingNavMsg = nil
    // handle NavigateMsg or QuitAppMsg immediately
}
```

## View Components

### Hub (`internal/unified/hub.go`)

Main menu with scrollable details panel and command palette overlay. Details pane content is cached and only rebuilt when the selected detail type changes.

**Command palette**: `Ctrl+P` opens fuzzy-search overlay. Actions map to `NavigateMsg` values. Uses `bubbletea-commandpalette`.

### Browser (`internal/tui/list_browser.go`, `browser_render.go`)

Dual-panel layout: scrollable book list on the left, details panel on the right.

- Multi-select with spacebar for batch operations
- Actions: open (o), download (g), uncache (x), sync (s), edit (e), delete (d)
- Live filtering via bubbles list
- Cover art rendering with terminal image protocol detection
- Tag pills, cache status indicators, file size display

### File Picker (`internal/tui/file_picker.go`)

Miller columns layout for hierarchical directory navigation. Uses `bubbletea-millercolumns` with `bubbletea-multiselect` for checkbox state.

### Carousel (`internal/tui/edit_form.go` batch mode)

Peeking card layout for batch editing. Active card centered, adjacent cards peek from edges. Green border = saved, orange = active. Uses `bubbletea-carousel`.

## Shared Styling (`internal/tui/common.go`)

Package-level pre-computed lipgloss styles to avoid per-frame allocations:

- `StyleBorder`, `StyleBorderFocused` — view borders
- `StyleTagPill` — tag display
- `StyleDivider`, `StyleError`, `StyleProgress` — shared UI elements
- `viewOuterStyle`, `viewMasterBase`, `viewListBorderStyle` — browser layout

Color theme: teal (#1b8487, #2ecfd4) primary, orange (#fb6820) highlights.

## External Components

Five Bubble Tea components extracted to standalone repos:

| Package | Used For |
|---------|----------|
| [bubbletea-picker](https://github.com/blackwell-systems/bubbletea-picker) | Shelf picker, book picker |
| [bubbletea-multiselect](https://github.com/blackwell-systems/bubbletea-multiselect) | Batch delete, move, cache clear, file selection |
| [bubbletea-millercolumns](https://github.com/blackwell-systems/bubbletea-millercolumns) | File browser for shelving |
| [bubbletea-carousel](https://github.com/blackwell-systems/bubbletea-carousel) | Batch edit card navigation |
| [bubbletea-commandpalette](https://github.com/blackwell-systems/bubbletea-commandpalette) | Hub quick actions (Ctrl+P) |

## Performance

- **Styles**: Package-level vars, not allocated per frame
- **Image protocol**: Detected once via `sync.Once`, cached for session
- **Divider strings**: Cached on BrowserModel, rebuilt only on `WindowSizeMsg`
- **Hub details**: Raw content cached, rebuilt only when detail type changes
- **Catalog loading**: Parallel goroutines per shelf
- **Cover fetching**: Bounded concurrency (semaphore channel, 8 concurrent)

## Key Bindings (`internal/tui/keys.go`)

Pre-configured binding sets used across all components:

| Set | Keys | Used By |
|-----|------|---------|
| `PickerKeys` | quit, select | Shelf picker, book picker |
| `NavigablePickerKeys` | quit, select, back | File picker |
| `MultiSelectPickerKeys` | quit, select, toggle, back | Batch pickers |
| `FormKeys` | quit, submit, next, prev | Edit form, shelve form |
