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

**Organize the PDFs and books you already have scattered across GitHub.**

Most devs end up with a mess: one monolithic "books" repo with hundreds of PDFs, or files scattered across random repos and gists. Eventually you hit GitHub's 100MB limit, or you bolt on Git LFS and discover it's expensive and annoying for a personal library.

shelfctl fixes this by storing files as GitHub Release assets (not Git commits) and keeping metadata in a simple `catalog.yml`. That means you can split a bloated repo into topic-based shelves, migrate books out of existing repos, and search + download on demand—without moving your library to a new service.

Your GitHub account already gives you reliable distribution and storage primitives. shelfctl turns them into a library:
- Release assets for the PDFs/EPUBs
- `catalog.yml` for searchable metadata
- one repo per shelf (`shelf-programming`, `shelf-history`, …)

Browse and search from the CLI, fetch only what you need, and open books immediately. Your library stays portable, backed by normal git repos.

Zero infrastructure. Free by default—only pay if you choose Git LFS or exceed GitHub plan limits. Works anywhere you can use GitHub.

---

## Why

### On-demand downloads (no cloning)
Fetch a single book without cloning a repo or pulling a whole archive.
`shelfctl open <book-id>` downloads *only that file* from GitHub's CDN and opens it. Your library can be huge, but you only download what you actually read.

### Better than committing binaries (and often better than LFS)
If you've hit GitHub's 100MB file limit, you've already felt the pain of storing PDFs in git history. Git LFS can work, but it adds cost and makes clones heavier.

shelfctl stores PDFs/EPUBs as **GitHub Release assets** instead of git commits:
- avoids bloating git history
- avoids repo clone overhead for large libraries
- enables selective, per-file downloads

Free by default. You only pay if *you* choose LFS or exceed whatever limits apply to your GitHub plan.

### No ops burden
No database, no object storage configuration, no servers. GitHub handles hosting, availability, and distribution.

### Portable by design
Everything is API-driven. No local repos required. The same config works on any machine where you can authenticate to GitHub. Your library remains normal GitHub repos.

### Scriptable
CLI-first. Pipe output, write shell scripts, and integrate shelfctl into your existing workflows.

---

<p align="center">
  <img src="assets/shelf3.png" alt="shelfctl features" width="800">
</p>

## How it works

- **One repo per topic shelf**
  Create shelves like `shelf-programming`, `shelf-history`, etc.

- **Books live in Releases, not git history**
  PDFs/EPUBs are uploaded as **GitHub Release assets** (not committed to the repo).

- **`catalog.yml` is the source of truth**
  Each shelf repo contains a `catalog.yml` that stores searchable metadata and maps book IDs to release assets.

- **API-driven, no cloning required**
  shelfctl interacts with GitHub via the API (via `gh`), so you can run commands from any machine without cloning repos.

- **On-demand, per-book downloads**
  `shelfctl open <book-id>` downloads *only that one file* from GitHub's CDN and opens it.

- **Full lifecycle management**
  shelfctl supports the workflow end-to-end: **add**, **get**, **open**, **migrate**, **split**, and more.

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

### Optional: PDF Cover Thumbnails

For automatic cover extraction from PDFs, install poppler:

```bash
# macOS
brew install poppler

# Ubuntu/Debian
sudo apt-get install poppler-utils

# Fedora/RHEL
sudo dnf install poppler-utils

# Arch Linux
sudo pacman -S poppler
```

Not required - shelfctl works fine without it. Covers are extracted automatically when you download PDFs if poppler is installed.

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

# Create organized shelves (private by default)
shelfctl init --repo shelf-programming --name programming --create-repo --create-release
shelfctl init --repo shelf-research --name research --create-repo --create-release

# Or make a shelf public
shelfctl init --repo shelf-public --name public --create-repo --create-release --private=false

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
| `delete-shelf` | Remove a shelf from configuration |
| `browse` | Browse your library (interactive TUI or text) |
| `index` | Generate local HTML index for web browsing |
| `info <id>` | Show metadata and cache status |
| `open <id>` | Open a book (auto-downloads if needed) |
| `shelve <file\|url>` | Add a book to your library |
| `edit-book <id>` | Update metadata for a book |
| `delete-book <id>` | Remove a book from your library |
| `move <id>` | Move between releases or shelves |
| `split` | Interactive wizard to split a shelf |
| `migrate one` | Migrate a single file from an old repo |
| `migrate batch` | Migrate a queue of files |
| `migrate scan` | List files in a source repo |
| `import` | Import all books from another shelf |
| `completion` | Generate shell autocompletion scripts |

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

<p align="center">
  <img src="assets/logo3.png" alt="shelfctl" width="400">
</p>

## Documentation

- **[Tutorial](docs/TUTORIAL.md)** - Step-by-step walkthrough from installation to advanced workflows
- **[Architecture Guide](docs/ARCHITECTURE.md)** - How shelves work, organization strategies, schemas, and configuration
- **[Interactive Hub](docs/HUB.md)** - Guide to the interactive TUI menu
- **[Commands Reference](docs/COMMANDS.md)** - Complete documentation for all commands
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Contributing](CONTRIBUTING.md)** - Development guidelines

---

## ⚖️ Disclaimer

shelfctl is a specialized management tool designed to help developers organize their personal document libraries using the GitHub API. By using this software, you agree to the following:

**Platform Compliance**: You are solely responsible for ensuring that your use of GitHub as a storage backend complies with GitHub's Terms of Service and Acceptable Use Policies. The author(s) of shelfctl are not responsible for any account suspensions or data removal by GitHub.

**Content Responsibility**: shelfctl does not provide, host, or distribute any content. Users are responsible for ensuring they have the legal right to store and distribute any files they upload to their own GitHub repositories and releases.

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
