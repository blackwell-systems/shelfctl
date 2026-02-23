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

## 🚧 In Progress

### Hub View Size Fix
- [ ] Test hub list sizing after returning from browse view
- [ ] Verify all menu items visible without scrolling

---

## 📋 TODO - TUI Views (Critical Path)

### 1. Browse Library View
**File:** `internal/tui/list_browser.go` (912 lines)
**Status:** Placeholder exists, needs full migration

**Required Functionality:**
- [ ] Book list with multi-select (spacebar)
- [ ] Details panel (right side)
- [ ] Cover image display
- [ ] Filter and search
- [ ] Keyboard shortcuts: o (open), e (edit), g (download), x (uncache), s (sync), c (clear selections)
- [ ] Shelf filtering
- [ ] Modified/cached indicators
- [ ] Background downloads with progress
- [ ] Navigation: q/esc returns to hub (emit NavigateMsg)

**Migration Strategy:**
- Extract core list_browser model
- Adapt quit behavior to emit NavigateMsg instead of tea.Quit
- Integrate with unified model
- Test all shortcuts and multi-select

---

### 2. Add Book (Shelve) View
**Files:** `internal/tui/file_picker.go`, `internal/tui/shelve_form.go`
**Status:** Not implemented

**Required Functionality:**
- [ ] Miller columns file picker
- [ ] Multi-file selection
- [ ] PDF metadata extraction
- [ ] Metadata form (title, author, year, tags, cache checkbox)
- [ ] Progress indicators for batch adds
- [ ] Navigation: q/esc returns to hub

**Migration Strategy:**
- Wrap existing file picker + shelve form
- Adapt to emit NavigateMsg on completion/cancel
- Handle batch operations
- Show success/error summary before returning to hub

---

### 3. Edit Book View
**File:** `internal/app/edit_book.go`, `internal/tui/edit_form.go`
**Status:** Not implemented

**Required Functionality:**
- [ ] Book picker (multi-select for batch edit)
- [ ] Edit form with current values
- [ ] Batch confirmation for multiple books
- [ ] Progress indicators
- [ ] Success/failure summary
- [ ] Navigation: q/esc returns to hub

**Migration Strategy:**
- Wrap book picker + edit form
- Emit NavigateMsg on completion/cancel
- Preserve batch edit functionality

---

### 4. Move Book View
**File:** `internal/app/move.go`
**Status:** Not implemented

**Required Functionality:**
- [ ] Book picker (multi-select)
- [ ] Shelf/release picker
- [ ] Batch confirmation
- [ ] Progress indicators
- [ ] Validation (same shelf → different release)
- [ ] Success/failure summary
- [ ] Navigation: q/esc returns to hub

---

### 5. Delete Book View
**File:** `internal/app/delete_book.go`
**Status:** Not implemented

**Required Functionality:**
- [ ] Book picker (multi-select)
- [ ] Batch confirmation ("DELETE N BOOKS")
- [ ] Progress indicators
- [ ] Success/failure summary
- [ ] Navigation: q/esc returns to hub

---

### 6. Cache Clear View
**File:** `internal/app/cache.go` (clear subcommand)
**Status:** Not implemented

**Required Functionality:**
- [ ] Book picker (multi-select)
- [ ] Protection for modified books
- [ ] Batch confirmation
- [ ] Progress indicators
- [ ] Success/failure summary
- [ ] Navigation: q/esc returns to hub

---

## 📋 TODO - Non-TUI Commands

These commands output to terminal and don't use full-screen TUI.
**Strategy:** Suspend TUI → run command → show output → wait for Enter → return to hub

### Commands
- [ ] View Shelves (`shelves --table`)
- [ ] Generate HTML Index (`index`)
- [ ] Cache Info (`cache info`)
- [ ] Add from URL (`shelve-url`)
- [ ] Import from Repository (`import-repo`)
- [ ] Delete Shelf (`delete-shelf`)

**Implementation Approach:**
```go
case "shelves":
    // Suspend TUI (exit alt screen)
    tea.SuspendProgram()
    // Run command
    cmd := newShelvesCmd()
    cmd.SetArgs([]string{"--table"})
    err := cmd.Execute()
    // Show output (already printed)
    fmt.Println(color.CyanString("\nPress Enter to return to menu..."))
    fmt.Scanln()
    // Resume TUI (enter alt screen)
    tea.ResumeProgram()
    // Stay on hub
    return m, nil
```

---

## 📋 TODO - CLI Mode Compatibility

**Requirement:** All commands must work independently for scripting

### CLI Commands (must work unchanged)
- [ ] `shelfctl browse`
- [ ] `shelfctl shelve file.pdf`
- [ ] `shelfctl edit-book book-id`
- [ ] `shelfctl move book-id --to-shelf foo`
- [ ] `shelfctl delete-book book-id`
- [ ] `shelfctl cache clear`
- [ ] All other commands (shelves, index, cache info, etc.)

**Implementation:**
- Commands detect if launched from hub (unified mode) vs CLI
- If CLI: run independently as now (separate TUI programs)
- If unified: integrate into orchestrator

**Detection Strategy:**
```go
// In each command
if isUnifiedMode() {
    // Return model for integration
    return createUnifiedModel()
} else {
    // Run standalone TUI as now
    p := tea.NewProgram(...)
    return p.Run()
}
```

---

## 🧪 Testing Checklist

### Unit Tests
- [ ] Message flow (NavigateMsg, QuitAppMsg)
- [ ] View switching logic
- [ ] Model orchestration

### Integration Tests
- [ ] Hub → each view → back to hub
- [ ] All keyboard shortcuts work in each view
- [ ] Multi-select operations
- [ ] Batch confirmations
- [ ] Progress indicators
- [ ] Error handling

### Manual Testing
- [ ] Hub menu displays correctly
- [ ] All views size properly on first load
- [ ] All views size properly when returning
- [ ] No screen flicker on any transition
- [ ] CLI commands work independently
- [ ] Non-TUI commands suspend/resume correctly

### Regression Testing (vs main)
- [ ] All features from main work identically
- [ ] All keyboard shortcuts preserved
- [ ] All data operations work correctly
- [ ] No functional regressions

---

## 📊 Progress Tracking

**Completion Estimate:**
- Core architecture: ✅ Done
- TUI views migration: 0% (0/6 views)
- Non-TUI command handling: 0% (0/6 commands)
- CLI mode compatibility: 0%
- Testing: 0%

**Overall Progress: ~10%**

**Estimated Remaining Work:**
- Browse migration: 6-8 hours
- Other TUI views: 8-10 hours
- Non-TUI commands: 2-3 hours
- CLI compatibility: 2-3 hours
- Testing & bug fixes: 4-6 hours

**Total: 22-30 hours of focused development**

---

## 🚀 Deployment Criteria

Before merging to main:
- [ ] 100% feature parity confirmed
- [ ] All TUI views working
- [ ] All non-TUI commands working
- [ ] CLI mode fully compatible
- [ ] Zero screen flicker on all transitions
- [ ] No functional regressions
- [ ] All tests passing
- [ ] Manual testing complete
- [ ] User acceptance testing passed

---

## 📝 Notes

- Architecture is proven - flicker elimination works
- No shortcuts or compromises on feature parity
- CLI mode backward compatibility is non-negotiable
- Each view migration should be tested before moving to next
- Commit frequently for rollback safety
