# Command Reference

Complete reference for all `shelfctl` commands.

## Global flags

```
--config string       Config file path (default: ~/.config/shelfctl/config.yml)
--no-color           Disable colored output
--no-interactive     Disable interactive TUI mode
```

---

## init

Bootstrap a shelf repository and initial release.

```bash
shelfctl init --repo REPO --name NAME [flags]
```

### Flags

- `--repo` (required): Repository name (e.g., `shelf-programming`)
- `--name` (required): Shelf name for config (e.g., `programming`)
- `--create-repo`: Create the GitHub repository if it doesn't exist
- `--create-release`: Create the initial release tag
- `--release`: Release tag name (default: from config, usually `library`)
- `--owner`: Repository owner (default: from config)

### Examples

```bash
# Create new shelf with repo and release
shelfctl init --repo shelf-programming --name programming --create-repo --create-release

# Initialize existing repo as a shelf
shelfctl init --repo existing-repo --name mybooks
```

### What it does

1. Optionally creates the GitHub repository
2. Optionally creates the initial release tag
3. Adds shelf to `~/.config/shelfctl/config.yml`
4. Creates empty `catalog.yml` in the repo

---

## shelves

Validate all configured shelves and show their status.

```bash
shelfctl shelves
```

### Output

For each shelf:
- Shelf name and repo
- Whether repo exists on GitHub
- Whether catalog.yml exists
- Number of books in catalog
- Validation errors (if any)

### Example output

```
[OK] programming (shelf-programming)
  catalog: catalog.yml (15 books)

[X] history (shelf-history)
  error: repository not found
```

---

## shelve

Add a book to your library.

```bash
shelfctl shelve [file|url|github:owner/repo@ref:path] [flags]
```

### Interactive Mode (no arguments)

When run in a terminal with no arguments, `shelve` launches a fully guided workflow:

1. **Select shelf** - Choose from configured shelves
2. **Pick file** - Browse filesystem (starts in ~/Downloads, shows .pdf/.epub/.mobi/.djvu)
3. **Enter metadata** - Fill form with title, author, tags, ID
4. **Upload** - Automatic upload to GitHub and catalog update

```bash
# Fully interactive
shelfctl shelve
```

### Arguments

Source can be:
- Local file: `~/Downloads/book.pdf`
- HTTP URL: `https://example.com/book.pdf`
- GitHub repo: `github:user/repo@main:path/to/book.pdf`
- None: launches file picker (interactive mode)

### Flags

- `--shelf`: Target shelf name (interactive picker if omitted)
- `--title`: Book title (prompts if omitted)
- `--author`: Author name
- `--year`: Publication year
- `--tags`: Comma-separated tags (e.g., `cs,algorithms`)
- `--id`: Book ID (default: prompt or slugified title)
- `--id-sha12`: Use first 12 chars of SHA256 as ID
- `--release`: Target release tag (default: shelf's default)
- `--asset-name`: Override asset filename
- `--no-push`: Update catalog locally without pushing

### Examples

```bash
# Fully interactive (no arguments)
shelfctl shelve

# Interactive form with file provided
shelfctl shelve ~/Downloads/sicp.pdf

# Non-interactive with all metadata
shelfctl shelve ~/Downloads/sicp.pdf \
  --shelf programming \
  --title "Structure and Interpretation of Computer Programs" \
  --author "Abelson & Sussman" \
  --tags lisp,cs,textbook

# Add from URL
shelfctl shelve https://example.com/paper.pdf \
  --shelf research \
  --title "Important Paper" \
  --year 2024

# Add from GitHub repo
shelfctl shelve github:user/books@main:pdfs/algo.pdf \
  --shelf cs \
  --title "Algorithm Design"

# Use SHA-based ID for uniqueness
shelfctl shelve book.pdf --shelf fiction --id-sha12 --title "Novel"
```

### What it does

1. Interactive mode: Launches shelf/file pickers and metadata form
2. Downloads/reads the source file
3. Computes SHA256 checksum and size
4. Collects metadata (interactively or from flags)
5. Checks for duplicates (SHA256 and asset name)
6. Uploads file as GitHub release asset
7. Updates `catalog.yml` with metadata
8. Commits and pushes catalog changes

---

## browse

Browse your library (interactive TUI or text output).

```bash
shelfctl browse [--shelf NAME] [--tag TAG] [--format FORMAT]
```

### Interactive Mode (TUI)

When run in a terminal, `browse` shows an interactive browser with:
- Keyboard navigation (↑/↓ or j/k)
- Live filtering (press `/` to search)
- Visual display with tags and cache status
- Color-coded indicators ([OK] = cached)

Use `--no-interactive` or pipe output to get text mode.

### Flags

- `--shelf`: Filter by shelf name
- `--tag`: Filter by tag
- `--format`: Filter by format (pdf, epub, etc.)
- `--search`: Full-text search across title, author, tags

### Examples

```bash
# List all books
shelfctl browse

# List books in specific shelf
shelfctl browse --shelf programming

# Filter by tag
shelfctl browse --tag algorithms

# Combine filters
shelfctl browse --shelf programming --tag lisp
```

### Output

```
programming/sicp
  Structure and Interpretation of Computer Programs
  Abelson & Sussman
  tags: lisp, cs, textbook
  format: pdf, 6.2 MB

programming/taocp-vol1
  The Art of Computer Programming, Vol 1
  Donald Knuth
  tags: algorithms, cs
  format: pdf, 15.8 MB
```

---

## info

Show detailed metadata and cache status for a book.

```bash
shelfctl info <id>
```

### Arguments

- `id`: Book ID (searches across all shelves)

### Example

```bash
shelfctl info sicp
```

### Output

```
ID:        sicp
Title:     Structure and Interpretation of Computer Programs
Author:    Abelson & Sussman
Year:      1996
Tags:      lisp, cs, textbook
Format:    pdf
Size:      6.2 MB
SHA256:    a1b2c3d4...
Source:    github_release
  Owner:   your-username
  Repo:    shelf-programming
  Release: library
  Asset:   sicp.pdf
Cache:     cached (/Users/you/.local/share/shelfctl/cache/sicp.pdf)
Added:     2024-01-15T10:30:00Z
```

---

## open

Open a book (auto-downloads if needed).

```bash
shelfctl open <id>
```

### Arguments

- `id`: Book ID (searches across all shelves)

### Example

```bash
shelfctl open sicp
```

### What it does

1. Downloads book if not in cache (same as `open`)
2. Opens file with system default application
   - macOS: uses `open`
   - Linux: uses `xdg-open`
   - Windows: uses `start`

---

## move

Move a book between releases or shelves.

```bash
shelfctl move <id> [flags]
```

### Flags

- `--to-shelf`: Target shelf name
- `--to-release`: Target release tag
- `--keep-asset`: Keep the original asset (don't delete)

### Examples

```bash
# Move to different release in same shelf
shelfctl move book-id --to-release 2024

# Move to different shelf
shelfctl move book-id --to-shelf history

# Move to different shelf and release
shelfctl move book-id --to-shelf history --to-release archive

# Keep original asset when moving
shelfctl move book-id --to-shelf backup --keep-asset
```

### What it does

1. Downloads book from source
2. Uploads to target release
3. Updates target catalog
4. Removes from source catalog
5. Optionally deletes source asset
6. Commits both catalog changes

---

## split

Interactive wizard to split a shelf into multiple shelves.

```bash
shelfctl split
```

### Interactive workflow

1. Select source shelf
2. Define target shelves
3. Map books to target shelves (by tag, title pattern, etc.)
4. Review migration plan
5. Execute split

Use this when a shelf grows too large or topics diverge.

---

## migrate one

Migrate a single file from an old repository.

```bash
shelfctl migrate one <source> <path> --shelf NAME [flags]
```

### Arguments

- `source`: Source repo in format `owner/repo@ref`
- `path`: Path to file in source repo

### Flags

- `--shelf` (required): Target shelf name
- `--title`: Book title (prompts if omitted)
- `--author`, `--year`, `--tags`: Metadata

### Example

```bash
shelfctl migrate one user/old-books@main books/sicp.pdf \
  --shelf programming \
  --title "SICP" \
  --tags lisp,cs
```

---

## migrate batch

Migrate multiple files from a queue.

```bash
shelfctl migrate batch <queue-file> [flags]
```

### Arguments

- `queue-file`: File with one migration per line (see format below)

### Flags

- `--n`: Process only first N entries
- `--continue`: Skip entries already in ledger
- `--ledger`: Ledger file path (default: `.shelfctl-ledger.txt`)

### Queue file format

```
owner/repo@ref:path/to/file.pdf shelf=programming title="..." tags=cs,algo
owner/repo@ref:another.pdf shelf=history title="..."
```

### Example

```bash
# Generate queue from old repo
shelfctl migrate scan --source user/old-books > queue.txt

# Edit queue.txt to add metadata

# Migrate in batches
shelfctl migrate batch queue.txt --n 10 --continue

# Continue from where you left off
shelfctl migrate batch queue.txt --continue
```

### Ledger

The ledger file tracks completed migrations to support resumption. Format:

```
[OK] owner/repo@ref:path/to/file.pdf shelf=programming id=book-id
[X] owner/repo@ref:failed.pdf error: file not found
```

---

## migrate scan

List all files in a source repository.

```bash
shelfctl migrate scan --source owner/repo[@ref]
```

### Flags

- `--source` (required): Source repo in format `owner/repo[@ref]` (default ref: main)

### Example

```bash
# Scan repo and generate queue
shelfctl migrate scan --source user/old-books > queue.txt

# Scan specific ref
shelfctl migrate scan --source user/old-books@archive > archive-queue.txt
```

### Output format

One file per line:

```
user/old-books@main:books/programming/sicp.pdf
user/old-books@main:books/history/rome.pdf
user/old-books@main:papers/algo.pdf
```

---

## import

Import books from another shelfctl shelf.

```bash
shelfctl import --from-shelf SOURCE --to-shelf TARGET [flags]
```

### Flags

- `--from-shelf` (required): Source shelf name
- `--to-shelf` (required): Target shelf name
- `--from-owner`: Source owner (default: config)
- `--from-repo`: Source repo (default: from shelf config)
- `--filter-tag`: Only import books with this tag
- `--dry-run`: Show what would be imported without doing it

### Examples

```bash
# Import all books from another shelf
shelfctl import --from-shelf programming --to-shelf cs

# Import only specific tag
shelfctl import --from-shelf research --to-shelf ml --filter-tag machine-learning

# Import from different owner
shelfctl import --from-shelf programming --to-shelf my-prog --from-owner other-user

# Preview import
shelfctl import --from-shelf prog --to-shelf cs --dry-run
```

### What it does

1. Reads source catalog
2. Filters books (if --filter-tag)
3. For each book:
   - Downloads from source
   - Uploads to target release
   - Adds to target catalog
4. Commits target catalog

Note: Source catalog is not modified. This is a copy operation.
