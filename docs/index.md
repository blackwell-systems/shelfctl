# shelfctl Documentation

Zero-infrastructure document library using GitHub repos and releases as storage.

## Getting Started

- **[Tutorial](TUTORIAL.md)** - Step-by-step guide from installation to common workflows
- **[Architecture Guide](ARCHITECTURE.md)** - How shelves work, organization strategies, and scaling
- **[Interactive Hub](HUB.md)** - Guide to the interactive TUI menu
- **[Commands Reference](COMMANDS.md)** - Complete command documentation with examples
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions

## Reference

- **[Technical Specification](SPEC.md)** - Architecture, design decisions, and configuration schema
- **[Contributing Guide](../CONTRIBUTING.md)** - Development setup and contribution guidelines
- **[Changelog](../CHANGELOG.md)** - Release history and version notes

## Quick Links

### First Time Setup
1. [Install shelfctl](TUTORIAL.md#step-2-install-shelfctl)
2. [Configure GitHub token](TUTORIAL.md#step-1-create-a-github-token)
3. [Create your first shelf](TUTORIAL.md#step-5-create-your-first-shelf)

### Common Tasks
- [Add a book](COMMANDS.md#shelve) - `shelfctl shelve ~/book.pdf --shelf programming`
- [Browse your library](COMMANDS.md#browse) - `shelfctl browse --tag algorithms`
- [Edit a book](COMMANDS.md#edit-book) - `shelfctl edit-book book-id`
- [Open a book](COMMANDS.md#open) - `shelfctl open book-id`
- [Migrate existing files](COMMANDS.md#migrate-batch) - Organize your monolithic repo

### Migration Workflows
- [From monolithic repo](TUTORIAL.md#workflow-4-migrating-from-an-existing-collection)
- [Split large shelf](COMMANDS.md#split)
- [Import from another shelf](COMMANDS.md#import)

## Architecture

shelfctl uses GitHub Releases as a storage backend:
- **Metadata**: Version-controlled `catalog.yml` files in git repos
- **Files**: PDF/EPUB/etc. stored as Release assets (GitHub's CDN)
- **Downloads**: Individual files on-demand (no need to clone or download entire releases)

See [SPEC.md](SPEC.md) for detailed architecture documentation.

## Support

- **Issues**: [GitHub Issues](https://github.com/blackwell-systems/shelfctl/issues)
- **Discussions**: [GitHub Discussions](https://github.com/blackwell-systems/shelfctl/discussions)
- **Troubleshooting**: [Common problems and solutions](TROUBLESHOOTING.md)
