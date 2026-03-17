# Changelog

All notable changes to shelfctl will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- `generateHTML`, `renderBookCard`, and `renderUncachedCard` in the HTML index
  generator are now methods on `*cache.Manager`, eliminating the need to thread
  `baseDir` as a parameter through the call chain (`cache/html_index.go`).

### Fixed
- Cover images in the browse TUI details pane now render correctly on Kitty and
  Ghostty terminals. Two bugs prevented them from ever displaying: the Kitty
  graphics protocol `t=f` parameter (file path) was used instead of `t=d`
  (inline base64), and the APC escape sequences were passed through lipgloss
  layout operations that corrupted them. The cover image is now rendered outside
  the lipgloss pipeline (`tui/image.go`, `tui/browser_render.go`).
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
