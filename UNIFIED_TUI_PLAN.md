# Unified TUI Implementation Plan

## Goal
Eliminate screen flicker when navigating between views by implementing a single persistent Bubble Tea program with internal view switching, while maintaining 100% feature and function parity with main branch.

## Architecture
- **Current (main):** Multiple separate TUI programs (flicker on transitions)
- **Target (develop):** Single unified TUI program (zero flicker, seamless navigation)
- **Constraint:** CLI mode must remain unchanged (full backward compatibility for scripting)

---

## ✅ Completed

### Core Architecture
- [x] Created `internal/unified` package
- [x] Implemented message-based navigation (NavigateMsg, QuitAppMsg)
- [x] Created unified Model orchestrator
- [x] Hub view fully functional
- [x] Browse view placeholder (navigation works, content pending)
- [x] Proven zero-flicker architecture works

### Testing
- [x] Hub menu displays correctly
- [x] Browse navigation (hub → browse → hub) works without flicker
- [x] View switching message flow confirmed working

---

## ✅ Phase 2: Core Views (COMPLETED)

### Fully Unified Views ✅

**Hub View** - Main menu with scrollable details
- Menu navigation with filtering
- Scrollable details panel for shelves/cache
- Focus switching with Tab/arrows
- Zero flicker navigation

**Browse Library View** ✅
- Book list with multi-select (spacebar)
- Details panel with cover image display
- Filter and search functionality
- All keyboard shortcuts: o (open), e (edit), g (download), x (uncache), s (sync), c (clear selections)
- Shelf filtering
- Modified/cached indicators
- Background downloads with streaming progress
- Navigation: q/esc returns to hub via NavigateMsg

**Create Shelf View** ✅
- Text inputs for shelf name and repo name
- Checkboxes for create repo and private flags
- Async shelf creation with progress indicator
- Instant cancel returns to hub (zero flicker)
- Navigation: esc returns to hub via NavigateMsg

---

## ✅ Phase 3: Non-TUI Commands (COMPLETED)

Implemented suspend-resume pattern for terminal output commands:

- View Shelves (`shelves --table`)
- Generate HTML Index (`index`)
- Cache Info (`cache info`)
- Add from URL (`shelve-url`)
- Import from Repository (`import-repo`)
- Delete Shelf (`delete-shelf`)

All commands properly suspend TUI, execute, show output, and return to hub.

---

## ✅ Phase 4: Form Unification (COMPLETED)

### Create-Shelf View ✅

Successfully migrated create-shelf from exit-suspend pattern to unified view.

**Before:**
```
Hub → CommandRequestMsg → tea.Suspend → Exit TUI → Launch separate form
→ User cancels → Exit form → Terminal drop → Resume TUI → Hub
(4 screen clears, terminal flash on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "create-shelf"} → Show form view
→ User cancels → NavigateMsg{Target: "hub"} → Back to hub
(Zero screen clears, instant return)
```

**Implementation:**
- [x] Created `internal/unified/create_shelf.go` - Unified view with form
- [x] Created `internal/operations/shelf.go` - Shared creation logic
- [x] Added `CreateShelfCompleteMsg` to messages
- [x] Integrated into orchestrator (Init, Update, View routing)
- [x] Removed from CommandRequestMsg handler in root.go
- [x] CLI `shelfctl init` unchanged and fully scriptable

**Result:** Zero flicker, no terminal drops, instant navigation.

---

## ✅ Phase 5: Multi-Step Workflow Migrations (IN PROGRESS)

### Cache Clear View ✅ COMPLETED

Successfully migrated cache-clear from exit-restart pattern to unified view.

**Before:**
```
Hub → CacheClearRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
(Terminal drop, screen flicker on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "cache-clear"} → Picking → Confirming → Processing → Hub
(Zero screen clears, instant cancel with Esc)
```

**Implementation:**
- [x] Created `internal/unified/cache_clear.go` - Three-phase view (picking → confirming → processing)
- [x] Added `NewBookPickerMultiModel()` and `CollectSelectedBooks()` to `tui/book_picker.go`
- [x] Added `CacheClearCompleteMsg` to messages
- [x] Integrated into orchestrator (Update, View, updateCurrentView, handleNavigation)
- [x] Removed cache-clear handler from root.go
- [x] Removed dead code: `runCacheClearFromUnified()`, `CacheClearRequestMsg`
- [x] Feature parity: modified book protection, multi-select, confirmation, size reporting

### Delete Book View ✅ COMPLETED

Successfully migrated delete-book from exit-restart pattern to unified view.

**Before:**
```
Hub → DeleteRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
(Terminal drop, screen flicker on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "delete-book"} → Picking → Confirming → Processing → Hub
(Zero screen clears, instant cancel with Esc)
```

**Implementation:**
- [x] Created `internal/unified/delete_book.go` - Three-phase view (picking → confirming → processing)
- [x] Created `internal/operations/readme.go` - Shared README manipulation functions
- [x] Updated `internal/app/shelf_readme_template.go` - Thin wrappers to operations package
- [x] Added `DeleteBookCompleteMsg` to messages
- [x] Integrated into orchestrator (Update, View, updateCurrentView, handleNavigation)
- [x] Removed delete handler from root.go
- [x] Removed dead code: `runDeleteFromUnified()`, `DeleteRequestMsg`
- [x] Feature parity: multi-select, destructive warnings, asset deletion, catalog update, cache clear, README update

---

## ✅ Phase 5 (continued): Remaining Migrations (COMPLETED)

All operations have been migrated from the exit-restart pattern:

### Edit Book View ✅ COMPLETED

Successfully migrated edit-book from exit-restart pattern to unified view.

**Before:**
```
Hub → EditRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
(Terminal drop, screen flicker on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "edit-book"} → Picking → Editing (per book) → Processing → Hub
(Zero screen clears, instant cancel with Esc)
```

**Implementation:**
- [x] Created `internal/unified/edit_book.go` - Three-phase view (picking → editing → processing)
- [x] Embedded text inputs for title, author, year, tags with tab navigation
- [x] Sequential editing with [N/M] progress for multi-select
- [x] Batch commit optimization: one commit per shelf
- [x] Added `EditBookCompleteMsg` to edit_book.go
- [x] Integrated into orchestrator (Update, View, updateCurrentView, handleNavigation)
- [x] Removed edit handler from root.go
- [x] Removed dead code: `runEditFromUnified()`, `EditRequestMsg`
- [x] Feature parity: multi-select, per-book forms, batch commit, README update

### Move Book View ✅ COMPLETED

Successfully migrated move-book from exit-restart pattern to unified view.

**Before:**
```
Hub → MoveRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
(Terminal drop, screen flicker on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "move"} → BookPicking → TypePicking → DestPicking → Confirming → Processing → Hub
(Zero screen clears, instant cancel with Esc)
```

**Implementation:**
- [x] Created `internal/unified/move_book.go` - Five-phase view (book picking → type picking → dest picking → confirming → processing)
- [x] Multi-select book picker (reuses existing multiselect pattern)
- [x] Move type selection: different shelf or different release
- [x] Shelf picker for cross-shelf moves, text input for release moves
- [x] Cross-shelf move: download asset → upload to destination → delete old → update both catalogs → clear cache → update READMEs → migrate covers
- [x] Same-shelf release move: download → upload → delete old → update catalog release field
- [x] Added `MoveBookCompleteMsg` to move_book.go
- [x] Integrated into orchestrator (Update, View, updateCurrentView, handleNavigation)
- [x] Removed `MoveRequestMsg` from messages.go
- [x] Removed `pendingMove` handling from model.go and root.go
- [x] Removed dead code: `runMoveFromUnified()` from move.go
- [x] Feature parity: multi-select, cross-shelf moves, same-shelf release moves, asset transfer, catalog updates, README updates, cache clearing, cover migration
- [x] CLI `shelfctl move` unchanged and fully scriptable

### Add Book (Shelve) View ✅ COMPLETED

Successfully migrated shelve from exit-restart pattern to unified view.

**Before:**
```
Hub → ShelveRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
(Terminal drop, screen flicker on cancel)
```

**After:**
```
Hub → NavigateMsg{Target: "shelve"} → ShelfPicking → FilePicking → Setup → Ingesting → Form → Uploading → Committing → Hub
(Zero screen clears, instant cancel with Esc)
```

**Implementation:**
- [x] Exported `FilePickerModel` in `tui/file_picker.go` with `NewFilePickerModel()` constructor and accessors
- [x] Added `NewShelfDelegate()` export to `tui/shelf_picker.go`
- [x] Created `internal/unified/shelve.go` - Seven-phase view (shelf picking → file picking → setup → ingesting → form → uploading → committing)
- [x] Channel-based upload progress forwarding with inline progress bar
- [x] Embedded text inputs for metadata form (title, author, tags, ID, cache checkbox)
- [x] PDF metadata extraction for form defaults
- [x] Added `ShelveCompleteMsg` to shelve.go
- [x] Integrated into orchestrator (Update, View, updateCurrentView, handleNavigation)
- [x] Removed `ShelveRequestMsg` from messages.go
- [x] Removed `pendingShelve` handling from model.go and root.go
- [x] Removed dead code: `runShelveFromUnified()`, `runShelveInteractive()` from shelve.go
- [x] Feature parity: shelf picker, Miller columns file picker, PDF metadata, duplicate checking, asset collision handling, upload with progress, local caching, batch catalog commit, README update
- [x] CLI `shelfctl shelve` unchanged and fully scriptable

---

## ✅ CLI Mode Compatibility (MAINTAINED)

All commands work independently for scripting:

- ✅ `shelfctl browse` - Standalone TUI browser
- ✅ `shelfctl shelve file.pdf` - Direct file add
- ✅ `shelfctl edit-book book-id` - Direct edit
- ✅ `shelfctl move book-id --to-shelf foo` - Direct move
- ✅ `shelfctl delete-book book-id` - Direct delete
- ✅ `shelfctl cache clear` - Direct cache operations
- ✅ `shelfctl init` - Standalone shelf creation
- ✅ All other commands (shelves, index, cache info, etc.)

**Implementation:** Commands work both ways:
- Launched from CLI: Run standalone (original behavior)
- Launched from unified TUI: Integrated as views (zero flicker)

---

## 📊 Progress Summary

### Completion Status

| Phase | Status | Progress |
|-------|--------|----------|
| **Phase 1:** Core Infrastructure | ✅ Complete | 100% |
| **Phase 2:** Core Views (hub, browse, create-shelf) | ✅ Complete | 100% |
| **Phase 3:** Non-TUI Commands (suspend pattern) | ✅ Complete | 100% |
| **Phase 4:** Form Unification (create-shelf) | ✅ Complete | 100% |
| **Phase 5:** Multi-step workflows | ✅ Complete | 100% (5/5) |

**Overall Progress: 100%** (8/8 operations unified)

### What's Done ✅

- Hub menu (main view with details panel)
- Browse library (book browser with all features)
- Create shelf (form with checkboxes)
- Cache clear (picker → confirmation → processing)
- Delete book (picker → confirmation → processing)
- Edit book (picker → form per book → batch commit)
- Shelve / add book (shelf picker → file picker → form per file → upload with progress → batch commit)
- Move book (book picker → type picker → dest picker → confirmation → batch move)
- Terminal output commands (shelves, index, cache-info)

### What Remains ❌

All operations have been unified. No remaining exit-restart operations.

---

## 🧪 Testing Status

### Flicker Test

| Operation | Status | Notes |
|-----------|--------|-------|
| Hub → Browse → Hub | ✅ Zero flicker | Fully unified |
| Hub → Create Shelf → Hub | ✅ Zero flicker | Unified form |
| Hub → Cache Clear → Hub | ✅ Zero flicker | Unified picker + confirm |
| Hub → Delete → Hub | ✅ Zero flicker | Unified picker + confirm |
| Hub → Edit → Hub | ✅ Zero flicker | Unified form |
| Hub → Shelve → Hub | ✅ Zero flicker | Fully unified |
| Hub → Move → Hub | ✅ Zero flicker | Fully unified |

### Cancel Behavior

| Operation | Status | Notes |
|-----------|--------|-------|
| Cancel in browse | ✅ Instant return | Fully unified |
| Cancel in create-shelf | ✅ Instant return | Unified form |
| Cancel in cache-clear | ✅ Instant return | Unified picker |
| Cancel in delete | ✅ Instant return | Unified picker |
| Cancel in edit | ✅ Instant return | Unified form |
| Cancel in shelve | ✅ Instant return | Unified view |
| Cancel in move | ✅ Instant return | Unified view |

### CLI Compatibility ✅
- [x] All commands work standalone
- [x] Scripts and automation unaffected
- [x] Both TUI and CLI paths functional
- [x] `shelfctl init` works unchanged

---

## 📋 Definition of Done

The unified TUI migration will be complete when:

- [x] Core infrastructure (orchestrator, navigation, messages)
- [x] Hub view with zero flicker
- [x] Browse view with zero flicker
- [x] Create-shelf view with zero flicker
- [x] Cache clear migrated to unified view ✅
- [x] Delete books migrated to unified view ✅
- [x] Edit books migrated to unified view ✅
- [x] Move books migrated to unified view ✅
- [x] Shelve (add books) migrated to unified view ✅
- [x] Non-TUI commands use suspend pattern appropriately
- [x] CLI mode 100% backward compatible

**Current Status:** 100% complete (8/8 operations unified)

---

## 📦 Implementation Files

### Created Files

| File | Purpose | Phase |
|------|---------|-------|
| `internal/unified/model.go` | Orchestrator (~850 lines) | Phase 1 |
| `internal/unified/hub.go` | Hub view | Phase 2 |
| `internal/unified/browse.go` | Browse view | Phase 2 |
| `internal/unified/messages.go` | Navigation messages | Phase 1 |
| `internal/unified/create_shelf.go` | Create shelf form | Phase 4 |
| `internal/unified/cache_clear.go` | Cache clear view | Phase 5 |
| `internal/unified/delete_book.go` | Delete book view | Phase 5 |
| `internal/unified/edit_book.go` | Edit book view (embedded form) | Phase 5 |
| `internal/unified/shelve.go` | Shelve (add book) view (~1200 lines) | Phase 5 |
| `internal/unified/move_book.go` | Move book view (~600 lines) | Phase 5 |
| `internal/operations/shelf.go` | Shared shelf creation | Phase 4 |
| `internal/operations/readme.go` | Shared README operations | Phase 5 |

### Shared Operations Pattern

To avoid import cycles (`unified` → `app` → `unified`), shared business logic lives in `internal/operations/`:

- `shelf.go` - `CreateShelf()` used by both CLI init and TUI create-shelf
- `readme.go` - `UpdateShelfREADMEStats()`, `AppendToShelfREADME()`, `RemoveFromShelfREADME()` used by both CLI commands and TUI views

### Unified View Pattern

All unified views follow the same three-phase state machine:

```
Picking → Confirming → Processing → NavigateMsg{Target: "hub"}
```

Each phase handles its own key events and rendering. Async operations return completion messages via `tea.Cmd`.

---

## 📝 Architecture Notes

### Unified View Pattern (For TUI Forms)

Forms without terminal output should be unified views:

```go
case "esc":
    // Return to hub instantly (no terminal drop)
    return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }

case "enter":
    // Submit form, process async
    m.processing = true
    return m, func() tea.Msg {
        err := performOperation()
        return CompleteMsg{Err: err}
    }

case CompleteMsg:
    if msg.Err != nil {
        // Show error in-form, stay in TUI
        m.err = msg.Err
        return m, nil
    }
    // Success - return to hub
    return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
```

### Suspend Pattern (For Terminal Output)

Commands that print formatted output should suspend:

```go
case "shelves":
    return m, tea.Suspend(func() error {
        // Exit alt screen → terminal
        cmd := newShelvesCmd()
        cmd.Execute()
        fmt.Println("\nPress Enter to return...")
        fmt.Scanln()
        // Resume alt screen → TUI
        return nil
    })
```

**Rule:** If it's interactive TUI, unify it. If it's terminal output, suspend it.
