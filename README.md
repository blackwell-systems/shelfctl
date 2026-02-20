# shelfctl

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![CI](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/shelfctl/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/blackwell-systems/shelfctl)](https://goreportcard.com/report/github.com/blackwell-systems/shelfctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow.svg?logo=buy-me-a-coffee)](https://buymeacoffee.com/blackwellsystems)

<p align="center">
  <img src="assets/padded.png" alt="shelfctl">
</p>

<p align="center">
  <img src="assets/shelf.png" alt="shelfctl architecture" width="800">
</p>

**Organize the PDFs and books you already have scattered across GitHub repos.**

Most developers have a mess: one monolithic `books` repo with 500+ PDFs, or files scattered across random repos and gists. Maybe you've hit Git's file size limits or that "this exceeds GitHub's file size limit of 100.00 MB" error. Maybe you're using Git LFS but it's expensive and clunky.

shelfctl solves this by using GitHub Releases (release assets avoid Git's 100MB file limit) instead of git commits. Split your monolith into organized topic-based shelves, migrate scattered files, search and download on-demand. No migration to another service — it works with what you already have.

Your GitHub account is already free CDN and blob storage. shelfctl uses Release assets for files and `catalog.yml` for metadata. Organize by topic (`shelf-programming`, `shelf-history`), search from the CLI, open books instantly. Your entire library stays as portable git repos.

Zero infrastructure. Start free; only pay if you opt into LFS or exceed GitHub's plan limits. Works anywhere git works.

---

## Why

**On-demand downloads**: Download individual books without cloning repos or pulling entire releases. `shelfctl open book-id` fetches just that one file from GitHub's CDN and opens it. Your library can be 100GB+ but you only download what you need.

**Better than Git LFS**: Tired of "exceeds GitHub's file size limit" errors? Git LFS costs money ($5/mo for 50GB) and makes clones slow. shelfctl uses GitHub Releases instead — release assets avoid Git's 100MB file limit and enable instant selective downloads. Start free; only pay if you opt into LFS or exceed GitHub's plan limits.

**No ops burden**: No database to maintain, no blob storage to configure, no servers to patch. GitHub handles availability, backups, and CDN distribution.

**Portable**: All API-based—no local repos required. Same config works on any machine. Your library is standard GitHub repos you can access from anywhere.

**Scriptable**: CLI-first design. Pipe commands, write shell scripts, integrate with your existing workflows.

---

<p align="center">
  <img src="assets/shelf3.png" alt="shelfctl features" width="800">
</p>

## How it works

- Each topic gets a GitHub repo: `shelf-programming`, `shelf-history`, etc.
- Files (PDFs, EPUBs) are stored as **Release assets** — not committed to git.
- `catalog.yml` in each repo is the source of truth for metadata.
- **Everything happens via GitHub API** — no local cloning required. Run commands from anywhere.
- **Download individual books on demand** — `shelfctl open book-id` fetches just that file from GitHub's CDN.
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

### Interactive Mode (Easiest)

Run `shelfctl` with no arguments to launch an interactive menu:

```bash
shelfctl
```

This provides a visual interface with:
- 🎯 **Guided workflows** - No need to remember commands or flags
- 📚 **Browse Library** - Visual book browser with search
- ➕ **Add Book** - File picker + metadata form
- 📊 **Status dashboard** - See shelf and book counts at a glance

See [docs/HUB.md](docs/HUB.md) for full details.

### Command-Line Mode

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
shelfctl shelve ~/Downloads/sicp.pdf --shelf programming --title "SICP" --author "Abelson & Sussman" --tags lisp,cs

# List books across all shelves
shelfctl browse --shelf programming

# Open a book — downloads just this one file (6MB), not the entire release
shelfctl open sicp

# On another machine? Same command fetches it on-demand from GitHub
shelfctl open sicp
```

---

<p align="center">
  <img src="assets/logo2.png" alt="shelfctl" width="400">
</p>

## Commands

| Command | Description |
|---------|-------------|
| `init` | Bootstrap a shelf repo and release |
| `shelves` | Validate all configured shelves |
| `browse` | Browse your library (interactive TUI or text) |
| `info <id>` | Show metadata and cache status |
| `open <id>` | Open a book (auto-downloads if needed) |
| `shelve <file\|url>` | Add a book to your library |
| `move <id>` | Move between releases or shelves |
| `split` | Interactive wizard to split a shelf |
| `migrate one` | Migrate a single file from an old repo |
| `migrate batch` | Migrate a queue of files |
| `migrate scan` | List files in a source repo |
| `import` | Import all books from another shelf |

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

See [`config.example.yml`](config.example.yml) for a complete example.

---

## Documentation

- **[Tutorial](docs/TUTORIAL.md)** - Step-by-step walkthrough from installation to advanced workflows
- **[Architecture Guide](docs/ARCHITECTURE.md)** - How shelves work, organization strategies, and scaling
- **[Interactive Hub](docs/HUB.md)** - Guide to the interactive TUI menu
- **[Commands Reference](docs/COMMANDS.md)** - Complete documentation for all commands
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Technical Spec](docs/SPEC.md)** - Architecture and configuration schema
- **[Contributing](CONTRIBUTING.md)** - Development guidelines

---

## Note

This tool manages user-provided files in user-owned GitHub repos and releases. It does not distribute content. Designed for personal document libraries (PDF/EPUB/etc.).

---

## Support This Project

<p align="center">
  <a href="https://github.com/blackwell-systems/shelfctl">
    <img src="assets/enjoying.png" alt="Enjoying shelfctl? Star the repo!" width="500">
  </a>
</p>

If you find shelfctl useful:
- ⭐ **Star the repo** on GitHub
- 🐛 **Report issues** or suggest features
- 🤝 **Contribute** improvements (see [CONTRIBUTING.md](CONTRIBUTING.md))
- ☕ **Buy me a coffee** via the badge above

---

## License

MIT - See [LICENSE](LICENSE) for details
