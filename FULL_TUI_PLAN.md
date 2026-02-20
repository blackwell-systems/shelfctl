# Full TUI Mode Implementation Plan

## Current State

**Already have TUI:**
- [DONE] `browse` - Interactive book browser with navigation/search
- [DONE] `shelve` - Shelf picker → File picker → Metadata form

**CLI-only (no TUI):**
- `init` - Bootstrap shelf repos
- `shelves` - Validate shelves
- `info` - Show book metadata
- `open` - Download and open book
- `move` - Move books between releases/shelves
- `split` - Split shelf into sub-shelves
- `migrate` - Migrate from old repos
- `import` - Import from other shelves

## Approach A: TUI Hub (Recommended)

Create a main dashboard when running `shelfctl` with no command.

### Implementation

#### 1. Main Menu (new file: `internal/tui/hub.go`)

```go
type HubModel struct {
    list     list.Model
    quitting bool
}

// Menu items
type MenuItem struct {
    Key         string
    Label       string
    Description string
    Action      func() tea.Cmd
}

var menuItems = []MenuItem{
    {"browse", "Browse Library", "View and search your books", launchBrowse},
    {"shelve", "Add Book", "Add a new book to your library", launchShelve},
    {"shelves", "View Shelves", "List all configured shelves", launchShelves},
    {"move", "Move Book", "Move a book between shelves/releases", launchMove},
    {"open", "Open Book", "Search and open a book", launchOpen},
    {"info", "Book Info", "View detailed book metadata", launchInfo},
    {"migrate", "Migrate", "Import books from old repos", launchMigrate},
    {"import", "Import Shelf", "Copy books from another shelf", launchImport},
    {"split", "Split Shelf", "Organize a large shelf", launchSplit},
    {"quit", "Quit", "Exit shelfctl", nil},
}
```

**Visual Design:**
```
┌─────────────────────────────────────────────────┐
│  shelfctl - Personal Library Manager            │
├─────────────────────────────────────────────────┤
│                                                  │
│  › Browse Library          View and search books│
│    Add Book               Add new book          │
│    View Shelves           List shelves          │
│    Move Book              Reorganize books      │
│    Open Book              Search and open       │
│    Book Info              View metadata         │
│    Migrate                Import from old repo  │
│    Import Shelf           Copy from shelf       │
│    Split Shelf            Organize large shelf  │
│    Quit                   Exit                  │
│                                                  │
│  ↑/↓: navigate  enter: select  q: quit          │
└─────────────────────────────────────────────────┘
```

#### 2. Enhanced Operations with TUI

##### `open` command (new file: `internal/tui/open_picker.go`)
Currently: `shelfctl open <id>`
TUI enhancement: Fuzzy search picker

```go
// Show searchable list of books
type OpenPickerModel struct {
    books    []BookItem
    filtered []BookItem
    search   textinput.Model
    list     list.Model
}
```

**Flow:**
1. Show all books with search box
2. Type to filter (fuzzy search on title/author/ID)
3. Select book → downloads if needed → opens

##### `info` command (new file: `internal/tui/info_viewer.go`)
Currently: Prints metadata
TUI enhancement: Searchable book list → detail view

```go
type InfoViewerModel struct {
    picker   OpenPickerModel  // reuse picker
    viewing  *catalog.Book
    viewport viewport.Model
}
```

**Flow:**
1. Show book picker (same as open)
2. Display full metadata in scrollable viewport
3. Actions: Open, Move, Delete, Back

##### `move` command (new file: `internal/tui/move_wizard.go`)
Currently: `shelfctl move <id> --to-shelf X --to-release Y`
TUI enhancement: Multi-step wizard

```go
type MoveWizardModel struct {
    step         int  // 0=select book, 1=target, 2=confirm
    selectedBook *BookItem
    targetShelf  string
    targetRelease string
    shelves      []ShelfOption
    releases     []string
}
```

**Flow:**
1. Select book (picker)
2. Choose target shelf (list)
3. Choose target release (list)
4. Confirm with preview
5. Execute with progress bar

##### `split` command (new file: `internal/tui/split_wizard.go`)
Currently: Interactive but uses bufio prompts
TUI enhancement: Visual wizard

```go
type SplitWizardModel struct {
    step          int  // 0=shelf, 1=strategy, 2=preview, 3=execute
    sourceShelf   string
    strategy      string  // "by-tag", "by-size", "manual"
    groupings     map[string][]catalog.Book
    targetReleases map[string]string
}
```

**Flow:**
1. Select source shelf
2. Choose split strategy:
   - By tag (group books by tags)
   - By size (split evenly)
   - Manual (assign one by one)
3. Preview groupings with editable releases
4. Confirm and execute with progress

##### `migrate` command (new file: `internal/tui/migrate_wizard.go`)
Currently: CLI-only with queue files
TUI enhancement: Source browser + mapping wizard

```go
type MigrateWizardModel struct {
    step        int  // 0=source, 1=scan, 2=map, 3=execute
    sourceRepo  string
    sourceRef   string
    files       []string
    mappings    map[string]string  // file -> shelf
    progress    int
}
```

**Flow:**
1. Enter source repo (text input)
2. Scan and show files (tree view)
3. Map files to shelves (interactive assignment)
4. Execute with progress bars
5. Show ledger/summary

##### `import` command (new file: `internal/tui/import_wizard.go`)
Currently: `shelfctl import --from-shelf X --to-shelf Y`
TUI enhancement: Source picker + book selector

```go
type ImportWizardModel struct {
    step         int  // 0=from, 1=to, 2=books, 3=confirm
    fromOwner    string
    fromRepo     string
    toShelf      string
    books        []catalog.Book
    selected     map[string]bool  // multi-select
}
```

**Flow:**
1. Enter source owner/repo
2. Select target shelf
3. Show books with checkboxes (select which to import)
4. Confirm and execute with progress

##### `shelves` command (new file: `internal/tui/shelves_viewer.go`)
Currently: Prints validation results
TUI enhancement: Dashboard with actions

```go
type ShelvesViewerModel struct {
    shelves []ShelfStatus
    list    list.Model
}

type ShelfStatus struct {
    Name   string
    Repo   string
    Status string  // "ok", "error"
    Books  int
    Error  string
}
```

**Visual:**
```
┌─────────────────────────────────────────────────┐
│  Configured Shelves                              │
├─────────────────────────────────────────────────┤
│  › ✓ programming (shelf-programming) 42 books   │
│    ✓ history (shelf-history) 18 books           │
│    ✗ fiction (shelf-fiction) repo not found     │
│                                                  │
│  enter: browse  d: delete  r: refresh           │
└─────────────────────────────────────────────────┘
```

#### 3. Root Command Integration

Modify `internal/app/root.go`:

```go
func Execute() error {
    rootCmd := &cobra.Command{
        Use:   "shelfctl",
        Short: "Personal library manager",
        RunE: func(cmd *cobra.Command, args []string) error {
            // If no subcommand, launch TUI hub
            if tui.ShouldUseTUI(cmd) {
                return tui.RunHub(cfg, gh, cacheMgr)
            }
            return cmd.Help()
        },
    }
    // ... register subcommands
}
```

## Approach B: Enhance Individual Commands Only

Skip the hub, just add TUI to each command when called directly.

**Pros:**
- Simpler to implement incrementally
- Each command works standalone
- Familiar CLI pattern

**Cons:**
- No unified starting point
- User must know which command they want
- Less discoverable

## Recommended Implementation Order

If going with Approach A (Hub):

### Phase 1: Core Hub (1-2 days)
1. Create `internal/tui/hub.go` with main menu
2. Wire up existing TUI commands (browse, shelve)
3. Make hub launch from `shelfctl` with no args
4. Add "Press enter for interactive mode" to CLI help

### Phase 2: Simple TUI Commands (2-3 days)
5. Implement `open` picker (reuse book list)
6. Implement `info` viewer (picker + detail)
7. Implement `shelves` dashboard

### Phase 3: Wizard Commands (3-4 days)
8. Implement `move` wizard
9. Implement `import` wizard
10. Implement `split` wizard
11. Implement `migrate` wizard

### Phase 4: Polish (1-2 days)
12. Add status bar to hub showing shelf/cache stats
13. Add keyboard shortcuts (o=open, b=browse, etc.)
14. Add help screen (press ? for keybindings)
15. Add themes/colors configuration
16. Error handling and edge cases

## File Structure

```
internal/
  tui/
    # Existing
    detection.go
    common.go
    list_browser.go
    shelve_form.go
    file_picker.go
    shelf_picker.go

    # New for Hub
    hub.go              # Main menu/dashboard

    # New command TUIs
    open_picker.go      # Searchable book picker for opening
    info_viewer.go      # Book metadata viewer
    shelves_viewer.go   # Shelf status dashboard
    move_wizard.go      # Move book wizard
    import_wizard.go    # Import shelf wizard
    split_wizard.go     # Split shelf wizard
    migrate_wizard.go   # Migration wizard

    # Reusable components
    book_picker.go      # Generic searchable book selector
    progress_bar.go     # Progress display for long operations
    confirm_dialog.go   # Yes/no confirmation dialogs
    text_input_form.go  # Multi-field text input form
```

## Components Needed

### Reusable Components

1. **BookPicker** (used by open, info, move)
   - Searchable list of books across all shelves
   - Fuzzy search on title/author/ID
   - Returns selected book

2. **ProgressBar** (used by move, migrate, import)
   - Show progress for long operations
   - Supports: count (N/M), bytes (MB/GB), percentage
   - Can be indeterminate spinner

3. **ConfirmDialog** (used by move, split, delete)
   - Yes/No confirmation
   - Shows preview of action
   - Returns boolean

4. **TextInputForm** (already have for shelve)
   - Enhance to be more reusable
   - Support validation
   - Optional fields with defaults

## Testing Strategy

Each TUI component should:
1. Work in alt screen (cleans up on exit)
2. Handle Ctrl+C gracefully
3. Fall back to CLI mode when piped/redirected
4. Have `--no-interactive` override

## CLI Compatibility

All commands must still work in non-interactive mode:
```bash
# These should continue to work exactly as before
shelfctl browse --shelf programming --no-interactive
shelfctl shelve book.pdf --shelf prog --title "..."
shelfctl open book-id
shelfctl move book-id --to-shelf history

# But these also work (TUI mode):
shelfctl browse          # launches TUI browser
shelfctl shelve          # launches TUI workflow
shelfctl open            # launches TUI picker
shelfctl move            # launches TUI wizard
```

## Estimated Effort

- **Phase 1 (Hub):** 1-2 days
- **Phase 2 (Simple):** 2-3 days
- **Phase 3 (Wizards):** 3-4 days
- **Phase 4 (Polish):** 1-2 days

**Total:** 7-11 days of focused development

Can be done incrementally - each phase is useful on its own.

## Benefits of Full TUI

1. **Discoverability** - Users see all available actions
2. **Guidance** - Wizards walk through complex operations
3. **Feedback** - Real-time validation and progress
4. **Efficiency** - Faster than remembering CLI flags
5. **Visual** - See library state at a glance
6. **Forgiving** - Easy to cancel/go back vs. CLI mistakes

## Next Steps

If you want to proceed:
1. Choose approach (A=Hub, B=Individual)
2. Prioritize which commands to add TUI to first
3. Create detailed design for hub menu/navigation
4. Start with Phase 1 implementation
