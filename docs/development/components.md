# Reusable Bubble Tea Components

shelfctl's TUI components have been extracted to five standalone Go modules. Each handles a common Bubble Tea pattern with production-tested behavior and consistent keyboard shortcuts.

## Components

| Component | Package | Purpose | LOC |
|-----------|---------|---------|-----|
| **Base Picker** | [bubbletea-picker](https://github.com/blackwell-systems/bubbletea-picker) | Foundation for selection UIs — handles key bindings, resize, borders, error management | ~200 |
| **Multi-Select** | [bubbletea-multiselect](https://github.com/blackwell-systems/bubbletea-multiselect) | Checkbox wrapper for any `list.Item` with persistent selection state | ~300 |
| **Miller Columns** | [bubbletea-millercolumns](https://github.com/blackwell-systems/bubbletea-millercolumns) | Hierarchical column navigation (macOS Finder style) | ~400 |
| **Carousel** | [bubbletea-carousel](https://github.com/blackwell-systems/bubbletea-carousel) | Peeking card layout with centered active card and adjacent previews | ~350 |
| **Command Palette** | [bubbletea-commandpalette](https://github.com/blackwell-systems/bubbletea-commandpalette) | Fuzzy-search action overlay (VS Code Ctrl+P style) | ~365 |

## Installation

```bash
go get github.com/blackwell-systems/bubbletea-picker
go get github.com/blackwell-systems/bubbletea-multiselect
go get github.com/blackwell-systems/bubbletea-millercolumns
go get github.com/blackwell-systems/bubbletea-carousel
go get github.com/blackwell-systems/bubbletea-commandpalette
```

Dependencies are limited to official Bubble Tea libraries (`bubbles`, `bubbletea`, `lipgloss`, `x/ansi`) plus `sahilm/fuzzy` for the command palette.

## Usage in shelfctl

### Base Picker
- **Shelf picker** — select which shelf to operate on
- **Book picker** — select books for edit/delete/move
- Reduces picker boilerplate by ~60%

### Multi-Select
- **Batch delete** — select multiple books to remove
- **Batch move** — select multiple books to relocate
- **Cache clear** — select cached books to remove locally
- **Multi-file shelve** — select multiple PDFs to add in one session
- Selection state persists across list navigation

### Miller Columns
- **File browser** — navigate directories to select files for shelving
- 3 columns visible simultaneously for hierarchy context
- Combined with multi-select for checkbox state across directories

### Carousel
- **Batch edit** — navigate between selected books as cards
- Center card shows full metadata form, adjacent cards peek from edges
- Green border = saved, orange = active, dot indicator for position

### Command Palette
- **Hub quick actions** — `Ctrl+P` opens fuzzy-search over all menu items
- Searches by label, description, and hidden keywords
- Selection emits `NavigateMsg` for instant view switching

## API Documentation

Each component repository contains full API reference, usage examples, and integration guides. See the links in the table above.
