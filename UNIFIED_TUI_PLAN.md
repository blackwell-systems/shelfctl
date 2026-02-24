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

**Create Shelf View** ✅ (Just Completed)
- Text inputs for shelf name and repo name
- Checkboxes for create repo and private flags
- Async shelf creation with progress indicator
- Instant cancel returns to hub (zero flicker)
- Navigation: esc returns to hub via NavigateMsg

---

## 🚧 Phase 3: Multi-Step Workflows (NOT YET MIGRATED)

These operations still use the **exit-restart pattern** (terminal drops on cancel):

### Add Book (Shelve) View ❌ NEEDS MIGRATION
- **Current:** ShelveRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
- **Workflow:** Shelf picker → file picker → metadata form (per file) → upload → catalog commit
- **Complexity:** Multi-step with multiple separate TUI programs
- **Status:** NOT UNIFIED

### Edit Book View ❌ NEEDS MIGRATION
- **Current:** EditRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
- **Workflow:** Book picker (multi-select) → edit form (per book) → catalog commit
- **Complexity:** Two-step with picker + forms
- **Status:** NOT UNIFIED

### Move Book View ❌ NEEDS MIGRATION
- **Current:** MoveRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
- **Workflow:** Book picker (multi-select) → destination picker → confirmation → move → commit
- **Complexity:** Three-step with pickers + confirmation
- **Status:** NOT UNIFIED

### Delete Book View ❌ NEEDS MIGRATION
- **Current:** DeleteRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
- **Workflow:** Book picker (multi-select) → confirmation → delete → commit
- **Complexity:** Two-step with picker + confirmation
- **Status:** NOT UNIFIED

### Cache Clear View ❌ NEEDS MIGRATION
- **Current:** CacheClearRequestMsg → tea.Quit → exit TUI → run workflow → restart TUI
- **Workflow:** Book picker (multi-select) → confirmation → cache clear
- **Complexity:** Two-step with picker + confirmation
- **Status:** NOT UNIFIED

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

## 🚧 Phase 5: Remaining Work

### Delete-Shelf View (Optional)

Currently uses CommandRequestMsg → suspend pattern.

**Current behavior:**
- Exit TUI → Run delete command → Return to hub
- Terminal drop, screen clear

**Should migrate to:** Unified view (same pattern as delete-book)
- [x] Currently uses suspend
- [ ] Create `internal/unified/delete_shelf.go`
- [ ] Shelf picker → confirmation → deletion
- [ ] NavigateMsg on complete/cancel

**Priority:** LOW - delete-shelf is rarely used, suspend pattern acceptable

### Import-Repository (Keep Suspend Pattern)

Complex multi-step workflow with terminal output:
1. Scan repository for files
2. Show Miller columns for path selection
3. Display migration progress table
4. Print final statistics

**Decision:** **Keep suspend pattern** because:
- Needs rich terminal output (tables, progress bars)
- Multi-step workflow benefits from terminal visibility
- Users may want to copy/paste paths or IDs
- Acceptable UX for infrequent operation

**No migration needed.**

### What Should Stay Suspended

Operations that **need terminal output** should keep suspend pattern:
- Shelves table (formatted output)
- Cache info (statistics table)
- Index generation (file creation, path output)
- Shelve-url (runs gh commands)

**Reason:** Terminal output is the feature. CLI and TUI should show identical formatted output.

---

## 📋 Remaining Optional Work

### Delete-Shelf View Migration (Optional, Low Priority)

Currently uses CommandRequestMsg → suspend pattern. Could be migrated to unified view for consistency.

**Would require:**
- Create `internal/unified/delete_shelf.go`
- Shelf picker → confirmation → deletion
- Emit NavigateMsg on complete/cancel
- Keep `shelfctl delete-shelf` CLI unchanged

**Justification for keeping suspend:**
- Rarely used operation (most users never delete shelves)
- Current UX is acceptable
- Not worth the effort unless user requests it

**Recommendation:** Keep suspend pattern, migrate only if requested

---

## 📦 Implementation Summary

### What Was Completed in This Session

**Primary Goal Achieved:** ✅ Eliminate terminal drops on cancel in create-shelf form

**Files Created:**
1. **`internal/unified/create_shelf.go`** (234 lines)
   - Unified view for shelf creation form
   - Text inputs: shelf name, repository name
   - Checkboxes: create repo (default: yes), private (default: yes)
   - Async shelf creation with `CreateShelfCompleteMsg`
   - Instant navigation with NavigateMsg (zero flicker)

2. **`internal/operations/shelf.go`** (172 lines)
   - Extracted shared shelf creation logic
   - Avoids import cycle between `app` and `unified` packages
   - Single source of truth for shelf creation
   - Used by both CLI and TUI

**Files Modified:**
1. **`internal/unified/model.go`**
   - Added `ViewCreateShelf` constant
   - Added `createShelf CreateShelfModel` field to orchestrator
   - Routed "create-shelf" in Init(), Update(), View() methods
   - Passes gh and cfg dependencies to view

2. **`internal/unified/messages.go`**
   - Added `CreateShelfCompleteMsg` for async completion
   - Removed "create-shelf" from CommandRequestMsg comment

3. **`internal/app/root.go`**
   - Deleted "create-shelf" case from CommandRequestMsg handler (lines 659-661)
   - No longer uses suspend pattern for this operation

4. **`UNIFIED_TUI_PLAN.md`** (this file)
   - Updated Phase 4 status to completed
   - Updated progress to 95%
   - Documented remaining optional work

5. **`docs/TUI_ARCHITECTURE.md`**
   - Added "Current State and Remaining Work" section
   - Documented what's unified vs what still uses suspend
   - Explained the terminal drop problem and solution

### Technical Approach

**Navigation Flow:**
```
Before: Hub → CommandRequestMsg → tea.Suspend → separate TUI → cancel → terminal drop → resume
After:  Hub → NavigateMsg → create-shelf view → cancel → NavigateMsg → hub (instant, zero flicker)
```

**Key Pattern:**
- Esc key emits: `NavigateMsg{Target: "hub"}` instead of `tea.Quit`
- Form submission runs async via tea.Cmd
- Completion emits: `CreateShelfCompleteMsg{Err: error}`
- Success or error stays in TUI, no terminal involvement

**Import Cycle Resolution:**
- Problem: `unified` needs `app` functions, `app` imports `unified`
- Solution: Extract shared logic to `operations` package
- Both `app` and `unified` can now import `operations` without cycle

### CLI Compatibility Preserved

The standalone `shelfctl init` command works unchanged:
```bash
# Interactive prompting
shelfctl init --repo shelf-books --name books --create-repo

# Fully specified (scriptable)
shelfctl init --owner johndoe --repo shelf-tech --name tech --create-repo --private=false
```

CLI and TUI both call `operations.CreateShelf()` for consistency.

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

## 🎯 Summary: Actual State

### What's Actually Unified ✅

**Only 3 operations:**
- Hub (main menu)
- Browse (library browser)
- Create-shelf (just completed)

### What Still Needs Migration ❌

**5 operations still use exit-restart (terminal drops on cancel):**
- Shelve (add books) - Multi-step: shelf picker → file picker → forms
- Edit books - Two-step: book picker → edit forms
- Move books - Three-step: book picker → dest picker → confirmation
- Delete books - Two-step: book picker → confirmation
- Cache clear - Two-step: book picker → confirmation

**Current actual completion: ~40%** (3/8 operations unified)

### Critical Work Remaining

**Priority 1: Simple Operations** (~5-6 hours)
1. **Delete books** - Book picker + confirmation (LOW complexity)
2. **Cache clear** - Book picker + confirmation (LOW complexity)

**Priority 2: Medium Operations** (~8-10 hours)
3. **Edit books** - Book picker + forms loop (MEDIUM complexity)
4. **Move books** - Book picker + dest picker + confirmation (MEDIUM complexity)

**Priority 3: Complex Operation** (~6-8 hours)
5. **Shelve (add books)** - Shelf picker + file picker + forms + upload (HIGH complexity)

**Total Remaining: 19-24 hours**

### Recommended Approach

**Option A: Complete Migration** ✅ RECOMMENDED
- Migrate all 5 remaining operations
- Achieve true unified experience
- Zero terminal drops across entire app
- Consistent UX
- **Effort:** ~20 hours
- **Result:** Production-ready unified TUI

**Option B: Partial Ship** ⚠️ NOT RECOMMENDED
- Ship with only hub/browse/create-shelf unified
- Users still see terminal drops for shelve/edit/move/delete/cache
- Inconsistent UX defeats purpose of unified TUI
- Would need to complete later anyway

**Option C: Incremental Migration**
- Migrate in priority order (delete → cache → edit → move → shelve)
- Test and commit after each
- Ship when comfortable with progress
- **Effort:** Same ~20 hours, but staged

### Verdict

**Cannot ship at 40% completion.** The most frequently used operations (shelve, edit, delete) still cause terminal drops, which was the original problem we're solving.

**Recommend:** Complete the migration (Option A or C) before shipping.

---

## 📊 Progress Summary - CORRECTED

### Actual Completion Status

| Phase | Status | Progress |
|-------|--------|----------|
| **Phase 1:** Core Infrastructure | ✅ Complete | 100% |
| **Phase 2:** Core Views (hub, browse, create-shelf) | ✅ Complete | 100% |
| **Phase 3:** Non-TUI Commands (suspend pattern) | ✅ Complete | 100% |
| **Phase 4:** Multi-step workflows | ❌ Not Started | 0% |

**Overall Progress: ~40%** (3/8 operations unified)

### What's Done ✅

- Hub menu (main view with details panel)
- Browse library (book browser with all features)
- Create shelf (form with checkboxes)
- Terminal output commands (shelves, index, cache-info)

### What Remains ❌

**Critical operations still using exit-restart:**
1. **Shelve (add books)** - Most used operation, HIGH complexity
2. **Edit books** - Frequently used, MEDIUM complexity
3. **Move books** - Occasionally used, MEDIUM complexity
4. **Delete books** - Occasionally used, LOW complexity
5. **Cache clear** - Occasionally used, LOW complexity

**Estimated remaining effort: 19-24 hours**

---

## 🧪 Testing Status

### Flicker Test

| Operation | Status | Notes |
|-----------|--------|-------|
| Hub → Browse → Hub | ✅ Zero flicker | Fully unified |
| Hub → Create Shelf → Hub | ✅ Zero flicker | Just completed |
| Hub → Shelve → Hub | ❌ Has flicker | Still uses exit-restart |
| Hub → Edit → Hub | ❌ Has flicker | Still uses exit-restart |
| Hub → Move → Hub | ❌ Has flicker | Still uses exit-restart |
| Hub → Delete → Hub | ❌ Has flicker | Still uses exit-restart |
| Hub → Cache Clear → Hub | ❌ Has flicker | Still uses exit-restart |

### Cancel Behavior

| Operation | Status | Notes |
|-----------|--------|-------|
| Cancel in browse | ✅ Instant return | Fully unified |
| Cancel in create-shelf | ✅ Instant return | Just fixed |
| Cancel in shelve | ❌ Terminal drop | Needs migration |
| Cancel in edit | ❌ Terminal drop | Needs migration |
| Cancel in move | ❌ Terminal drop | Needs migration |
| Cancel in delete | ❌ Terminal drop | Needs migration |
| Cancel in cache-clear | ❌ Terminal drop | Needs migration |

### CLI Compatibility ✅
- [x] All commands work standalone
- [x] Scripts and automation unaffected
- [x] Both TUI and CLI paths functional
- [x] `shelfctl init` works unchanged

---

## 📋 Definition of Done - IN PROGRESS

The unified TUI migration will be complete when:

- [x] Core infrastructure (orchestrator, navigation, messages)
- [x] Hub view with zero flicker
- [x] Browse view with zero flicker
- [x] Create-shelf view with zero flicker ✅ JUST COMPLETED
- [ ] **Shelve (add books) migrated to unified view**
- [ ] **Edit books migrated to unified view**
- [ ] **Move books migrated to unified view**
- [ ] **Delete books migrated to unified view**
- [ ] **Cache clear migrated to unified view**
- [x] Non-TUI commands use suspend pattern appropriately
- [x] CLI mode 100% backward compatible

**Current Status:** 40% complete (3/8 operations unified)

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
