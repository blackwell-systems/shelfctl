# shelfctl

<p align="center">
  <img src="padded.png" alt="shelfctl">
</p>

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![CI](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/shelfctl)](https://goreportcard.com/report/github.com/blackwell-systems/shelfctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow.svg?logo=buy-me-a-coffee)](https://buymeacoffee.com/blackwellsystems)

**Organize the PDFs and books you already have scattered across GitHub repos.**

Most developers have a mess: one monolithic `books` repo with 500+ PDFs, or files scattered across random repos and gists. shelfctl splits the monolith into organized topic-based shelves and migrates scattered files into a searchable library. No migration to another service — it works with what you already have.

Your GitHub account is already free CDN and blob storage. shelfctl uses Release assets for files and `catalog.yml` for metadata. Organize by topic (`shelf-programming`, `shelf-history`), search from the CLI, open books instantly. Your entire library stays as portable git repos.

Zero infrastructure. Zero cost for public repos, 2GB+ storage for private. Works anywhere git works.

---

## Why

**No ops burden**: No database to maintain, no blob storage to configure, no servers to patch. GitHub handles availability, backups, and CDN distribution.

**On-demand downloads**: Download individual books without cloning repos or pulling entire releases. `shelfctl open book-id` fetches just that one file from GitHub's CDN and opens it. Your library can be 100GB+ but you only download what you need.

**Zero cost to start**: Free for public repos. Private repos get 2GB storage + 10GB bandwidth/month on GitHub Free. Scale up only if you need it.

**Portable**: Your library is just git repos. Clone anywhere, migrate anytime. No vendor lock-in, no proprietary formats.

**Scriptable**: CLI-first design. Pipe commands, write shell scripts, integrate with your existing workflows.

---

## How it works

- Each topic gets a GitHub repo: `shelf-programming`, `shelf-history`, etc.
- Files (PDFs, EPUBs) are stored as **Release assets** — not committed to git.
- `catalog.yml` in each repo is the source of truth for metadata.
- **Download individual books on demand** — no need to clone repos or download entire releases.
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

**Already have PDFs in GitHub repos?** Organize them:

```bash
export GITHUB_TOKEN=ghp_...

# Scan your existing repos for files
shelfctl migrate scan --source you/old-books-repo > queue.txt

# Create organized shelves
shelfctl init --repo shelf-programming --name programming --create-repo --create-release
shelfctl init --repo shelf-research --name research --create-repo --create-release

# Edit queue.txt to map files to shelves, then migrate
shelfctl migrate batch queue.txt --n 10 --continue
```

**Starting fresh?** Add books directly:

```bash
# Add a book
shelfctl add ~/Downloads/sicp.pdf --shelf programming --title "SICP" --author "Abelson & Sussman" --tags lisp,cs

# List books across all shelves
shelfctl list --shelf programming

# Open a book — downloads just this one file (6MB), not the entire release
shelfctl open sicp

# On another machine? Same command fetches it on-demand from GitHub
shelfctl open sicp
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
