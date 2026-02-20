# shelfctl

<p align="center">
  <img src="padded.png" alt="shelfctl">
</p>

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![CI](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/shelfctl)](https://goreportcard.com/report/github.com/blackwell-systems/shelfctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Manage thousands of PDFs, EPUBs, and documents using GitHub as zero-cost infrastructure.**

shelfctl turns GitHub into a personal document library. Your account is already a free CDN and blob storage — no database, no S3, no servers to run. Organize books by topic in separate repos (`shelf-programming`, `shelf-history`), store files as Release assets, and track metadata in version-controlled `catalog.yml` files.

Search, open, and migrate documents from the CLI. Your entire library is portable git repos. Free for public collections, 2GB+ storage for private ones. Works anywhere git works.

---

## Why

**No ops burden**: No database to maintain, no blob storage to configure, no servers to patch. GitHub handles availability, backups, and CDN distribution.

**Zero cost to start**: Free for public repos. Private repos get 2GB storage + 10GB bandwidth/month on GitHub Free. Scale up only if you need it.

**Portable**: Your library is just git repos. Clone anywhere, migrate anytime. No vendor lock-in, no proprietary formats.

**Scriptable**: CLI-first design. Pipe commands, write shell scripts, integrate with your existing workflows.

---

## How it works

- Each topic gets a GitHub repo: `shelf-programming`, `shelf-history`, etc.
- Files (PDFs, EPUBs) are stored as **Release assets** — not committed to git.
- `catalog.yml` in each repo is the source of truth for metadata.
- `shelfctl` manages the whole lifecycle: add, get, open, migrate, split.

---

## Install

```bash
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest
```

Or build from source:

```bash
git clone https://github.com/blackwell-systems/shelfctl
cd shelfctl
make build
```

---

## Quick start

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_...

# Bootstrap a shelf
shelfctl init --repo shelf-programming --name programming --create-repo --create-release

# Add a book
shelfctl add ~/Downloads/sicp.pdf --shelf programming --title "SICP" --author "Abelson & Sussman" --tags lisp,cs

# List books
shelfctl list --shelf programming

# Open a book (downloads to cache on first run)
shelfctl open sicp

# Migrate from an old monorepo
shelfctl migrate scan --source you/old-books > queue.txt
shelfctl migrate batch queue.txt --n 10 --continue
```

---

## Commands

| Command | Description |
|---------|-------------|
| `init` | Bootstrap a shelf repo and release |
| `shelves` | Validate all configured shelves |
| `list` | List books with filtering |
| `info <id>` | Show metadata and cache status |
| `get <id>` | Download to local cache |
| `open <id>` | Open a book (downloads if needed) |
| `add <file\|url>` | Ingest a file or URL into a shelf |
| `move <id>` | Move between releases or shelves |
| `split` | Interactive wizard to split a shelf |
| `migrate one` | Migrate a single file from an old repo |
| `migrate batch` | Migrate a queue of files |
| `migrate scan` | List files in a source repo |
| `import` | Import from another shelfctl shelf |

---

## Configuration

Default config path: `~/.config/shelfctl/config.yml`

```yaml
github:
  owner: "you"
  token_env: "GITHUB_TOKEN"

defaults:
  release: "library"

shelves:
  - name: "programming"
    repo: "shelf-programming"
  - name: "history"
    repo: "shelf-history"
```

See [`SPEC.md`](SPEC.md) for the full configuration schema.

---

## Note

This tool manages user-provided files in user-owned GitHub repos and releases. It does not distribute content. Designed for personal document libraries (PDF/EPUB/etc.).

---

## License

MIT
