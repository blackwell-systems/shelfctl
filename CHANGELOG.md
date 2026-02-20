# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Defensive measures for duplicate content detection
  - `add` command now checks for existing files with same SHA256 checksum
  - `add` command now checks for asset name collisions before upload
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

### Changed
- Operations made safer and more idempotent
  - `add` command loads catalog once for both duplicate checks and appending
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
