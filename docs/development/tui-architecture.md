# TUI Architecture

This document describes shelfctl's terminal user interface architecture — a unified single-program design using the Bubble Tea framework.

---

## Overview

When launched with no arguments (`shelfctl`), a single persistent Bubble Tea program runs for the entire session. All views (hub, browse, edit, shelve, etc.) are internal state transitions — no screen flicker, no program restarts.

```
shelfctl (no args)
  ↓
Single tea.NewProgram starts
  ↓
Hub view → user selects "Browse" → Browse view (instant, no flicker)
  ↓
Browse view → user presses 'q' → Hub view (instant, no flicker)
  ↓
... continues until user quits
```

Direct commands (`shelfctl browse`, `shelfctl shelve book.pdf`, etc.) still run standalone as before.

---

## Core Components

### Orchestrator (`internal/unified/model.go`)

The central coordinator. Holds all view models and routes messages/rendering to the active view.

```go
type Model struct {
    currentView string       // "hub", "browse", "edit-book", etc.
    hub         HubModel
    browse      BrowseModel
    editBook    EditBookModel
    shelve      ShelveModel
    // ... other view models

    width, height int
}
```

- `Update()` routes messages to the current view, except navigation messages which switch views
- `View()` delegates rendering to the current view
- Single `tea.NewProgram()` call in `internal/app/root.go`

### Navigation Messages (`internal/unified/messages.go`)

Views communicate via typed messages instead of return values:

```go
// Switch to another view
type NavigateMsg struct {
    Target   string        // "hub", "browse", "shelve", etc.
    Data     interface{}   // Optional context
    BookItem *tui.BookItem // Optional single book for direct-edit flows
}

// Exit the application
type QuitAppMsg struct{}

// Request an operation that suspends the TUI
type ActionRequestMsg struct {
    Action   tui.BrowserAction
    BookItem *tui.BookItem
    ReturnTo string
}

// Request a non-TUI command (shelves, index, cache-info, etc.)
type CommandRequestMsg struct {
    Command  string
    ReturnTo string
}
```

### View Models (`internal/unified/*.go`)

Each view is a self-contained model that:
- Handles its own key bindings and rendering
- Emits `NavigateMsg` to switch views (never calls `tea.Quit`)
- Emits `ActionRequestMsg` or `CommandRequestMsg` for operations needing TUI suspension

**Fully integrated views** (zero flicker):
- `hub.go` — Main menu with scrollable details panel and command palette (`ctrl+p`)
- `browse.go` — Book browser with multi-select and actions
- `shelve.go` — File picker and metadata form
- `edit_book.go` — Book picker and edit form
- `move_book.go` — Multi-select picker and destination selector
- `delete_book.go` — Multi-select picker with confirmation
- `cache_clear.go` — Multi-select picker for cache operations
- `create_shelf.go` — Shelf creation form

**Suspend-resume operations** (brief screen clear, then returns):
- View shelves, generate index, cache info — print output to terminal
- Import repository, delete shelf — run interactive workflows

---

## Shared TUI Utilities (`internal/tui/`)

### Footer Highlight (`footer.go`)

Shared across all views. When a shortcut key is pressed, the corresponding footer label highlights for 500ms.

```go
// Set highlight and return a 500ms clear timer
func SetActiveCmd(activeCmd *string, key string) tea.Cmd

// Render footer bar with optional highlight
func RenderFooterBar(shortcuts []ShortcutEntry, activeCmd string) string
```

Each view has an `activeCmd string` field, handles `ClearActiveCmdMsg`, and calls `RenderFooterBar` in its `View()`.

### Reusable Components

- `list_browser.go` — Browse list with details panel, multi-select, download progress
- `book_picker.go` — Single and multi-select book pickers
- `shelf_picker.go` — Shelf selection picker
- `file_picker.go` — Miller columns file browser with multi-select
- `edit_form.go` — Metadata edit form with text inputs
- `shelve_form.go` — Add book form with metadata + cache checkbox
- `shelf_create_form.go` — New shelf creation form
- `progress.go` — Download/upload progress bar

All pickers use `picker.Base` from `bubbletea-picker` for consistent key handling, window resize, and border rendering.

---

## Suspend-Resume Pattern

For operations that need normal terminal output (tables, external commands):

```
Hub view (in TUI)
  ↓ User selects "View Shelves"
  ↓
CommandRequestMsg{Command: "shelves", ReturnTo: "hub"}
  ↓
Orchestrator suspends TUI (exits alt screen)
  ↓
Command runs in normal terminal, prints output
  ↓
"Press Enter to return..."
  ↓
TUI resumes, navigates back to hub
```

This causes one screen clear (unavoidable when exiting alt screen), but only for operations that require terminal output.

---

## File Structure

```
internal/
├── app/
│   └── root.go              # Entry point, launches unified program
├── tui/
│   ├── footer.go            # Shared footer highlight utilities
│   ├── list_browser.go      # Browse view (model + renderer)
│   ├── hub.go               # Hub menu (standalone mode)
│   ├── book_picker.go       # Book selection pickers
│   ├── shelf_picker.go      # Shelf selection picker
│   ├── file_picker.go       # Miller columns file browser
│   ├── edit_form.go         # Metadata edit form
│   ├── shelve_form.go       # Add book form
│   ├── shelf_create_form.go # Shelf creation form
│   ├── progress.go          # Progress bar
│   ├── styles.go            # Shared lipgloss styles
│   ├── keys.go              # Standard key bindings
│   └── delegate/            # List delegate base component
└── unified/
    ├── model.go             # Orchestrator (~850 lines)
    ├── messages.go          # Navigation message types
    ├── hub.go               # Hub view (wraps tui.HubModel)
    ├── browse.go            # Browse view adapter
    ├── edit_book.go         # Edit book workflow
    ├── shelve.go            # Shelve workflow
    ├── move_book.go         # Move book workflow
    ├── delete_book.go       # Delete book workflow
    ├── cache_clear.go       # Cache clear workflow
    └── create_shelf.go      # Create shelf form
```

---

## Message Flow Example

User browses library, edits a book, returns to hub:

```
1. Hub renders menu → user presses Enter on "Browse Library"
2. Hub emits NavigateMsg{Target: "browse"}
3. Orchestrator sets currentView = "browse", initializes browse model
4. Browse renders instantly (no flicker)
5. User presses 'e' on a book
6. Browse emits NavigateMsg{Target: "edit-book-single", BookItem: selected}
7. Orchestrator sets currentView = "edit-book", initializes with single book
8. Edit form renders instantly (no flicker)
9. User edits metadata, presses Ctrl+S to save
10. Edit view commits changes, emits NavigateMsg{Target: "browse"}
11. Browse re-renders with updated metadata (no flicker)
12. User presses 'q' → NavigateMsg{Target: "hub"} → hub renders instantly
```

---

## CLI Compatibility

The unified TUI only activates when running `shelfctl` with no arguments. All direct commands work exactly as before:

```bash
# Direct commands (no unified TUI)
shelfctl browse --shelf programming
shelfctl shelve book.pdf --shelf prog --title "..."
shelfctl edit-book book-id --title "New Title"

# Interactive hub (unified TUI)
shelfctl
```

Detection in `root.go`:
```go
func shouldRunUnifiedTUI() bool {
    return len(os.Args) == 1 && tui.ShouldUseTUI(rootCmd)
}
```

---

## Adding a New View

1. Create `internal/unified/new_view.go` with a model struct implementing `Init()`, `Update()`, `View()`
2. Add `activeCmd string` field and `ClearActiveCmdMsg` handler for footer highlights
3. Emit `NavigateMsg` to navigate away (never `tea.Quit`)
4. Add model field to `Model` struct in `model.go`
5. Add routing cases in `Update()`, `View()`, and `initView()`
6. Add menu item in `hub.go` if accessible from hub

---

## Performance

- **Startup:** ~50ms (single program launch)
- **View transitions:** <1ms (internal state change)
- **Memory:** ~10-15 MB (all view models in memory)
- **Terminal:** Alt screen entered once, persists until quit
