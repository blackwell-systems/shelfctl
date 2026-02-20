# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Interactive TUI mode using Bubble Tea library
  - `browse` command shows interactive browser with keyboard navigation
  - `shelve` command with complete guided workflow: shelf picker → file picker → metadata form
  - Run `shelfctl shelve` with no arguments for fully guided book addition
  - Filtering, search, and visual selection throughout
  - Auto-detects terminal mode vs piped/scripted output
  - New `--no-interactive` global flag to disable TUI
- Better command names for clarity
  - `browse` (was `list`) - Browse library with TUI or text
  - `shelve` (was `add`) - Add a book to your library
  - `open` now auto-downloads (removed redundant `get` command)

### Changed
- **BREAKING:** Command renames for end-user clarity
  - `list` → `browse` (better matches interactive TUI functionality)
  - `add` → `shelve` (stronger library metaphor)
  - `get` command removed (use `open` instead - it auto-downloads)
- README improvements
  - Fixed factual claims about GitHub storage limits (no more "1TB+" or "2GB storage" specifics)
  - Reordered "Why" section for better pitch flow (on-demand first, then Git LFS comparison)
  - Clarified API-based operation (no local repos required)
  - Added logo2.png above Commands section
- Reorganized documentation into `docs/` directory for better structure
  - Moved TUTORIAL.md, COMMANDS.md, SPEC.md, TROUBLESHOOTING.md to docs/
  - Created docs/index.md as documentation home page
  - Added docs/nav.yml for documentation generator compatibility
  - Updated README links to point to new locations

- Defensive measures for duplicate content detection
  - `shelve` command checks for existing files with same SHA256 checksum
  - `shelve` command checks for asset name collisions before upload
  - New `--force` flag to bypass duplicate checks and overwrite existing assets
- Comprehensive documentation
  - `config.example.yml`: Template configuration file with detailed comments
  - `CONTRIBUTING.md`: Development setup, testing, and PR guidelines
  - `COMMANDS.md`: Complete command reference with examples for all commands
  - `TUTORIAL.md`: Step-by-step guide from installation to common workflows
  - `TROUBLESHOOTING.md`: Common issues and solutions
- README improvements
  - Centered logo with padding
  - Zero-infrastructure value proposition front and center
  - New "Why" section highlighting benefits (no ops, zero cost, portable, scriptable)
  - Fixed placeholder URLs (YOUR_GH_OWNER → blackwell-systems)
  - Removed reference to non-existent config.example.yml

- Operations made safer and more idempotent
  - `shelve` command loads catalog once for both duplicate checks and appending
  - Clearer error messages when duplicates are detected
  - Force mode explicitly deletes existing asset before re-uploading

## [0.1.0] - 2024-02-20

### Added
- Initial implementation of shelfctl
- GitHub releases as storage backend for document libraries
- Core commands: init, add, list, info, get, open, move, split, migrate, import
- Catalog-based metadata management with YAML
- Local cache for downloaded files with SHA256 verification
- Support for multiple shelf repos per user
- Flexible asset naming strategies (id-based or original filename)
- Migration tools for importing from existing repos
- Import command with SHA256-based duplicate detection
- Test suite with coverage: cache 72.7%, catalog 74.2%, util 71.9%, ingest 40.3%, migrate 49.4%, app 12.4%

[Unreleased]: https://github.com/blackwell-systems/shelfctl/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/blackwell-systems/shelfctl/releases/tag/v0.1.0
