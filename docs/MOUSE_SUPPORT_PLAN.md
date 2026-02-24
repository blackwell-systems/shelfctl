# Mouse Support for shelfctl TUI

## Context

The shelfctl TUI currently only supports keyboard navigation. Adding mouse support (click to select, scroll wheel, click footer shortcuts) makes the TUI more intuitive and accessible. bubbletea v1.3.10 supports mouse events via `tea.WithMouseCellMotion()`. The bubbles `list.Model` has NO built-in mouse handling, so we implement it ourselves.

---

## Change 1: Enable Mouse in All `tea.NewProgram` Calls

Add `tea.WithMouseCellMotion()` to every `tea.NewProgram` call:

| File | Line | Current |
|------|------|---------|
| `internal/app/root.go` | 505 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/hub.go` | 449 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/list_browser.go` | 1061 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/book_picker.go` | 165 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/book_picker.go` | 411 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/edit_form.go` | 297 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/shelf_create_form.go` | 234 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/shelve_form.go` | 265 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/file_picker.go` | 570 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/progress.go` | 178 | `tea.NewProgram(m, tea.WithAltScreen())` |
| `internal/tui/shelf_picker.go` | 119 | `tea.NewProgram(m, tea.WithAltScreen())` |

Each becomes `tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())`.

---

## Change 2: Mouse Helper — `internal/tui/mouse.go` (new file)

Shared utility to convert mouse Y coordinate to a list item index. The list renders items starting at a Y offset (after title, status bar, filter bar). We need to account for this.

```go
package tui

// ListClickIndex translates a mouse click Y position to the list item index.
// headerLines = number of lines before the first item (title + status + filter + padding).
// Returns the item index (0-based from visible top), or -1 if outside the list.
func ListClickIndex(list list.Model, mouseY int, headerLines int) int {
    itemIndex := mouseY - headerLines
    if itemIndex < 0 {
        return -1
    }
    // Paginator offset: list.Paginator.Page * list.Paginator.PerPage
    absoluteIndex := itemIndex + list.Paginator.Page*list.Paginator.PerPage
    if absoluteIndex >= len(list.Items()) {
        return -1
    }
    return absoluteIndex
}
```

---

## Change 3: Browse List Mouse — `internal/tui/list_browser.go`

In `BrowserModel.Update()`, add a `case tea.MouseMsg:` handler after the existing `case tea.KeyMsg:` block:

```go
case tea.MouseMsg:
    switch msg.Button {
    case tea.MouseButtonWheelUp:
        m.list.CursorUp()
        return m, nil
    case tea.MouseButtonWheelDown:
        m.list.CursorDown()
        return m, nil
    case tea.MouseButtonLeft:
        if msg.Action != tea.MouseActionPress {
            break
        }
        if cmd := m.handleFooterClick(msg.X); cmd != nil {
            return m, cmd
        }
        if idx := listClickIndex(m.list, msg.Y, m.listHeaderLines()); idx >= 0 {
            m.list.Select(idx)
            return m, nil
        }
    }
```

**Footer click handling**: Store rendered shortcut positions (start X, end X, key) in a `[]shortcutRegion` field on `BrowserModel`. Populate in `renderFooter()`. In `handleFooterClick`, find which region contains `mouseX` and dispatch the action.

**`listHeaderLines()` helper**: Returns the number of rendered lines before the first list item. Compute from `list.ShowTitle()`, `list.ShowStatusBar()`, `list.FilterState()`.

---

## Change 4: Hub Menu Mouse — `internal/unified/hub.go`

In `HubModel.Update()`, add `case tea.MouseMsg:` handler:

- Wheel up/down → scroll list or details panel (depending on `m.detailsFocused`)
- Left click on menu item → select + navigate (single click acts as enter)
- Left click on details panel area → focus details

---

## Change 5: Book Pickers Mouse — `internal/tui/book_picker.go`

**Single picker**: Click on item → select it via `m.base.List().Select(idx)`.

**Multi picker**: Click on item → select + toggle checkbox. Wheel → scroll.

---

## Change 6: Edit Form Mouse — `internal/tui/edit_form.go`

Click-to-focus for form fields. `fieldAtY()` maps mouse Y coordinate to field index based on form layout.

---

## Change 7: Shelf Picker Mouse — `internal/tui/shelf_picker.go`

Click selects the shelf item via picker.Base.

---

## Change 8: File Picker Mouse — `internal/tui/file_picker.go`

Uses `millercolumns.Model`. Scroll wheel at minimum; click if millercolumns exposes column positions.

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/tui/mouse.go` | **New file** — `ListClickIndex()` helper, `shortcutRegion` type |
| `internal/app/root.go` | Add `tea.WithMouseCellMotion()` |
| `internal/tui/list_browser.go` | Add `tea.MouseMsg` handler, `handleFooterClick()`, `listHeaderLines()` |
| `internal/unified/hub.go` | Add `tea.MouseMsg` handler for menu click + details scroll |
| `internal/tui/book_picker.go` | Add mouse handlers to both single and multi picker models |
| `internal/tui/edit_form.go` | Add click-to-focus for form fields |
| `internal/tui/shelf_picker.go` | Add click-to-select |
| `internal/tui/file_picker.go` | Add scroll wheel + click if feasible |
| `internal/tui/hub.go` | Add `tea.WithMouseCellMotion()` |
| `internal/tui/shelf_create_form.go` | Add `tea.WithMouseCellMotion()` |
| `internal/tui/shelve_form.go` | Add `tea.WithMouseCellMotion()` |
| `internal/tui/progress.go` | Add `tea.WithMouseCellMotion()` |

---

## Implementation Order

1. `mouse.go` — shared helper
2. All `tea.NewProgram` calls — enable mouse (mechanical, all at once)
3. Browse list — most impactful, click items + scroll + footer clicks
4. Hub menu — click menu items + scroll details
5. Book pickers — click to select/toggle
6. Edit form — click to focus fields
7. Shelf picker — click to select
8. File picker — scroll wheel at minimum

---

## Verification

1. `go build ./...` — must pass
2. `golangci-lint run --timeout=5m` — 0 issues
3. Manual testing:
   - Browse: click a book item → cursor moves to it
   - Browse: scroll wheel → list scrolls
   - Browse: click footer shortcut → action fires + highlight
   - Hub: click menu item → navigates to that view
   - Hub: scroll wheel on details panel
   - Edit form: click a field → focus moves to it
   - Book picker: click item → selects it
   - Multi-select picker: click item → toggles checkbox
   - All views: ensure keyboard still works as before
