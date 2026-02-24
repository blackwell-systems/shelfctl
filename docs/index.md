# shelfctl Documentation

Zero-infrastructure document library using GitHub repos and releases as storage.

## Getting Started

- **[Tutorial](guides/tutorial.md)** - Step-by-step guide from installation to common workflows
- **[Architecture Guide](reference/architecture.md)** - How shelves work, organization strategies, and scaling
- **[Interactive Hub](guides/hub.md)** - Guide to the interactive TUI menu
- **[Commands Reference](reference/commands.md)** - Complete command documentation with examples
- **[Troubleshooting](reference/troubleshooting.md)** - Common issues and solutions

## Reference

- **[Contributing Guide](development/contributing.md)** - Development setup and contribution guidelines
- **[Changelog](development/changelog.md)** - Release history and version notes

## Quick Links

### First Time Setup
1. [Authenticate with GitHub](guides/tutorial.md#step-1-authenticate-with-github)
2. [Install shelfctl](guides/tutorial.md#step-2-install-shelfctl)
3. [Configure shelfctl](guides/tutorial.md#step-3-set-up-configuration)
4. [Create your first shelf](guides/tutorial.md#step-4-create-your-first-shelf)

### Common Tasks
- [Add a book](reference/commands.md#shelve) - `shelfctl shelve ~/book.pdf --shelf programming`
- [Browse your library](reference/commands.md#browse) - `shelfctl browse --tag algorithms`
- [Generate HTML index](reference/commands.md#index) - `shelfctl index` for web browsing
- [Manage cache](reference/commands.md#cache) - `shelfctl cache info` and `shelfctl cache clear`
- [Edit a book](reference/commands.md#edit-book) - `shelfctl edit-book book-id`
- [Open a book](reference/commands.md#open) - `shelfctl open book-id`
- [Migrate existing files](reference/commands.md#migrate-batch) - Organize your monolithic repo

### Migration Workflows
- [From monolithic repo](guides/tutorial.md#workflow-4-migrating-from-an-existing-collection)
- [Split large shelf](reference/commands.md#split)
- [Import from another shelf](reference/commands.md#import)

## Architecture

shelfctl uses GitHub Releases as a storage backend:
- **Metadata**: Version-controlled `catalog.yml` files in git repos
- **Files**: PDF/EPUB/etc. stored as Release assets (GitHub's CDN)
- **Downloads**: Individual files on-demand (no need to clone or download entire releases)

See [architecture.md](reference/architecture.md) for detailed architecture documentation and schemas.

## Support

- **Issues**: [GitHub Issues](https://github.com/blackwell-systems/shelfctl/issues)
- **Discussions**: [GitHub Discussions](https://github.com/blackwell-systems/shelfctl/discussions)
- **Troubleshooting**: [Common problems and solutions](reference/troubleshooting.md)
