# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `shelfctl index --open` — open the generated HTML index in the default browser immediately after generation; OS-aware (`open` on macOS, `xdg-open` on Linux, `rundll32` on Windows)
- Subtle SVG teal texture on HTML index sticky nav header (autumn chevron pattern at 4% opacity)
- **Carousel bulk edit** — press `a` from the carousel to open a bulk-edit overlay; pick an operation (Add tag, Remove tag, Set author, Set year) then type a value; applied to all books in the batch, merging with per-book edits already in progress

## [0.2.4] - 2026-02-24

### Changed
- **Hub menu redesign** — items grouped into labeled sections (Library, Organize, Shelves, Tools, Cache) with `─── Section ───` dividers; two-tone `shelf`/`ctl` wordmark and stat-pill status bar in the unified-mode hub header
- **Hub navigation** — cursor skips section headers automatically; clamped at top/bottom so pressing up from the first item stays put
- **Browse view** — author column capped at 26 characters
- **Refactor**: split `internal/app/root.go` (896 lines) into `init_wizard.go` and `hub_runner.go`; root reduced to 199 lines
- **Refactor**: split `internal/tui/list_browser.go` (1150 lines) into `book_item.go`, `browser_render.go`, and `list_browser.go` (685 lines)
- **Refactor**: split `internal/unified/move_book.go` (915 lines) into `move_book_ops.go` and `move_book_render.go` (435 lines)
- **Refactor**: split `internal/unified/shelve.go` (1217 lines) into `shelve_ops.go` and `shelve_render.go` (716 lines)
- **Refactor**: extracted carousel logic from `edit_book.go` into `carousel.go`; `edit_book.go` reduced from 1030 → 802 lines
- Fixed column header alignment in browse view
- **Brand color theming** applied across all TUI views
  - Added `ColorOrange` (`#fb6820`), `ColorTeal` (`#1b8487`), `ColorTealLight` (`#2ecfd4`), `ColorTealDim` (`#0d3536`) to the shared color palette
  - `StyleHighlight` updated from yellow → orange; `StyleTag` updated from cyan → teal-light; `StyleBorder` border color updated to teal
  - **Hub**: two-tone wordmark; teal-light stat numbers; teal panel divider; orange shelf names and teal book counts in detail pane
  - **Browse**: orange `›` cursor; teal `✓` multi-select; tags with `·` separator in teal; `✓ local` cache indicator; teal column headers
  - **Edit form**: uses shared tui color constants
  - Fixed `padOrTruncate` and `truncateText` to use rune count so multi-byte characters no longer cause column misalignment
- **Bulk Edit Carousel** — fully redesigned as a peeking single-row layout
  - Centered card flanked by adjacent cards peeking in from each side (clipped to ≤ half width)
  - Ghost cards at edges; dot position indicator (`○ ● ○`)
  - Card aspect ratio approximates a 3×5 library card
  - Selecting multiple books opens the carousel immediately
  - `↓`/`Enter` selects the current card and opens its edit form; `Esc` returns to carousel
  - Terminal resize artifacts fixed globally via `lipgloss.Place`
  - Fixed panic when `Esc` pressed before any card was opened

## [0.2.3] - 2026-02-24

### Changed
- **HTML Index Brand Colors**
  - Applied brand color system to the generated HTML library index
  - Two-tone header: "shelf" in orange (`#fb6820`), "ctl" in teal (`#2ecfd4`), matching the wordmark
  - Shelf section titles and active tag filters use orange
  - Tag pills, focus rings, and filter count use teal-light
  - Book card backgrounds and borders have subtle teal tint for depth
  - Card hover states lift with orange border and shadow
  - CSS variables at `:root` for maintainability

### Added
- **Tight Wordmark Crop** (`assets/wordmark-tight.png`)
  - Tightly cropped version of the wordmark with 30px transparent padding
  - 905×242px — suitable for compact contexts (README badges, social headers)

## [0.2.2] - 2026-02-23

### Added
- **`shelfctl tags` Command**
  - `shelfctl tags` / `shelfctl tags list` — list all tags with book counts, sorted by frequency
  - `shelfctl tags rename <old> <new>` — bulk rename a tag across all books and shelves
  - `--shelf` flag to scope to a single shelf, `--dry-run` for preview, `--json` for output
- **`shelfctl completion` Command**
  - Generate shell autocompletion scripts (bash, zsh, fish, powershell)
  - Was documented but not registered; now wired up
- **`shelfctl search` Command**
  - Full-text search across title, author, and tags (case-insensitive)
  - `--tag`, `--format`, `--shelf` flags to narrow results
  - `--json` flag for machine-readable output
  - Uses existing `catalog.Filter` engine
- **`shelfctl status` Command**
  - Per-shelf summary: book counts, cached count, modified count
  - `--verbose` flag for per-book status lines (cached/modified/remote)
  - `--shelf` flag to filter to a specific shelf
  - `--json` flag for machine-readable output

### Documentation
- Added `verify` command reference to `docs/reference/commands.md`

### Fixed
- **Footer Command Highlighting on All TUI Screens**
  - 500ms key-press highlight now works on shelve, edit book, delete book, move book, cache clear, and create shelf views
  - Previously only worked on the browse screen; other views had hardcoded footer strings
  - Wired `SetActiveCmd`/`ClearActiveCmdMsg`/`RenderFooterBar` into all 6 unified views

## [0.2.1] - 2026-02-23

### Added
- **Column Headers on Browse and Picker Screens**
  - TITLE, AUTHOR, TAGS, SHELF, CACHE headers displayed above list items
  - Underlined header row updates dynamically on terminal resize
  - Applied to: browse list, edit book picker, delete book picker, cache clear picker, move book picker
- **Shelf-Aware Browse Title**
  - Browse screen title now shows "Shelf: \<name\>" when viewing a single shelf
  - Shows "All Shelves" when viewing books across multiple shelves

### Changed
- **Compact Title Column**
  - Title column capped at 48 characters max width to keep the display compact
  - Longer titles truncate with ellipsis; extra space redistributed to other columns
- **Shelf Creation Form: Auto `shelf-` Prefix**
  - Repository name field now shows `shelf-` as a fixed prefix
  - Users only type the suffix (e.g., "programming" instead of "shelf-programming")
  - Reduces confusion and enforces naming convention
- **Documentation Reorganized into Subdirectories**
  - Moved all docs into `guides/`, `reference/`, and `development/` subdirectories
  - Renamed files to lowercase kebab-case for consistency
  - Deleted stale `nav.yml` (mkdocs.yml is canonical)
  - Updated all cross-references in markdown, YAML config, and Go source
- **TUI Architecture Docs Rewritten**
  - Rewritten as a clean post-migration architecture reference
  - Removed all migration history, phase tracking, and old/new comparisons
- Removed `docs/UNIFIED_TUI_PLAN.md` (migration complete)

## [0.2.0] - 2026-02-23

### Added
- **Footer Command Highlight on All TUI Pages**
  - When a shortcut key is pressed, the corresponding label in the footer briefly highlights (500ms)
  - Shared `RenderFooterBar` and `SetActiveCmd` utilities in `internal/tui/footer.go`
  - Applied to: browse, hub, edit book, delete book, cache clear, move book, create shelf, shelve form, standalone edit form, standalone shelve form
- **Mouse Support Plan** — documented in `docs/development/mouse-support-plan.md`

### Changed
- Browse detail panel now off by default (toggle with tab)
- Browse footer label changed from "tab toggle" to "tab detail toggle"
- Consolidated docs: removed duplicate root `CHANGELOG.md` and `CONTRIBUTING.md` (canonical copies live in `docs/`), moved `UNIFIED_TUI_PLAN.md` to `docs/`
- CI updated to checkout bubbletea-components sibling repo and use Go 1.25
- **MAJOR: Unified TUI Architecture (Zero Flicker)**
  - Complete redesign of interactive mode to eliminate screen flicker
  - **Old architecture:** Each operation launched a separate Bubble Tea program
    - Hub → Browse = 2 screen clears (exit hub, enter browse)
    - Visible flash/flicker on every navigation
    - Terminal state reset between operations
    - Felt like switching between separate applications
  - **New architecture:** Single persistent Bubble Tea program with internal view switching
    - Hub → Browse = instant state transition (zero flicker)
    - Alt screen entered once and persists throughout session
    - Seamless navigation feels like native application
    - View state preserved when navigating away and back
  - **Implementation:** Orchestrator pattern with message-based navigation
    - `internal/unified/model.go` coordinates all views (~850 lines)
    - `NavigateMsg` for view switching, `QuitAppMsg` for exit
    - `ActionRequestMsg` for operations requiring TUI suspension
    - Each view is a self-contained model that emits navigation messages
  - **Suspend-and-resume pattern:** Forms and external commands temporarily exit alt screen
    - Create shelf form, import repository, cache info output
    - Screen clears only when REQUIRED, not on every navigation
    - Automatically resumes unified TUI after external operation completes
  - **CLI compatibility:** Unchanged - scripts and direct commands work identically
    - `shelfctl browse` runs standalone (old behavior)
    - `shelfctl` (no args) runs unified TUI (new behavior)
  - **Technical tradeoffs:**
    - Added ~3000 lines of orchestration code
    - Higher memory usage (~10-15 MB vs 2-5 MB per view)
    - More complex architecture and debugging
    - Worth it: UX improvement is dramatic and user-facing
  - **Performance:** Instant view transitions (<1ms) vs old visible flicker (~200-500ms)
  - See [tui-architecture.md](tui-architecture.md) for complete technical deep dive
  - All features migrated with 100% feature parity - no functionality lost

### Added
- **Cache Clear Unified View (Zero Flicker)**
  - Cache clear now runs entirely within the unified TUI (no terminal drop)
  - Three-phase workflow: book picker → confirmation → processing
  - Modified book protection: skipped books shown in confirmation screen
  - Shows space to be freed in human-readable format
  - Esc returns instantly to hub (zero flicker, no screen clear)
  - Full feature parity with CLI `cache clear` command
- **Delete Book Unified View (Zero Flicker)**
  - Delete book now runs entirely within the unified TUI (no terminal drop)
  - Three-phase workflow: book picker → confirmation → processing
  - Destructive action warnings with red styling and "THIS CANNOT BE UNDONE"
  - Full deletion: removes asset from GitHub release, updates catalog, clears cache, updates README
  - Esc returns instantly to hub (zero flicker, no screen clear)
  - Full feature parity with CLI `delete-book` command
- **Edit Book Unified View (Zero Flicker)**
  - Edit book now runs entirely within the unified TUI (no terminal drop)
  - Multi-phase workflow: book picker → edit form (per book) → batch commit
  - Embedded text inputs for title, author, year, tags with tab navigation
  - Sequential editing: for multi-select, shows form for each book with [N/M] progress
  - Batch commit optimization: one catalog commit per shelf (not per book)
  - README automatically updated with edited metadata
  - Esc returns instantly to hub (zero flicker, no screen clear)
  - Full feature parity with CLI `edit-book` command
- **Shelve (Add Book) Unified View (Zero Flicker)**
  - Add book now runs entirely within the unified TUI (no terminal drop)
  - Seven-phase workflow: shelf picker → file picker → setup → metadata ingestion → edit form → upload → commit
  - Miller columns file picker embedded directly in unified TUI
  - PDF metadata extraction pre-fills form defaults (title, author)
  - Channel-based upload progress with inline progress bar
  - Embedded text inputs for title, author, tags, ID, and cache checkbox
  - Duplicate checking and asset collision handling
  - Batch catalog commit and README update per shelf
  - Esc returns instantly to hub from any phase (zero flicker, no screen clear)
  - Full feature parity with CLI `shelve` command
- **Move Book Unified View (Zero Flicker)**
  - Move book now runs entirely within the unified TUI (no terminal drop)
  - Five-phase workflow: book picker → type picker → destination picker → confirmation → processing
  - Supports cross-shelf moves: downloads asset → uploads to destination → deletes original → updates both catalogs → clears cache → updates READMEs → migrates covers
  - Supports same-shelf release moves: downloads asset → uploads to new release → deletes original → updates catalog
  - Multi-select book picker for batch moves
  - Shelf picker for cross-shelf destination, text input for release destination
  - Esc returns instantly to hub from any phase (zero flicker, no screen clear)
  - Full feature parity with CLI `move` command
- **Shared README Operations Package**
  - Extracted `UpdateShelfREADMEStats`, `AppendToShelfREADME`, `RemoveFromShelfREADME` to `internal/operations/readme.go`
  - Shared between CLI (`app` package) and unified TUI (`unified` package)
  - Eliminates import cycle between packages
- **Create Shelf in TUI**
  - New "Create Shelf" option in hub menu for adding shelves interactively
  - Interactive form collects shelf name, repository name, and configuration flags
  - Checkboxes for: create repository, make private
  - Creates repo via GitHub API with 'library' release, generates README.md, and adds shelf to config
  - Release is always created when repo is created (simplifies workflow, release is required anyway)
  - No need to exit to CLI for shelf creation workflow
  - Returns to hub menu after completion
- **Scrollable Details Panel in Hub**
  - Hub view now supports scrolling when viewing shelves or cache info with many items
  - Press `tab` or `→` to focus the details panel (right side)
  - Use `↑`/`↓` to scroll through content when focused
  - Press `←` or `esc` to return focus to main menu
  - Visual indicators: thick cyan border when details panel focused, dimmed menu border
  - Scroll position indicator shows "Lines X-Y of Z" at bottom
  - No more 10-book limit on modified books display in cache info
  - Useful when you have many shelves or many locally modified books
- **Toggle Hidden Files in File Picker**
  - Press `.` (dot) in the file picker to show/hide hidden files and directories
  - Hidden files/folders (starting with `.`) are hidden by default
  - Useful for edge cases like `.hidden-book.pdf` or navigating into `.config` directories
  - Toggle state rebuilds all visible columns immediately
  - Help text shows `. show hidden` in the bottom bar

### Changed
- **Simplified Shelf Creation**
  - Removed optional 'create release' checkbox from shelf creation form
  - Release is now always created when creating a new repo (it's required for storing books anyway)
  - Reduces confusion - release will be auto-created on first shelve if missing
  - CLI: removed `--create-release` flag from `shelfctl init` (always creates release with repo)
- **Improved Hub Menu Layout**
  - Increased label width from 20 to 25 characters to accommodate longer menu item names
  - Prevents text collision between labels and descriptions for items like "Import from Repository"
  - Cleaner visual separation throughout menu
- **Hub Menu Navigation Improvement**
  - TUI commands (browse, shelve, edit-book, move, delete-book, cache-clear) now return directly to hub menu
  - No more "Press Enter to return to menu..." prompt after using TUI commands
  - Non-TUI commands (shelves, index, cache-info) still show the prompt as before
  - Smoother workflow when using interactive commands from the hub
  - Reduced screen flicker when transitioning between TUI commands by clearing screen
- **Hub Menu Reordered by Frequency of Use**
  - Most-used commands now at top: Browse, Add Book, Edit Book
  - Cache commands grouped together: Cache Info, Clear Cache
  - Occasional commands in middle: View Shelves, Generate HTML Index, Add from URL
  - Rare commands at bottom: Move, Delete Book, Import, Delete Shelf
  - Faster access via muscle memory (j j enter for common tasks)

### Fixed
- **File Picker Right Arrow Key Behavior**
  - Fixed right arrow key triggering file selection in Miller columns file picker
  - Right arrow and 'l' now only navigate into directories, not select files
  - Enter key continues to work for both opening directories and selecting files
  - Prevents accidental navigation to add screen when browsing PDFs

## [0.1.4] - 2026-02-22

### Added
- **Enhanced Cache Info Display**
  - Cache info now lists which specific books are NOT cached
  - Shows both book ID and title for easy identification
  - Previously only showed count: "⚠ 3 books not cached"
  - Now shows detailed list:
    ```
    ⚠ 3 books not cached:
      - book-one (Book One Title)
      - book-two (Book Two Title)
      - book-three (Book Three Title)
    ```
  - Applies to both `cache info` (all shelves) and `cache info --shelf <name>`
- **Sticky Header Navigation in HTML Index**
  - Header, search/sort controls, and tag filters now stick to top when scrolling
  - Navigation controls remain accessible while browsing large book collections
  - Visual separation with background color, border, and shadow

### Fixed
- **Shelve Form Index Out of Range Panic**
  - Fixed crash when navigating to "Cache locally" checkbox in add book form
  - Root cause: Code tried to update `m.inputs[4]` but array only had 4 elements (0-3)
  - Checkbox is not a text input field and should be skipped during input updates
  - Affected all users adding books via TUI - pressing tab/arrow keys would crash
  - Now safely navigates and toggles checkbox without panic
- **Cover Image Not Generated for Auto-Cached Books**
  - Fixed redundant ExtractCover() call that was deleting successfully generated covers
  - Books cached during upload (via --cache flag or TUI checkbox) now properly get covers
  - Store() already handles cover extraction, redundant second call was problematic
- **Cover Extraction Compatibility with pdftoppm 26.x**
  - Fixed cover generation for newer pdftoppm versions (26.x) that use 4-digit padding
  - ExtractCover() now tries all formats: -0001.jpg, -001.jpg, -1.jpg
  - Previously failed silently when pdftoppm created -0001.jpg but code expected -001.jpg
  - Fixes cover extraction for all PDFs when using recent poppler installations

## [0.1.3] - 2026-02-22

### Added
- **Homebrew Installation Support**
  - Added official Homebrew tap: `brew install blackwell-systems/tap/shelfctl`
  - Pre-built binaries for macOS (Intel/ARM) and Linux (x86_64/ARM64)
  - Formula available at [homebrew-tap](https://github.com/blackwell-systems/homebrew-tap)
- **Cache Management**
  - New `cache` command with `clear` and `info` subcommands
  - CLI: `shelfctl cache clear [book-id...]` to remove specific books from cache
  - CLI: `shelfctl cache clear --shelf books` to clear entire shelf cache
  - CLI: `shelfctl cache clear --all` to nuke entire cache (with confirmation)
  - CLI: `shelfctl cache clear` for interactive picker (multi-select)
  - CLI: `shelfctl cache info` shows cache statistics and lists which books have local changes
  - TUI: 'x' key removes selected books from cache (supports multi-select)
  - Hub menu: Added "Cache Info" and "Clear Cache" options
  - Cache Info shows details panel with modified book list when highlighted
  - Books remain in catalog/release, only local cache cleared
  - Useful for reclaiming disk space without affecting library
- **HTML Index Enhancements**
  - Added clickable tag filter boxes (word cloud style) above book grid
  - Click tags to filter books (multiple tags combine with AND logic)
  - Each tag shows book count (e.g., "programming 12")
  - Active tags highlight in blue
  - "Clear filters" button appears when tags are active
  - Live count shows filtered results (e.g., "15 books")
  - Works seamlessly with text search
  - Added sort dropdown with options: Recently Added, Title (A-Z), Author (A-Z), Year (Newest/Oldest)
  - Sorting works per-shelf and persists through filtering
- **Cache on Add**
  - New `--cache` flag for `shelve` command to cache book locally after upload
  - TUI add form includes "Cache locally" checkbox (enabled by default)
  - Book is immediately available to open without extra download step
  - Extracts PDF cover during caching if poppler installed
  - CLI: `shelfctl shelve book.pdf --shelf prog --title "..." --cache`
- **Sync Command** (handle annotations/highlights)
  - New `sync` command to upload locally modified books back to GitHub
  - Detects when cached files differ from Release asset (annotations, highlights, etc.)
  - Re-uploads modified file and updates catalog (replaces asset, no versioning)
  - CLI: Supports `sync book-id`, `sync --all`, `sync --all --shelf name`
  - CLI: Progress indicators with "[2/5]" counter and upload progress bar for each book
  - TUI: Press 's' in browse view to sync selected books (or current if none selected)
  - TUI: Shows progress message during sync operation
  - `cache info` lists modified books by ID and title: "sicp (Structure and Interpretation...)"
  - Hub Cache Info panel displays modified books with hint to press 's' in browse or run sync
  - Keeps GitHub in sync with your working copies without version clutter
- **Cache Clear Protection**
  - Modified files (with annotations/highlights) are now protected by default
  - `cache clear` skips modified files and shows warning with sync suggestion
  - New `--force` flag to delete modified files if absolutely needed
  - Applies to all clear operations: specific books, shelf, all, interactive
  - Prevents accidental loss of annotations when reclaiming disk space
- **Multi-Select Operations in Browse**
  - Spacebar to toggle selection (checkboxes appear in list)
  - 'g' downloads all selected books to cache (or current if none selected)
  - 'x' removes selected books from cache (or current if none selected)
  - 's' syncs selected modified books to GitHub (or current if none selected)
  - 'c' to clear all selections
  - Downloads happen in background without exiting TUI
  - Progress bar shows at bottom of screen
  - Multiple downloads queue sequentially
  - Books marked as cached when complete
  - Error isolation: failed downloads don't abort batch
  - Useful for pre-caching books for offline reading

### Changed
- **Cleaner Browse TUI**
  - Removed ID column from browse list view (increased title width from 50 to 66 chars)
  - Removed ID field from details panel (not needed in visual browser)
  - Removed Asset filename from details panel (implementation detail clutter)
  - More focus on book content, less on technical identifiers
  - Checkboxes now appear when books are selected for batch operations
- **Simplified Shelf README Template**
  - Removed placeholder Organization sections (By Topic, Reading Lists, Favorites)
  - Cleaner default template, less intrusive on personal repos
  - Still includes Quick Stats, Recently Added, and maintenance sections
- **Improved File Picker Filtering**
  - Fuzzy search now prioritizes filename matching over full path
  - Type part of filename to quickly find files (e.g., "sicp" finds "structure-and-interpretation.pdf")
  - Press '/' to activate filter, results update as you type

### Fixed
- **ID Generation and Validation**
  - Fixed slugify to trim trailing hyphens after truncating to 63 characters
  - Fixed shelve form to return original default values instead of truncated placeholders with "…"
  - Prevents validation errors for long filenames like "how-linux-works-what-every-superuser-should-know-3rd-edition.pdf"
- **Concurrent README Edits**
  - Added `git pull --rebase` before push in CommitFile to handle remote edits
  - Automatically merges README changes made via GitHub UI with local shelfctl updates
  - Prevents push failures when editing README remotely during operations

## [0.1.2] - 2026-02-22

### Added
- **Verify Command for Catalog/Release Sync**
  - New `verify` command detects catalog vs release mismatches
  - Finds orphaned catalog entries (book in catalog but asset missing from release)
  - Finds orphaned assets (asset in release but not referenced in catalog)
  - Read-only by default - shows detailed issue report
  - Auto-fix mode with `--fix` flag:
    - Removes orphaned catalog entries and commits updated catalog
    - Deletes orphaned assets from release to reclaim storage
    - Clears local cache for removed books
    - Updates README.md with new book counts
  - Supports `--shelf` flag to verify specific shelf only
  - Processes all shelves by default
  - Use case: Recover from manual asset deletions or interrupted operations
  - Similar to `git fsck` or `npm audit` for library integrity checks
- **Batch Metadata Edit with Multi-Select**
  - Enhanced `edit-book` command now supports editing multiple books at once
  - Multi-select book picker (spacebar to toggle, enter to confirm)
  - Batch confirmation: "Type 'UPDATE N BOOKS' to confirm" for safety
  - Individual edit forms for each selected book (allows different values)
  - Progress indicators show `[2/5] Editing book-id` during batch processing
  - Optimized commits: one catalog commit per shelf (not per book)
  - Error isolation: failed edits don't abort the batch
  - Summary output: "Successfully updated 5 books, 1 failed"
  - Works across multiple shelves (groups by shelf for commits)
  - Backward compatible: CLI mode unchanged for single book edits
  - Use case: Batch tag additions, author name fixes, year corrections

### Changed
- **Hub Menu with Inline Shelf Preview**
  - "View Shelves" now shows shelf information in split-panel on the right when highlighted
  - No need to exit hub menu - details appear automatically as you navigate
  - Shows shelf name, repository, book count, and status for each shelf
  - Navigate away to hide panel, Enter to execute full "View Shelves" command
  - Details panel constrained to fixed 45-character width (doesn't expand on wide terminals)
  - Smoother UX without context switching or "Press Enter" pauses
- **Enhanced Browse Details Panel**
  - Added file size display in human-readable format (e.g., "7.6 MiB", "21.9 MiB")
  - Added year field when available
  - All text fields now truncate with ellipsis to prevent wrapping into left pane
  - Dynamically adjusts on terminal resize
  - Affects long IDs, titles, authors, repository names, asset names

### Fixed

## [0.1.1] - 2026-02-22

### Added
- **Miller Columns File Browser (Hierarchical View)**
  - File picker now uses Miller columns layout (like macOS Finder column view)
  - Multiple directory levels displayed side-by-side for visual hierarchy
  - Enter/Right/L: Navigate into selected directory (adds column to the right)
  - Backspace/Left/H: Go back to parent level (removes rightmost column)
  - Tab/Shift+Tab: Switch focus between visible columns
  - Shows up to 3 columns at once, automatically adjusts to terminal width
  - Focused column highlighted with cyan border, unfocused columns dimmed
  - Faster navigation - see multiple levels at once without repeated backspace
- **Multi-File Selection for Shelve Command (TUI Mode)**
  - Select multiple files using checkboxes in the file picker
  - Spacebar to toggle selection, enter to confirm
  - Checkbox state persists across directory navigation and columns
  - Can select files from multiple directories in one session
  - Progress indicators show `[2/5] filename.pdf` during batch processing
  - Individual metadata forms for each file in the batch
  - Batch commit optimization: single catalog commit + single README commit for all files
  - Error isolation: one failed file doesn't abort the entire batch
  - Summary output: "Successfully added 3 books, 1 failed"
  - Backward compatible: CLI mode unchanged, single file selection still works
- **Batch Delete with Multi-Select (TUI Mode)**
  - Select multiple books to delete using checkboxes in the book picker
  - Spacebar to toggle selection, enter to confirm
  - If no checkboxes selected, deletes the currently highlighted book
  - Batch confirmation: "Type 'DELETE N BOOKS' to confirm" for multiple books
  - Progress indicators show `[2/5] Deleting book-id …` during processing
  - Error isolation: failed deletions don't abort the batch
  - Summary output: "Successfully deleted 4 books, 1 failed"
  - Backward compatible: CLI mode (`shelfctl delete-book book-id`) unchanged
- **Batch Move with Multi-Select (TUI Mode)**
  - Select multiple books to move using checkboxes in the book picker
  - Spacebar to toggle selection, enter to confirm
  - Single destination selection applies to all selected books
  - Validation: prevents moving books from different shelves to different release (must use different shelf option)
  - Progress indicators show `[3/8] Moving book-id …` during processing
  - Error isolation: failed moves don't abort the batch
  - Summary output: "Successfully moved 5 books, 2 failed"
  - Backward compatible: CLI mode (`shelfctl move book-id --to-shelf foo`) unchanged
- **Reusable TUI Components** (Extracted to External Package)
  - Created `millercolumns` package: Generic Miller columns layout component
    - Pure layout/rendering component for hierarchical views
    - Works with any model type (list.Model, multiselect.Model, custom models)
    - Manages column stack, focus, and responsive width allocation
    - Zero dependencies on shelfctl internals - only requires `lipgloss`
    - Extracted to github.com/blackwell-systems/bubbletea-components
    - Fully documented with examples
  - Created `multiselect` package: Generic multi-select wrapper for any bubbles list
    - Checkbox UI with state persistence
    - Customizable appearance
    - Works with any `list.Item` implementation
    - Extracted to github.com/blackwell-systems/bubbletea-components
    - Fully documented with examples
  - Created `picker` package: Base picker component reduces boilerplate by 60-70%
    - Standard key handling (quit, select)
    - Window resize handling
    - Border rendering
    - Error handling
    - Custom behavior via handlers
    - Extracted to github.com/blackwell-systems/bubbletea-components
  - Standard key bindings in `keys.go`:
    - `PickerKeys` - Basic selection (quit, select)
    - `NavigablePickerKeys` - Selection with navigation (quit, select, back)
    - `MultiSelectPickerKeys` - Multi-selection (quit, select, toggle, back)
    - `FormKeys` - Form input (quit, submit, next, prev)
    - Consistent across all TUI components
  - Architecture guide in `internal/tui/ARCHITECTURE.md`
- **MkDocs Material Documentation Site**
  - Professional documentation site at https://blackwell-systems.github.io/shelfctl/
  - Material theme with search functionality, dark/light mode toggle, and mobile-responsive design
  - Automatic deployment via GitHub Actions on pushes to main branch
  - Symlinked CONTRIBUTING.md and CHANGELOG.md to avoid content duplication
  - Navigation includes Getting Started, Reference, and Development sections
  - Searchable documentation with syntax highlighting for code blocks
  - Source editing links to GitHub for all pages

- **Improved Book Picker Display**
  - Book pickers (edit, delete, move) now show tags alongside book information
  - Display format: `book-id  Book Title [tag1,tag2] [local] [shelf-name]`
  - Cached books show `[local]` indicator in green to quickly identify downloaded books
  - Consistent with list browser display for better visual recognition
  - Helps users identify books by tags without opening details panel
- **Quick Edit from Browse**
  - Added `e` key binding in browse mode to directly open edit form
  - Streamlines workflow: browse → press `e` → edit metadata → save
  - No need to quit browse, navigate menu, find book again
  - Shows updated book info immediately after saving
  - Works with full edit form (title, author, year, tags)

### Changed
- **Refactored TUI Pickers to Use Base Components**
  - `shelf_picker.go` now uses `picker.Base` (reduced from 163 to 133 lines)
  - `book_picker.go` now uses `picker.Base` (reduced from 143 to 138 lines)
  - `file_picker.go` uses `multiselect` component and `MultiSelectPickerKeys`
  - All pickers now share consistent key bindings and behavior
  - Eliminated duplicate key handling, window resize, and border rendering code
  - Easier to maintain and extend with future picker components
- **Introduced Base Delegate Component**
  - New `delegate/base.go` eliminates boilerplate in all list delegates
  - All 5 delegates now use `delegate.New()` or `delegate.NewWithSpacing()`
  - Reduced delegate code by ~75 lines across the codebase
  - Only rendering logic needs to be specified, height/spacing/update are automatic
- **Catalog Manager for Centralized Operations**
  - New `catalog/manager.go` consolidates load → parse → modify → marshal → commit pattern
  - Provides high-level operations: `Load()`, `Save()`, `Update()`, `Append()`, `Remove()`, `FindByID()`
  - Used in 7+ files (shelve, delete_book, edit_book, move, import, migrate, browse)
  - Eliminates ~105 lines of repetitive catalog loading/saving code
  - Centralized error handling and consistent error messages
- **README Updater for Shelf Documentation**
  - New `readme/updater.go` consolidates README.md update operations
  - Methods: `AddBook()`, `AddBooks()`, `RemoveBook()`, `UpdateStats()`, `UpdateWithStats()`
  - Replaces 4 separate README functions with single consistent interface
  - Saved ~80 lines of duplicate README update code
  - Consistent commit messages and error handling

### Fixed
- **Progress UI Hanging at 0% and Not Completing**
  - **Root cause**: HTTP client buffers entire file before upload, causing all progress updates to happen at once, overwhelming the channel
  - **Progress throttling**: ProgressReader now sends updates every 1MB instead of every 16KB, reducing messages from ~1060 to ~17 for a 17MB file
  - **Increased channel buffer**: Raised from 10 to 50 to handle burst updates without dropping
  - **Removed timeout race condition**: `waitForProgress` now blocks on channel indefinitely instead of using 100ms timeout that could miss final messages
  - **Separate tick for UI responsiveness**: Independent tickCmd keeps UI refreshing while waiting for progress
  - **Result**: Progress bar updates smoothly and exits cleanly when complete
  - Shows "Connecting to GitHub..." message before progress bar to explain initial delay during connection establishment
  - Ctrl+C properly cancels uploads/downloads and returns cancellation error
- **Multi-Select Book Picker Only Showing One Item**
  - Multi-select book picker now properly handles window resize events
  - Previously displayed only one item at a time due to 0x0 list dimensions
  - Now calculates proper dimensions accounting for border frame size
  - Affects delete-book and move-book commands in TUI mode
- **Batch Delete Confirmation Always Failing**
  - Fixed confirmation prompt to read full phrase "DELETE N BOOKS"
  - Previously used `fmt.Scanln()` which stopped at first space, only capturing "DELETE"
  - Now uses `bufio.ReadString('\n')` to read complete line including spaces
  - Batch deletions now work correctly after user confirmation
- **README Update Warnings for Unchanged Content**
  - Skip README commits when content hasn't actually changed
  - Eliminates spurious "nothing to commit, working tree clean" warnings
  - Affects updateREADMEAfterRemove, updateREADMEAfterAdd, and updateREADME functions
  - Books that aren't in "Recently Added" section no longer trigger warnings
- **Apostrophes and Quotes in Book IDs**
  - slugify() now removes apostrophes and quotes instead of converting them to hyphens
  - "Let's Go Further" → "lets-go-further" (was: "let-s-go-further")
  - Handles ASCII quotes (', "), smart quotes (' ' " "), and curly quotes
  - Added test cases for apostrophes, quotes, and possessives
- **UTF-16BE Encoded PDF Metadata Not Decoded Properly**
  - PDF metadata with UTF-16BE encoding (starting with BOM FE FF) now decodes correctly
  - Added UTF-16BE detection and decoding in `decodePDFString()` function
  - Fixes rendering issues like `��` appearing before book titles extracted from PDFs
  - Affects titles like "Let's Go Further" which are UTF-16BE encoded in PDF metadata
  - Added defensive `sanitizeForTerminal()` for converting problematic Unicode characters to ASCII equivalents

## [0.1.0] - 2026-02-21

### Added
- **Styled Table Output for Shelves Command**
  - Beautiful table format with box-drawing characters
  - Color-coded status indicators (✓ Healthy, ⚠ Warning, ✗ Error)
  - Shows shelf name, repository, book count, and status at a glance
  - Dynamic column width adjustment
  - Cyan bold headers for improved readability
- **Duplicate Book ID Warning**
  - Warns when a book ID exists in multiple shelves
  - Shows which shelf is being used and lists all matches
  - Prompts user to use `--shelf` flag to disambiguate
  - Affects: info, open, edit-book, delete-book, move commands
- **Stylish Colored Help Output**
  - Added colors to all CLI help screens for better readability
  - Cyan bold section headers (Usage, Flags, Commands)
  - Green command names
  - Yellow help hints
  - Respects `--no-color` flag
  - Applied recursively to all commands and subcommands
- **CLI Accessibility Audit Documentation**
  - Comprehensive CLI_ACCESSIBILITY_AUDIT.md document
  - Confirms 99% of functionality is scriptable
  - Documents the one exception (split command requires interactive input)
  - Includes testing checklist and automation examples
- **Repository Configuration**
  - Added CODEOWNERS file: automatic review requests for @blackwell-systems on all PRs
  - Added .gitignore: excludes bin/, IDE files (.idea/, .vscode/), build artifacts
  - Removed bin/shelfctl from git tracking (now in .gitignore)
- **Enhanced Shelves Command**
  - Now displays book count for each shelf in output
  - Shows "(empty)" for shelves with no books
  - Shows "(N books)" for shelves with books
  - Added to hub menu as "View Shelves"
  - Accessible from interactive hub interface
  - Provides quick overview of entire library organization
- **Enhanced Move Command**
  - Full interactive workflow when called with no arguments
  - Book picker to select which book to move
  - Choice between moving to different shelf or different release
  - Shelf picker for cross-shelf moves (excludes current shelf)
  - Optional release tag specification
  - Confirmation summary before executing move
  - Added to hub menu between "Edit Book" and "Delete Book"
  - Comprehensive edge case handling:
    - Detects and prevents no-op moves (same location)
    - Warns when destination has book with same ID (replaces existing)
    - Clears local cache (file path changes after move)
    - Updates README.md on both source and destination shelves
    - Migrates catalog cover images between repos
    - Updates "Quick Stats" and "Recently Added" sections
  - Works for both same-shelf (release-to-release) and cross-shelf moves
  - Preserves --dry-run and --keep-old flags for testing and backup scenarios
- **Local HTML Index for Web Browsing**
  - New `shelfctl index` command generates browsable index.html
  - Located at `~/.local/share/shelfctl/cache/index.html`
  - Features:
    - Visual book grid with covers and metadata
    - Real-time search/filter by title, author, or tags
    - Organized by shelf sections
    - Click books to open with system viewer (file:// links)
    - Responsive layout for mobile/desktop
    - Dark theme matching shelfctl aesthetic
  - Works without running shelfctl - just open in any browser
  - Shows only cached books (download books first to include them)
  - Added to hub menu as "Generate HTML Index"
  - Browser compatibility notes: Safari works best, Chrome/Edge block file:// links
- **Cover Image Support (Two Types)**
  - **Auto-Extracted Thumbnails**: Automatically extracts cover from first page of PDFs during download
    - Stores in cache at `<repo>/.covers/<book-id>.jpg` (only 1 per book)
    - Uses `pdftoppm` from poppler-utils (silently skips if not installed)
    - Thumbnails ~200x300px, under 200KB
    - Local only (not in catalog or git)
  - **Catalog Covers**: Displays user-curated covers from catalog.yml
    - Downloads from git repo when browsing (cached at `<book-id>-catalog.jpg`)
    - Higher priority than extracted thumbnails
    - Portable across machines via git
  - **Display in TUI**:
    - Shows 📷 camera emoji in browser list when any cover exists
    - Displays inline cover in details pane on Kitty/Ghostty/iTerm2 terminals
    - Prioritizes catalog cover > extracted thumbnail > none
    - Note: Inline images don't work through tmux - run directly in terminal
  - **User-Friendly Discovery**:
    - One-time hint shown after first PDF download if poppler not installed
    - Platform-specific guidance (macOS: brew install, Linux: package manager)
    - Hint never repeats (marker file: `~/.config/shelfctl/.poppler-hint-shown`)
    - Silent if already installed - zero friction
  - Both cover types removed automatically when book deleted from cache
  - Overwrites existing covers on re-download
  - **Display in HTML Index**:
    - Cover thumbnails shown in book grid
    - Click anywhere on book card (including cover) to open with system viewer
- **Shell Completion**
  - Cobra-generated completion scripts for bash, zsh, fish, and powershell
  - `shelfctl completion [shell]` command
  - Completes commands, flags, and configured shelf names
  - Installation instructions in docs

### Changed
- **Documentation: Installation and Authentication**
  - Clarified GitHub CLI (gh) is optional, not required
  - Added Prerequisites section to README and TUTORIAL
  - Restructured authentication section with two clear options:
    - Option A: Using gh CLI (optional convenience)
    - Option B: Manual token
  - Documented both classic PAT scopes and fine-grained PAT permissions
  - Added security note: tokens never stored in config (only environment variable name)
  - Added API rate limits note (5,000 requests/hour authenticated)
- **Documentation: Core Value Proposition**
  - Enhanced git history teaching with clear explanation:
    - "Once PDFs land in git history, every clone stays heavy forever. Even after you delete the files, git history never forgets."
  - Added warning callout about deleted files still bloating clones
  - Added comparison table (Git commit vs LFS vs Release assets) with defensible claims
  - Changed cost language to be evergreen (LFS: "Paid storage/bandwidth")
  - Surfaced migration tools as core capability (not hidden feature)
  - Added callout: "Already have PDFs committed in git? shelfctl can scan and migrate them"
  - Added "Two ways to use it" section (Interactive TUI + Scriptable CLI)
- **Documentation: Shelby the Mascot**
  - Introduced Shelby the Shelf mascot throughout README
  - Updated all image alt text to reference Shelby
  - Added centered introduction: "Meet Shelby, your friendly library assistant."
  - Added shelby5.png (600px) above Install section
  - Added Credits section explaining Shelby character
  - Made Optional PDF Cover Thumbnails section collapsible
- **Documentation: Design Philosophy**
  - Added Design Philosophy section explaining domain-specific focus
  - Positioned above Disclaimer in README
  - Expanded in ARCHITECTURE.md with rationale for not generalizing
  - Clear statement: "Does one thing well rather than many things poorly"
- **Documentation Improvements**
  - Added missing flag documentation: `--force`, `--private`, `--app`, `--dry-run`, `--keep-old`
  - Added completion command to all guides
  - Added browser compatibility notes for HTML index file:// links
  - Added troubleshooting section for HTML index clicking issues
  - Removed implementation status tracking from HUB.md (user-focused docs)
  - Added shell completion setup to TUTORIAL.md
  - Removed AI-sounding phrases (em dashes, "Most devs" → "If you use GitHub")
  - Fixed confusing "(via gh)" language to "(via GitHub REST API directly)"
- **Documentation Consolidation**
  - Merged SPEC.md content into ARCHITECTURE.md
  - Catalog and config schemas now in ARCHITECTURE.md reference sections
  - Removed duplicated content (commands, workflows)
  - All references updated across documentation
- **Simplified Cache Structure**
  - Changed from 4-level to 2-level cache directory structure
  - Old: `~/.local/share/shelfctl/cache/<owner>/<repo>/<bookID>/<file>`
  - New: `~/.local/share/shelfctl/cache/<repo>/<file>`
  - Makes browsing cached books easier in Finder/Explorer
  - Reduces unnecessary nesting for personal library use
  - **Migration**: If upgrading from an older version, you can safely delete your old cache directory and books will re-download with the new structure:
    ```bash
    rm -rf ~/.local/share/shelfctl/cache
    ```

### Added
- **Book Edit Command**
  - New `edit-book` command for updating book metadata
  - Interactive TUI form when no flags provided
  - CLI flags for scripted updates: `--title`, `--author`, `--year`, `--add-tag`, `--rm-tag`
  - Pre-populates form with current metadata values
  - Updates catalog.yml in GitHub without touching the asset file
  - Supports adding/removing tags incrementally (no need to re-enter all tags)
  - Added to hub menu as "Edit Book" option
  - Filtered from menu when no books exist
  - Completes full CRUD: Create (shelve), Read (browse/info), Update (edit-book), Delete (delete-book)
  - Editable fields: title, author, year, tags
  - Non-editable fields: ID, format, checksum, asset (tied to the file)
- **Book Deletion Command**
  - New `delete-book` command for removing books from library
  - Interactive book picker when no ID provided
  - Deletes both release asset and catalog entry
  - Clears from local cache automatically
  - Requires confirmation (type book ID to confirm)
  - Added to hub menu as "Delete Book" option
  - Filtered from menu when no books exist
  - Completes CRUD operations: Create (shelve), Read (browse), Update (move), Delete (delete-book)
- **Fully Interactive Book Browser**
  - `browse` command now supports real-time actions:
    - `tab` - Toggle split-panel details view
    - `enter` - Show detailed book information (or perform action if details visible)
    - `o` - Open book (downloads if needed, opens with system viewer)
    - `g` - Download to cache only (for offline access)
    - `q` - Quit browser
  - Split-panel layout shows book metadata in right pane
    - ID, title, author, tags, shelf name, repository, cache status, format, asset name
    - Uses master wrapper pattern: compose panels first, then apply single outer border
    - Cleaner visual separation with vertical divider between panels
    - Responsive sizing based on terminal dimensions
    - Single rounded border wraps entire layout (no double borders)
    - Automatically adjusts list width (60%) and details width (40%) when panel visible
  - Auto-downloads books when opening if not cached
  - Shows download progress with file size
  - Displays full metadata on demand
  - All actions work seamlessly from TUI
- **Interactive Progress Bars**
  - Visual progress tracking for downloads and uploads in TTY mode
  - Shows real-time progress with bytes transferred and percentage
  - Clean progress bar UI using Bubble Tea and bubbles/progress
  - Displays file size and estimated completion
  - Used in:
    - `open` command when downloading books
    - `browse` command when downloading (actions 'o' and 'g')
    - `shelve` command when uploading assets
  - Automatically disabled in non-interactive mode (pipes, redirects, scripts)
  - Preserves existing text output for automation
- **Hub Loop Implementation**
  - Hub menu now loops continuously until user quits
  - Returns to menu after command completion
  - Shows "Press Enter to return to menu..." prompt
  - Reloads config after operations to reflect changes
  - Handles errors gracefully with option to retry
  - No more dropped back to shell after single operation
- **Enhanced Interactive Init**
  - Detects and handles existing repositories gracefully
  - Prompts to use existing repo or enter different name
  - Interactive public/private repository visibility choice
  - Shows visibility in summary before creation
  - Skips README creation if already exists
  - Checks for duplicate shelf names in config
  - Returns to hub menu after successful init (not back to shell)
- **Improved Delete Shelf Flow**
  - Interactive prompt: "What should happen to the GitHub repository?"
  - Clear numbered choices with explanations
  - Option 1: Keep repo (remove from config only)
  - Option 2: Delete permanently (repository and all books)
  - Visual warnings with color-coded dangerous operations
  - Shows exactly what will be deleted before confirmation
- **Smart Hub Menu Filtering**
  - Hides shelf-related options when no shelves configured
  - Hides "Browse Library" when no books exist
  - Hides "Delete Book" when no books exist
  - Shows appropriate first-time setup guidance
  - Menu always shows only available actions
- **Repository Visibility Control**
  - `--private` flag for `init` command (default: true)
  - Interactive prompt during init workflow
  - Clear explanations of public vs private
  - Visibility shown in summary before creation
- **Reusable TUI Components**
  - `tui.RunBookPicker()` - Selectable book list
  - `tui.BrowserAction` - Action result types
  - `tui.BrowserResult` - Structured action results
  - Components can be reused across features
- **PDF Metadata Extraction (Stdlib Only)**
  - Automatically extracts Title, Author, and Subject from PDFs
  - Pre-populates form fields when adding books
  - Uses only Go stdlib (no external dependencies)
  - Best-effort parsing - works for most PDFs with metadata
  - Handles both parentheses format and UTF-16BE hex format
  - Gracefully handles PDFs without metadata (falls back to filename)
  - Supports standard PDF string escaping
  - Zero binary size increase
  - Significantly improves UX for academic papers and published books
  - Tested with real-world PDFs
- **Brand Assets**
  - Created icon-16.png and icon-32.png from shelby-icon.png
  - Generated using ImageMagick for favicon/icon use
  - Sizes: 581B and 1.0K respectively

### Added
- **Comprehensive Architecture Documentation**
  - New `docs/ARCHITECTURE.md` guide covering:
    - Core concepts (shelves, catalogs, assets)
    - Organization philosophy (start broad, split later)
    - When to create multiple shelves vs using tags/releases
    - Naming conventions and best practices
    - Splitting strategies and scaling guidelines
    - Common patterns for different use cases
    - Migration strategies from existing repos
    - Decision trees and GitHub limits
  - Linked from README.md and docs/index.md
  - Referenced inline during interactive init
- **Automatic README.md Generation for Shelves**
  - Each shelf now gets a README.md created automatically during init
  - Template includes:
    - Shelf description and quick stats (book count, last updated)
    - Usage examples for adding and browsing books
    - Organization sections (by topic, reading lists, favorites)
    - Maintenance commands reference
    - Links to documentation and releases
  - Auto-updates on each book addition:
    - Updates book count and last updated date
    - Adds book to "Recently Added" section
    - Non-intrusive: won't fail if README is customized
  - Makes each shelf look professional on GitHub
  - Provides human-readable inventory alongside catalog.yml
- **Interactive Hub (Phase 1 Complete)**
  - Launch `shelfctl` with no arguments to get an interactive menu
  - Visual dashboard showing all available operations
  - Status bar displaying shelf count and total books
  - Keyboard navigation with ↑/↓ or j/k
  - Search/filter with `/`
  - Smart first-time setup guidance:
    - Visual status check (✓/✗) for token and shelves
    - Detects what's missing and shows specific next steps
    - Offers guided interactive init workflow
    - Inline architecture help with 'help' or '?' at any prompt
    - Clear distinction between repository name and shelf name
    - Example commands showing how shelf name is used
    - Smart defaults calculated from repository name
    - Enhanced summary confirmation with visual formatting
  - Clean focused menu showing only available features
  - Currently provides: Browse Library, Add Book, Quit
  - Additional commands accessible via direct invocation
  - No "coming soon" clutter - menu feels complete
- **Interactive TUI mode using Bubble Tea library**
  - `browse` command: Interactive browser with keyboard navigation, filtering, and search
  - `shelve` command: Fully guided workflow with no arguments required
    - Shelf picker (if multiple shelves)
    - File browser (starts in ~/Downloads, filters for .pdf/.epub/.mobi/.djvu)
    - Metadata form (title, author, tags, ID)
  - Auto-detects terminal vs piped/scripted output
  - New `--no-interactive` global flag to disable TUI mode
  - All TUI components use alt screen and restore terminal state on exit
- Comprehensive documentation
  - `config.example.yml`: Template configuration with detailed comments
  - `CONTRIBUTING.md`: Development setup, testing, and PR guidelines
  - `docs/COMMANDS.md`: Complete command reference with examples
  - `docs/TUTORIAL.md`: Step-by-step walkthrough from installation to workflows
  - `docs/TROUBLESHOOTING.md`: Common issues and solutions
  - `docs/ARCHITECTURE.md`: Organization guide with schemas and configuration reference
  - `docs/index.md`: Documentation home page
- Defensive measures for duplicate content detection
  - `shelve` checks for SHA256 duplicates before upload
  - `shelve` checks for asset name collisions before upload
  - `--force` flag to bypass checks and overwrite existing assets

### Changed
- **README Copy Improvements**
  - Rewrote intro section for better clarity and scannability
  - Enhanced "Why" section with clear headers and bullet points
  - Improved "How it works" section with detailed explanations
  - More conversational and direct tone throughout
  - Added visual hierarchy for easier scanning
- **Hub Menu Organization**
  - Added "Delete Book" option between "Add Book" and "Delete Shelf"
  - Logical grouping: Browse → Add → Delete Book → Delete Shelf → Quit
  - Dynamic filtering based on application state
- **Book Browser Return Type**
  - Changed from simple error return to structured `BrowserResult`
  - Returns selected book and action for processing
  - Enables interactive actions within browser
  - Cleaner separation of concerns
- **BREAKING: Command renames for end-user clarity**
  - `list` → `browse` (better matches interactive TUI functionality)
  - `add` → `shelve` (stronger library metaphor)
  - `get` removed (use `open` - it auto-downloads when needed)
  - All documentation and examples updated to reflect new names
- **README improvements**
  - Fixed factual claims about GitHub limits (removed "1TB+", "2GB storage" specifics)
  - Reordered "Why" section: on-demand first, then Git LFS comparison
  - Clarified API-based operation (no local repos required, run from anywhere)
  - Added shelby2.png above Commands section
- **Documentation structure**
  - Moved all docs to `docs/` directory
  - Created navigation structure for doc generators
  - Updated all cross-references
- **Operations made safer**
  - `shelve` loads catalog once for both duplicate checks and appending
  - Clearer error messages for duplicates
  - Force mode explicitly deletes existing assets before re-uploading
  - `open` command inlines download logic (previously called `get`)

### Changed
- **TUI Browser Details Pane**
  - Details pane now shown by default when launching browse command
  - Users see book metadata immediately without needing to press tab
  - Can still toggle to full-width list view with tab key
  - Improves discoverability and first-time user experience
- **Shelves Command Output Format**
  - Changed from plain text list to styled table format
  - Better visual organization with borders and columns
  - Easier to scan multiple shelves at once
- **Linter Configuration**
  - Updated to golangci-lint v2 configuration format
  - Disabled noisy stylistic linters (gocritic, revive)
  - Focus on substantive issues: errcheck, govet, ineffassign, staticcheck
  - All code now passes linting with zero issues

### Fixed
- **Text Overflow in Browse TUI**
  - Long book IDs, titles, and tags now truncate with ellipsis (…)
  - Prevents text from running into adjacent columns
  - ID max: 20 chars, Title max: 50 chars, Tags max: 30 chars
  - Fixes display issues like "book-id-that-is-way-too-long title-running-together ✓"
- **Error Handling**
  - Fixed all errcheck issues (14 instances of unchecked defer Close/Remove calls)
  - Properly handle errors in deferred cleanup operations across all packages
  - Improved robustness of file and HTTP response cleanup
- **Shelf README Duplicate Entries**
  - Fixed bug where books appeared multiple times in "Recently Added" section
  - Now checks for existing entries before adding to prevent duplicates
  - Implements 10-entry limit as originally intended
  - Updates book position when re-adding instead of creating duplicates
- **CI golangci-lint Version**
  - Updated GitHub Actions workflow to use golangci-lint v2
  - Fixes CI failures due to v1/v2 configuration mismatch
- **Config Reload After Hub Commands**
  - Fixed stale config bug where deleted shelves still appeared in menu
  - Config now reloads from disk after each successful command
  - Ensures menu reflects current state (added/removed shelves)
  - Prevents "not found" errors on subsequent operations
- **Linter Compliance**
  - Fixed all `errcheck` issues (ignored return values)
  - Fixed `revive` var-naming issues (renamed `select_` to `selectItem`)
  - Fixed `ineffassign` issues (proper variable initialization)
  - Fixed exported function documentation
  - Fixed error string capitalization
  - All code passes golangci-lint with project configuration
- **File Picker Permission Handling**
  - Gracefully handles permission denied errors
  - Shows parent directory (..) when access denied
  - Changed starting directory from Downloads to current working directory
  - No longer crashes on permission errors
- **Delete Shelf Repository Confusion**
  - Changed wording from "Delete repository?" to numbered choices
  - Clear explanation of what each option does
  - Prevents accidental permanent deletion
  - Shows exactly what will happen before confirmation
- **Cyclomatic Complexity**
  - Refactored 7 high-complexity functions
  - Extracted helper functions for better maintainability
  - `shelve.go`: 35 → ~10 (12 helpers)
  - `move.go`: 31 → ~8 (7 helpers)
  - `import.go`: 28 → ~10 (7 helpers)
  - `split.go`: 27 → ~12 (5 helpers)
  - `init.go`: 22 → ~8 (6 helpers)
  - `migrate.go`: 22 & 19 → ~10 & ~8 (9 helpers)

[Unreleased]: https://github.com/blackwell-systems/shelfctl/compare/v0.2.3...HEAD
[0.2.3]: https://github.com/blackwell-systems/shelfctl/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/blackwell-systems/shelfctl/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/blackwell-systems/shelfctl/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/blackwell-systems/shelfctl/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.4
[0.1.3]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.3
[0.1.2]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.2
[0.1.1]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.1
[0.1.0]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.0
