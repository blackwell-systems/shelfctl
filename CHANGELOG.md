# Changelog

All notable changes to shelfctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Panic in shelve TUI when upload fails:** `renderUploadProgress` crashed with index out of range when a single-file shelve failed (zero successes). The phase was not reset before navigating away, causing `View()` to access `selectedFiles` past its bounds.

## [0.4.9] - 2026-04-18

### Added
- **`shelfctl doctor` command:** Runs health checks on the user's setup — config file, GitHub token, API connectivity, token scopes, shelf accessibility, and cache integrity. Reports pass/fail/warn/skip with remediation hints. Exits 0 if all pass/warn, non-zero if any fail.
- **Tests for `internal/unified` package:** The `SyncAllModel`, `PerformPendingAction`, and `refreshModifiedStatusCmd` functions now have test coverage. Previously the entire `unified` package had zero test files.
- **Cache path layout regression tests:** Four new tests in `internal/cache/paths_test.go` verify the owner-scoped path layout (`<baseDir>/<owner>/<repo>/<asset>`), catching future regressions like the cross-owner collision that prompted the change.
- **`github.Client` context-aware variants:** New `GetFileContentCtx` and `DownloadAssetCtx` methods expose context propagation for callers that need request cancellation. `doJSON` now accepts a `context.Context` and uses `http.NewRequestWithContext` throughout; existing callers receive `context.Background()`.
- **`github.Client.ListDirContents`:** New method wrapping the GitHub Contents API for directory listings, replacing the raw `http.NewRequest` calls previously embedded in `internal/migrate/scan.go`.
- **`github.Client.APIBase()` and `Token()` accessors:** Expose the client's configured API base URL and token for callers that previously had to pass `token`/`apiBase` strings separately alongside a `*Client`.

### Changed
- **`migrate.ScanRepo` signature:** Changed from `ScanRepo(token, apiBase, owner, repo, ref string, exts []string)` to `ScanRepo(gh *github.Client, owner, repo, ref string, exts []string)`. Raw HTTP logic replaced with `gh.ListDirContents`. Update call sites in `internal/app/migrate.go` and `internal/unified/import_repo_ops.go`.
- **`ingest/resolver.go` GitHub path:** `resolveGitHub` now constructs a `*github.Client` via `github.New` and calls `gh.GetFileContent` instead of issuing raw `http.NewRequest` calls directly. The public `Resolve` signature is unchanged.
- **Cache directory layout is now owner-scoped:** Cached assets are stored at `<cacheDir>/<owner>/<repo>/<asset>` instead of `<cacheDir>/<repo>/<asset>`. Prevents collisions when two owners have a shelf with the same repo name. Existing caches will be treated as orphaned on next run.
- **`browserDownloader` consolidated:** The duplicate `browserDownloader` struct that existed in both `internal/app/browse.go` and `internal/unified/browse.go` is now a single implementation in `internal/unified` exported via `NewBrowserDownloader`. Removes ~160 lines of duplicate code.

### Fixed
- **Sync progress spinner now animates:** The braille spinner shown next to the current book during sync processing was static. The spinner tick chain was broken during the confirmation phase transition; restarting it on confirm resumes animation.
- **Silent download failure surfaced:** `Download` in `internal/unified/browse.go` was always returning `nil` on error instead of propagating the actual error. Download failures are now visible in the TUI.
- **Catalog parse errors surfaced in sync detect phase:** A `catalog.Parse` failure during shelf detection was silently skipped via `continue`. Now emits a `syncDetectErrorMsg` that the TUI can display.
- **Orphan detection with owner-scoped paths:** `cache.DetectOrphans` was parsing paths as `<repo>/<asset>` (2-level) after the layout changed to `<owner>/<repo>/<asset>` (3-level). Every cached file was being misclassified as an orphan. Fixed to parse the owner segment and key the known-assets map by `owner/repo`.

### Removed
- **Legacy `runHub()` deleted (~195 lines):** `internal/app/hub_runner.go` was calling the old `runHub()` implementation as a fallback for unconfigured states. These two paths (no-token, no-shelves) are now handled inline with user-friendly messages. `runHub()` and its unused imports are gone.
- **Dead `updateCatalog` function removed:** `internal/app/shelve.go` contained an unreachable `updateCatalog` function (28 lines) that was superseded by the catalog manager. Deleted.
- **Dead `BuildContext` export removed:** `internal/unified/model.go` exported `BuildContext` which had no external callers. Deleted. The `cacheMgr.Path("","","","")` workaround call it contained is replaced with `cacheMgr.BaseDir()`.
- **Dead `foundNextSection` boolean removed:** `internal/operations/readme.go`'s `AppendToShelfREADME` tracked a `foundNextSection` flag that was written but never read. Removed.
- **Duplicate `formatBytes` removed:** `internal/app/book_item.go` and `internal/unified/hub.go` both defined local `formatBytes` wrappers around `util.HumanBytes`. Removed; all call sites now use `util.HumanBytes` directly.

## [0.4.8] - 2026-04-13

### Added
- **Auto-create config on first run:** Running `shelfctl` or `shelfctl init` without a config file now prompts inline for GitHub owner and token env var name, then creates `~/.config/shelfctl/config.yml` automatically. Eliminates the chicken-and-egg problem where `init` required a config that didn't exist yet. Non-TTY environments (CI, piped stdin) skip the prompt and show the existing error message.

### Changed
- **`SHELFCTL_GITHUB_TOKEN` is now the canonical token env var.** The default `token_env` in config changed from `GITHUB_TOKEN` to `SHELFCTL_GITHUB_TOKEN`. This avoids collision with the GitHub Actions built-in token. The `GITHUB_TOKEN` fallback has been removed — update your shell profile if you were using the old name.

### Fixed
- **Actionable GitHub API errors:** HTTP 401, 403, 404, 429, and 5xx errors now return user-friendly messages with specific remediation steps instead of raw status codes. Connection timeouts show "Unable to reach GitHub API" instead of Go's net/http error.
- **Shelf validation names the failing shelf:** `shelfctl shelves` now reports which shelf failed and why (e.g. "Shelf 'programming' failed: repository 'owner/shelf-programming' not found") instead of the generic "one or more shelves have issues".
- **`errors.Is` for sentinel errors:** Replaced `err.Error() == "not found"` string comparison in `catalog/manager.go` and `err == ErrNotFound` direct comparison in `github/releases.go` and `github/repos.go` with idiomatic `errors.Is()`. Error wrapping no longer silently bypasses sentinel checks.
- **Secure git credential passing:** GitHub token is no longer embedded in the git clone URL (visible in `/proc/<pid>/cmdline`). Now uses `GIT_ASKPASS` with a temporary script to pass credentials securely.
- **TODO(bug9) migrate fix:** `processMigrationQueue` no longer spawns a cobra subcommand via `Execute()` which bypassed `PersistentPreRunE`. Now calls `migrateOneFile()` directly.
- **Deduplicated shared functions:** `humanBytes`, `openFile`, `isPDF`, and `computeFileHash` were copy-pasted between `internal/app/` and `internal/unified/`. Consolidated into `internal/util/file.go` with all call sites updated.

### Documentation
- **TUI error handling guide:** New "Error Handling" section in docs/guides/hub.md covering all failure modes (API unreachable, 401, 403, 429, download failure, sync failure).
- **catalog.yml editing guide:** New "Editing catalog.yml Directly" section in docs/reference/architecture.md for advanced users who want to edit metadata via git.

## [0.4.7] - 2026-04-13

### Added
- **Auto-sync TUI toggle:** New "Auto-sync: on/off" menu item under the Cache section in the hub. Pressing Enter toggles auto-sync and persists the change to `~/.config/shelfctl/config.yml` immediately, no restart required.
- **`shelfctl config set` CLI command:** Set config fields from the command line (e.g. `shelfctl config set sync.auto_sync true`, `shelfctl config set sync.debounce_minutes 10`).

### Fixed
- **Auto-sync catalog cache coherence:** After a successful auto-sync run, the hub now reloads catalog data from GitHub so the in-memory SHA cache reflects the new checksums. Previously, stale cached SHAs caused `HasBeenModified` to re-flag already-synced books, triggering a second sync that failed with "nothing to commit, working tree clean".
- **Auto-sync headless startup:** `SyncAllModel.Init()` in auto mode now skips the `detectAsync()` phase (which emits a `syncDetectedMsg` that is never forwarded in headless mode) and starts uploading immediately. Previously, auto-sync silently stalled after showing "Auto-syncing…".
- **Auto-sync tight retry loop:** Failed books were immediately re-queued after every sync attempt. Now, catalog reload only runs when at least one book was successfully synced; failed books are retried on the next natural 20-second hub scan tick.
- **Auto-sync error visibility:** The `autoSyncDoneMsg` now carries the actual error strings from each failed book. The hub status line shows the first error message (e.g. "↑ Auto-sync failed: <book>: <reason>") instead of a generic "N error(s)" count.
- **Auto-sync timestamp:** The hub status line now includes the time of the last run (e.g. `↑ Auto-synced 1 book(s) at 3:42 PM`), making it easy to tell how recently auto-sync ran.
- **Config round-trip:** Multi-word config fields (`auto_sync`, `debounce_minutes`, `cache_dir`, `token_env`, `api_base`, `asset_naming`, `catalog_path`, `default_release`) were being written without underscores when `config.Save()` was called. Added `yaml` struct tags matching the `mapstructure` tags to all config structs.

### Documentation & Discoverability
- **README install section:** Added winget (`winget install BlackwellSystems.shelfctl`) and Scoop install instructions, which were missing from the Install section entirely.
- **README structure:** Consolidated duplicate "30-Second Quickstart" / "Quick start" sections into a single "Usage" section — removes reader confusion.
- **pkg.go.dev:** Added package doc comment to `cmd/shelfctl/main.go` so pkg.go.dev displays a meaningful description for the module instead of a blank page.
- **GitHub topics:** Replaced `content-management-system` and `ebook-manager` (redundant) with `books`, `reading-list`, `personal-library`, and `document-management` for broader search discoverability.

### Scoop distribution
- **Scoop distribution:** shelfctl is now available via Scoop (`scoop bucket add blackwell-systems https://github.com/blackwell-systems/scoop-bucket && scoop install shelfctl`). Bucket is auto-updated on each release via GoReleaser.
- **Automated winget PR submission:** Future releases automatically submit a PR to `microsoft/winget-pkgs` via `winget-releaser` GitHub Action — no manual manifest updates needed.

### Infrastructure
- GoReleaser now auto-publishes Homebrew formula and Scoop manifest on every tagged release.
- winget manifest validation runs in a separate CI workflow triggered on changes to `winget/**`.

## [0.4.5] - 2026-04-11

### Added
- **Opt-in auto-sync:** When `sync.auto_sync: true` is set in config,
  the hub automatically uploads locally modified books to GitHub after
  they pass a debounce window (default 5 minutes). No confirmation
  prompt is shown. A status line "↑ Auto-synced N book(s)" appears in
  the hub after a successful run.
- **`sync.auto_sync` config field** (bool, default `false`): enables
  background auto-sync from the hub view.
- **`sync.debounce_minutes` config field** (int, default `5`): files
  modified within this many minutes are skipped to avoid uploading files
  still being written to.
- **Install script:** `curl -fsSL https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/install.sh | sh` — installs the latest release to `/usr/local/bin` with SHA256 verification. Supports `INSTALL_DIR` and `VERSION` overrides.
- **winget support:** shelfctl is available via Windows Package Manager (`winget install BlackwellSystems.shelfctl`), pending initial PR acceptance.

## [0.4.4] - 2026-04-09

### Improved
- **Sync Modified hub awareness:** The "Sync Modified" menu item now appears
  automatically within ~20 seconds of a cached book being modified locally,
  without requiring the user to navigate away and back. A lightweight background
  scan runs every 20 seconds while on the hub view, reusing already-cached
  catalog data (no GitHub API calls) and comparing local file SHA256 checksums.
  The scan auto-pauses when navigating to other views and resumes on return
  (`unified/model.go`: `refreshModifiedStatusCmd`, `hubModifiedRefreshTick`).

### Fixed
- **Sync Modified upload progress bar:** The processing phase now shows a gradient
  progress bar and `X.X / Y.Y MB (N%)` counter during each book upload. Previously
  only a static spinner was shown. Implemented by splitting `syncBookCmd` into
  `syncBookSetupCmd` (EnsureRelease / FindAsset / DeleteAsset / start goroutine) +
  `waitForSyncTick` (recursive channel reader) + `syncCatalogCmd` (catalog commit
  after upload). `progress.FrameMsg` forwarded to `progress.Model` for smooth
  gradient animation between ticks (`unified/sync_all.go`).
- **Sync Modified spinner:** Detecting and processing phases now animate a spinner
  instead of rendering a static `⏳` emoji.

## [0.4.3] - 2026-04-08

### Added
- **Sync Modified Books (hub menu):** New hub menu item "Sync Modified" under Cache —
  appears only when one or more locally modified cached books are detected. Navigates to
  a dedicated TUI view (`SyncAllModel`) with phases: detecting (async catalog scan) →
  confirming (list of modified books, enter to proceed) → processing (sequential per-book
  upload with live ✓/✗ status) → done (summary). Replaces the old hint directing users to
  the CLI (`unified/sync_all.go`, `tui/hub.go`, `unified/model.go`).
  The hub cache details pane hint now reads "Select Sync Modified from the menu" instead
  of "Press 's' in browse or run 'sync --all'".

## [0.4.2] - 2026-04-08

### Added
- **Rename Shelf:** New hub menu item under Shelves — renames both the display
  name and the GitHub repository in one operation. Renames the GitHub repo via
  API, updates both `name` and `repo` fields in config, and moves the local
  cache directory from the old repo path to the new one. Pre-fills inputs with
  the current name and repo suffix for easy editing.
- **Version in hub TUI:** The hub wordmark now shows the current version in a
  dim, unbolded style (`shelfctl v0.4.x`). Extracted `Wordmark(version string)`
  into `tui/common.go` for reuse. Version flows from ldflags via `HubContext`
  and is preserved through the async context load.

### Fixed
- **Hub menu missing browse after async load:** `UpdateContext` was updating the
  hub's context data but not rebuilding the menu items, so browse stayed hidden
  even after `BookCount` populated. Now calls `list.SetItems()` on update.
- **New shelf not visible in browse after creation:** `m.cfg` in the unified
  model was never reloaded after config-mutating operations (create shelf, delete
  shelf). The config file was updated on disk but the in-memory copy was stale,
  so browse only showed shelves that existed at startup. Now reloads config and
  clears the catalog cache on every hub navigation.

### Performance
- **Eliminated redundant catalog fetches:** The startup context builder now
  fetches each shelf's catalog once and reuses the data for both shelf details
  and cache stats (was 2 fetches per shelf). Health checks (`RepoExists`,
  `GetReleaseByTag`) removed from startup — deferred to `ShelvesModel` which
  already loads them async on demand.
- **In-session catalog cache:** `Model` now caches parsed catalogs in memory.
  Navigating to browse or index after the initial async load makes zero
  additional GitHub API calls. Cache is automatically reset on TUI restart
  (after any write operation).
- **Browse visible immediately:** Removed browse from the `BookCount == 0`
  filter — it is already gated on `ShelfCount`, so it now appears as soon as
  the TUI starts rather than waiting for the async load.

### Build
- `make build` and `make install` now inject the current git tag as the version
  string via `-ldflags "-X main.version=$(VERSION)"`. Eliminates the `dev`
  version shown in local builds.

## [0.4.1] - 2026-04-08

### Fixed
- **TUI startup hang:** `shelfctl` with no arguments would silently block on
  startup while making synchronous GitHub API calls (repo check + catalog fetch
  + release check per shelf) before the TUI started. With a 5-minute HTTP
  timeout this was indistinguishable from a frozen binary. Fixed by introducing
  `BuildContextFast` (local config only, no network) to start the TUI
  immediately, then loading the full context asynchronously via a Tea command
  (`unified/model.go`). The same fix applies when returning to the hub after
  an action (`handleNavigation` "hub" case).

## [0.4.0] - 2026-03-23

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

[Unreleased]: https://github.com/blackwell-systems/shelfctl/compare/v0.4.4...HEAD
[0.4.4]: https://github.com/blackwell-systems/shelfctl/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/blackwell-systems/shelfctl/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/blackwell-systems/shelfctl/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/blackwell-systems/shelfctl/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.3...v0.4.0
[0.3.3]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/blackwell-systems/shelfctl/compare/v0.3.1...v0.3.2
