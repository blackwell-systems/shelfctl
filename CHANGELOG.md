# Changelog

All notable changes to shelfctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-select move action in browse TUI: press `m` on selected books to move them
  to a different shelf. The TUI shows a shelf picker, then executes the catalog
  operations (remove from source, append to dest, commit both) automatically
  (`tui/list_browser.go`, `tui/keys.go`, `app/browse.go`).
- `Year` field added to shelve form. The form now has Title, Author, Year, Tags,
  ID, and Cache checkbox. Year is pre-populated from the `--year` flag if provided
  (`tui/shelve_form.go`).
- **Cache orphan detection:** `shelfctl cache clear --orphans` finds and removes
  cached files for books no longer in any shelf catalog. Cross-references cache
  against all shelves, shows orphaned files by repo/filename with sizes, prompts
  for confirmation before deletion. Addresses cache growth from removed books
  (`cache/orphan.go`, `app/cache.go`).

### Changed
- `catalog.Manager.gh` field narrowed to a new `GitHubClient` interface, enabling
  mock injection in tests without affecting production call sites (`catalog/manager.go`).
- `app/` commands now use a `GitHubClient` interface (defined in `app/interfaces.go`)
  instead of the concrete `*github.Client` type. This enables testability without
  changing production code - `*github.Client` satisfies the interface automatically.
- `generateHTML`, `renderBookCard`, and `renderUncachedCard` in the HTML index
  generator are now methods on `*cache.Manager`, eliminating the need to thread
  `baseDir` as a parameter through the call chain (`cache/html_index.go`).

### Improved

**New User Discoverability** - Extensive help text and documentation improvements
addressing 16 friction items from `docs/NEW-USER-FRICTION.md`. This wave focused
on documentation and help text only; functional improvements (interactive config
wizard, empty TUI state help, specific validation errors) are tracked separately.

- **Command help improvements:**
  - Root command (`shelfctl --help`) now includes a "Getting Started" section with
    the 4 critical first commands (set token, init, shelve, browse).
  - `init` command documents key concepts (shelf, release, asset, catalog.yml) and
    clarifies which flags are required vs optional.
  - `shelve` command explains interactive vs non-interactive workflows and documents
    the `--shelf` flag requirement.
  - `browse` command documents all TUI key bindings (↑/↓, enter, space, /, m, d, i, e, q, ?)
    and explains empty shelf behavior.
  - `open` command explains cache behavior (location, auto-download, sync workflow).

- **Error message improvements:**
  - Config loading errors now include the file path and hint to check permissions or
    recreate with `shelfctl init`.
  - Missing config error now suggests running `shelfctl init --help` to get started.

- **README restructuring:**
  - Added 30-second quickstart at the top (token setup → init → shelve → browse).
  - Added "Core Concepts" section early in README explaining shelf→repo→release→catalog.yml→assets.
  - Reordered Quick Start to show "Starting fresh" basics before migration examples.
  - Added TUI Quick Reference callout with navigation keys.
  - Clarified that fine-grained GitHub tokens need BOTH Contents and Releases permissions.

### Removed
- In-TUI cover image rendering removed entirely. Bubble Tea's frame renderer
  erases screen lines before writing them, overwriting Kitty/iTerm2 image cells
  on every redraw. Reliable inline image rendering inside a Bubble Tea alt-screen
  layout requires protocol features (Kitty unicode placeholders, explicit cell
  placement) that are out of scope. Cover images still work in the HTML index.
  `tui/image.go`, `HasCover`/`CoverPath` fields on `tui.BookItem`, and the
  associated cover-loading code in `app/browse.go`, `app/cache.go`, and
  `unified/model.go` have been removed.

### Fixed
- HTML index cover images now display correctly for all books. The wave agent's
  BUG 25 fix incorrectly used `filepath.Dir(book.FilePath)` as the anchor for
  relative cover paths; `FilePath` is the cached PDF path (empty for uncached
  books), not the index directory. Covers are now resolved relative to `baseDir`
  where `index.html` lives (`cache/html_index.go`).
- HTML index view is now rendered natively in the TUI (scrollable book list with
  search, cached/uncached indicators, `g` to generate HTML, `o` to generate and
  open in browser) instead of shelling out and dropping back to the terminal.
  `shelfctl index [--open]` CLI command is unchanged (`unified/index.go`,
  `unified/model.go`).
- **Empty shelf guidance:** Browsing a shelf with zero books now displays helpful
  text explaining how to add books (`shelfctl shelve <file.pdf>`) and how to exit.
  Disables filter/pagination UI when empty to reduce clutter. Addresses CRITICAL
  friction item from NEW-USER-FRICTION.md (`tui/list_browser.go`).

### Tests
- `internal/github/` package now has comprehensive test coverage (was previously
  untested). Added `client_test.go` (HTTP client, auth headers, status code handling),
  `contents_test.go` (GetFileContent, large-file blob fallback), and `gitops_test.go`
  (CommitFile error paths, git command helpers) using httptest fake servers.
- `app/` commands now testable via `GitHubClient` interface injection. Added
  `open_test.go` (isPDF), `browse_test.go` (checkDuplicates, handleAssetCollision),
  and `verify_e2e_test.go` (verifySingleShelf end-to-end with fake GitHub client).
- `tui/shelve_form.go` now has first test coverage via `shelve_form_test.go`
  (defaults, Year field, tab navigation, submit/cancel).
- `catalog.Manager` now has full test coverage via mock `GitHubClient` injection:
  `Load`, `Save`, `Update`, `FindByID`, `Remove`, and `Append` (happy path, load
  error, save error, fn error, not-found cases) (`catalog/manager_test.go`).
- Added `app/info_test.go`: table-driven coverage of `humanBytes` across all
  unit boundaries (B, KiB, MiB, GiB).
- Added `app/shelves_test.go`: coverage of `stripAnsi`, `padRight`,
  `padRightColored`, `formatBookCount`, and `formatStatus`.

**Comprehensive Testing Infrastructure** - Added 100+ tests across unit,
integration, and end-to-end layers with mock GitHub API server infrastructure
for fast, isolated testing.

- **Mock server foundation:** Test harness with mock GitHub API server using
  deterministic int64 Asset IDs and fixture loading (`test/mockserver/`,
  `test/fixtures/`).
- **Integration tests:** Expanded `test/scenarios/` with edge case coverage
  for cache and browse operations, plus mock server test suite (5 tests).
- **Unit test coverage:**
  - App operations: `delete_test.go` (6 tests), `move_test.go` (5 tests),
    `migrate_test.go` (16 tests), `cache_test.go` (5 tests), `init_test.go`
    (skipped - needs refactor to mockserver pattern).
  - TUI components: `list_browser_test.go` (8 tests + 20 subtests, 41.67%
    coverage), `book_picker_test.go` (8 tests), `edit_form_test.go` (8 tests).
  - Config: `load_test.go` (6 tests, 76.3% coverage).
  - GitHub: `releases_test.go` (5 tests, 100% coverage).
- **End-to-end test suite:**
  - Workflow tests: `test/e2e/workflows_test.go` (4 tests) covering new user
    flow (init→shelve→browse→open), migration workflow, cache management, and
    multi-shelf operations.
  - Edge cases: `test/e2e/edge_cases_test.go` (6 tests) covering empty shelf
    behavior, duplicate handling, network failures, corrupted cache recovery,
    malformed catalog YAML, and concurrent access.
- **CI/CD automation:** `.github/workflows/test-harness.yml` GitHub Actions
  workflow with unit, integration, and E2E test runs plus coverage reporting.

**Coverage achievements:** Config package 76.3%, GitHub releases 100%, TUI
list browser 41.67%. All tests use consistent TestHarnessSetup pattern with
mock server infrastructure, enabling fast isolated testing without external
GitHub API dependencies.

## [0.3.3] - 2026-03-17

### Fixed
- Removed test functions that wrote to the real user config file at
  `~/.config/shelfctl/config.yml`, which could overwrite existing shelf
  configuration when running the test suite (`operations/shelf_test.go`).

## [0.3.2] - 2026-03-17

### Fixed

**Critical**
- `cache clear` would delete the entire shelfctl data directory instead of just
  the cache directory due to an erroneous `filepath.Dir` call on the cache base
  path (`app/cache.go`). The cache info display showed the same wrong path.

**High**
- `delete-book` now commits the catalog removal *before* deleting the GitHub
  release asset. Previously, a network failure between those two steps left the
  asset gone but the catalog entry intact — an unrecoverable orphan
  (`app/delete_book.go`).
- Cross-shelf `move` now writes the destination catalog *before* removing the
  book from the source catalog. Previously a mid-operation failure could lose
  the book from both catalogs simultaneously (`app/move.go`).

**Medium**
- `verify --fix` no longer silently skips consecutive orphaned catalog entries
  due to iterator invalidation during in-place slice removal (`app/verify.go`).
- Cancelled downloads and uploads now drain their goroutines before returning,
  preventing background goroutine leaks across `open`, `shelve`, `browse`, and
  `sync` commands (`app/open.go`, `app/shelve.go`, `app/browse.go`,
  `app/sync.go`).
- `browse` download errors are now propagated to callers instead of being
  silently swallowed (`app/browse.go`).
- `git pull --rebase` in `CommitFile` now runs before writing the file rather
  than after committing, so a rebase conflict no longer leaves a half-committed
  catalog state (`github/gitops.go`).
- Migration file hashing now returns an error on non-EOF read failures instead
  of silently producing a partial SHA-256 and uploading a corrupt asset with
  the wrong `Content-Length` (`app/migrate.go`).
- Failed migrations no longer increment the processed counter toward the `--n`
  limit (`app/migrate.go`).
- `addShelfToConfig` now returns an error when `config.Load` fails instead of
  silently replacing the entire config with an empty one, which would drop all
  existing shelves (`operations/shelf.go`).
- `AppendToShelfREADME` no longer duplicates lines after the Quick Stats section
  when a Recently Added section does not yet exist (`operations/readme.go`).
- Migration `ScanRepo` HTTP client now has a 30-second timeout; previously it
  could hang indefinitely on a slow network (`migrate/scan.go`).
- Fixed nil pointer panic when `http.NewRequest` returns an error in migration
  scanning and GitHub resolver (`migrate/scan.go`, `ingest/resolver.go`).
- Removed eager HEAD request in HTTP resolver that blocked for up to 15 seconds
  before the file was actually needed (`ingest/resolver.go`).

**Low**
- `catalog.Manager.Remove` no longer creates a no-op GitHub commit when the
  requested book ID does not exist (`catalog/manager.go`).
- `delete-shelf` config update now filters the freshly-loaded config instead of
  the stale in-memory copy, preventing concurrent shelf additions from being
  silently lost (`app/delete.go`).
- PDF filename check in `open` is now case-insensitive (`.PDF`, `.Pdf`, etc.
  now correctly trigger the poppler hint) (`app/open.go`).
- Tag ordering after `edit-book` is now deterministic (sorted); previously map
  iteration produced random tag order, causing spurious catalog diffs on every
  edit (`app/edit_book.go`).
- Confirmation prompts for `delete-shelf` now correctly handle shelf names
  containing spaces (`app/delete.go`, `app/helpers.go`).
- PDF metadata scanner buffer increased from 8 KB to 64 KB and scanner errors
  are now checked after the scan loop (`ingest/pdfmeta.go`).
- Fixed incorrect relative path computation for cover images in the HTML index
  generator (`cache/html_index.go`).

[Unreleased]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.3...HEAD
[0.3.3]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.1...v0.3.2
