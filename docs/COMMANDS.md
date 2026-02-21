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
- Color-coded indicators (green ✓ = cached)
- **Interactive actions:**
  - `enter` - Show detailed book information
  - `o` - Open book (downloads if needed, opens with system viewer)
  - `g` - Download to cache only (for offline access)
  - `q` - Quit browser

The browser auto-downloads books when opening if not cached and shows progress.

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

## delete-book

Remove a book from your library.

```bash
shelfctl delete-book [id] [flags]
```

### Interactive Mode (no arguments)

When run in a terminal with no arguments, `delete-book` shows an interactive book picker:

```bash
# Interactive picker
shelfctl delete-book
```

### Flags

- `--shelf`: Specify shelf if book ID is ambiguous
- `--yes`: Skip confirmation prompt

### Examples

```bash
# Interactive mode - shows picker
shelfctl delete-book

# Direct deletion
shelfctl delete-book sicp --shelf programming

# Skip confirmation
shelfctl delete-book sicp --yes
```

### What it does

1. Finds the book in your library
2. Shows book details and warning
3. Requires confirmation (type book ID to confirm)
4. Deletes the GitHub release asset (PDF/EPUB file)
5. Removes entry from catalog.yml
6. Clears from local cache if present
7. Commits updated catalog

**Warning:** This is a destructive operation and cannot be easily undone.

---

## edit-book

Edit metadata for a book in your library.

```bash
shelfctl edit-book [id] [flags]
```

### Interactive Mode (no arguments)

When run in a terminal with no arguments, `edit-book` shows an interactive book picker followed by a form:

```bash
# Interactive mode - shows picker then form
shelfctl edit-book
```

The form pre-populates with current metadata values for easy editing.

### Flags

- `--shelf`: Specify shelf if book ID is ambiguous
- `--title`: New title
- `--author`: New author
- `--year`: Publication year
- `--add-tag`: Add tags (comma-separated, can be used multiple times)
- `--rm-tag`: Remove tags (comma-separated, can be used multiple times)

### Examples

```bash
# Interactive mode - form with current values
shelfctl edit-book design-patterns

# Update title only
shelfctl edit-book design-patterns --title "Design Patterns (Gang of Four)"

# Update author and year
shelfctl edit-book gopl --author "Donovan & Kernighan" --year 2015

# Add tags incrementally
shelfctl edit-book sicp --add-tag favorites --add-tag classics

# Remove a tag
shelfctl edit-book sicp --rm-tag draft

# Combine multiple changes
shelfctl edit-book gopl --title "The Go Programming Language" --add-tag reference
```

### What it does

1. Finds the book in your library
2. Shows interactive form (TUI) or applies flag changes (CLI)
3. Updates catalog.yml with new metadata
4. Commits changes to GitHub
5. Asset file remains unchanged (only metadata updates)

### What you can edit

- Title
- Author
- Year
- Tags (add/remove incrementally or replace all)

### What you cannot edit

- ID (used for commands and asset naming)
- Format (tied to the file)
- Checksum (tied to file content)
- Asset filename (tied to the file)

**Note:** This only updates catalog metadata. The actual PDF/EPUB file is not modified or re-uploaded.

---

## delete-shelf

Remove a shelf from your configuration.

```bash
shelfctl delete-shelf [name] [flags]
```

### Interactive Mode (no arguments)

When run with no arguments, shows an interactive shelf picker:

```bash
# Interactive picker
shelfctl delete-shelf
```

### Flags

- `--delete-repo`: Also delete the GitHub repository (DESTRUCTIVE)
- `--yes`: Skip confirmation prompt

### Examples

```bash
# Remove from config only (keeps repo)
shelfctl delete-shelf old-books

# Remove AND delete the GitHub repo permanently
shelfctl delete-shelf old-books --delete-repo --yes
```

### What it does

By default (without `--delete-repo`):
1. Removes shelf from your config file
2. **Keeps** the GitHub repository and all books
3. You can re-add the shelf later with `shelfctl init`

With `--delete-repo`:
1. Removes shelf from your config file
2. **Permanently deletes** the GitHub repository
3. **Deletes all books** (release assets)
4. **Deletes all catalog history**
5. **This cannot be undone**

### Interactive Prompts

When run interactively, you'll see:
1. Shelf picker (if no name provided)
2. Repository choice:
   - Keep it (default) - Remove from config only
   - Delete permanently - Delete repository and all books
3. Confirmation (type shelf name to confirm)

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

## index

Generate a local HTML index for browsing cached books in a web browser.

```bash
shelfctl index
```

### What it does

1. Scans all configured shelves
2. Collects books that are in your local cache
3. Generates `index.html` in your cache directory (`~/.local/share/shelfctl/cache/index.html`)
4. Includes cover images if available (catalog covers or auto-extracted thumbnails)

### Features

- **Visual book grid** with covers and metadata
- **Real-time search/filter** by title, author, or tags (no server needed)
- **Organized by shelf** sections
- **Click books to open** with system viewer (file:// links)
- **Responsive layout** for mobile/desktop
- **Dark theme** matching shelfctl aesthetic

### Usage

```bash
# Generate index
shelfctl index

# Open in browser (macOS)
open ~/.local/share/shelfctl/cache/index.html

# Open in browser (Linux)
xdg-open ~/.local/share/shelfctl/cache/index.html
```

### Notes

- Shows only cached books (download books first to include them)
- Works without running shelfctl - just open in any browser
- Updates automatically each time you run `shelfctl index`
- Useful for offline browsing or sharing your library locally

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
