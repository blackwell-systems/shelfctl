# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
    - Prompts user through shelf creation with defaults
  - Currently wired: Browse Library, Add Book
  - Coming soon indicators for unimplemented features
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
  - `docs/SPEC.md`: Architecture and configuration schema
  - `docs/index.md`: Documentation home page
- Defensive measures for duplicate content detection
  - `shelve` checks for SHA256 duplicates before upload
  - `shelve` checks for asset name collisions before upload
  - `--force` flag to bypass checks and overwrite existing assets

### Changed
- **BREAKING: Command renames for end-user clarity**
  - `list` → `browse` (better matches interactive TUI functionality)
  - `add` → `shelve` (stronger library metaphor)
  - `get` removed (use `open` - it auto-downloads when needed)
  - All documentation and examples updated to reflect new names
- **README improvements**
  - Fixed factual claims about GitHub limits (removed "1TB+", "2GB storage" specifics)
  - Reordered "Why" section: on-demand first, then Git LFS comparison
  - Clarified API-based operation (no local repos required, run from anywhere)
  - Added logo2.png above Commands section
- **Documentation structure**
  - Moved all docs to `docs/` directory
  - Created navigation structure for doc generators
  - Updated all cross-references
- **Operations made safer**
  - `shelve` loads catalog once for both duplicate checks and appending
  - Clearer error messages for duplicates
  - Force mode explicitly deletes existing assets before re-uploading
  - `open` command inlines download logic (previously called `get`)

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
