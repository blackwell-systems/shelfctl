# shelfctl

A Go CLI for managing a personal document library using GitHub repos and releases as storage.

**Shelf repos** (GitHub repos) hold metadata. **GitHub Release assets** hold the actual files (PDF/EPUB/etc.). No self-hosted infrastructure required.

---

## How it works

- Each topic gets a GitHub repo: `shelf-programming`, `shelf-history`, etc.
- Files (PDFs, EPUBs) are stored as **Release assets** — not committed to git.
- `catalog.yml` in each repo is the source of truth for metadata.
- `shelfctl` manages the whole lifecycle: add, get, open, migrate, split.

---

## Install

```bash
go install github.com/YOUR_GH_OWNER/shelfctl/cmd/shelfctl@latest
```

Or build from source:

```bash
git clone https://github.com/YOUR_GH_OWNER/shelfctl
cd shelfctl
make build
```

---

## Quick start

```bash
# Configure
cp config.example.yml ~/.config/shelfctl/config.yml
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
