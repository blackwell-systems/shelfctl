# shelfctl

Personal library manager for PDFs and books using GitHub Release assets as storage.

## Install

```bash
pip install shelfctl
```

This installs the `shelfctl` binary for your platform. No Python runtime is needed at execution time.

## Usage

```bash
# Launch interactive TUI
shelfctl

# Add a book to a shelf
shelfctl shelve ~/book.pdf --shelf programming

# Browse your library
shelfctl browse

# Generate HTML index
shelfctl index

# Check your setup
shelfctl doctor

# Search across shelves
shelfctl search "neural networks"
```

## What is shelfctl?

shelfctl organizes documents (PDFs, EPUBs) into topic-based "shelves" backed by GitHub repositories:

- **Metadata** lives in version-controlled `catalog.yml` files
- **Files** are stored as GitHub Release assets (free, unlimited storage for public repos)
- **Downloads** are on-demand per file (no need to clone entire repos)

## Other Installation Methods

| Method | Command |
|--------|---------|
| Homebrew | `brew install blackwell-systems/tap/shelfctl` |
| Scoop (Windows) | `scoop install shelfctl` |
| winget (Windows) | `winget install BlackwellSystems.shelfctl` |
| Go install | `go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest` |
| Install script | `curl -fsSL https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/install.sh \| sh` |
| APT/RPM | Download `.deb` or `.rpm` from [Releases](https://github.com/blackwell-systems/shelfctl/releases) |

## Supported Platforms

| Platform | Architecture |
|----------|-------------|
| macOS | arm64 (Apple Silicon), x86_64 (Intel) |
| Linux | arm64, x86_64 |
| Windows | arm64, x86_64 |

## Links

- [Documentation](https://github.com/blackwell-systems/shelfctl/tree/main/docs)
- [Source Code](https://github.com/blackwell-systems/shelfctl)
- [Changelog](https://github.com/blackwell-systems/shelfctl/blob/main/CHANGELOG.md)
- [Issues](https://github.com/blackwell-systems/shelfctl/issues)

## License

MIT
