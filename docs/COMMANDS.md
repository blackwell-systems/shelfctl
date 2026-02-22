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
- `--create-release`: Create the initial release tag (default: true)
- `--private`: Make the repository private (default: true, only applies with `--create-repo`)
- `--release`: Release tag name (default: from config, usually `library`)
- `--owner`: Repository owner (default: from config)

### Examples

```bash
# Create new private shelf with repo and release (default)
shelfctl init --repo shelf-programming --name programming --create-repo --create-release

# Create public shelf
shelfctl init --repo shelf-public --name public --create-repo --create-release --private=false

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
shelfctl shelves [--fix]
```

### Flags

- `--fix`: Automatically create missing catalog.yml files or releases

### Output

For each shelf:
- Shelf name and repository
- Repository status (exists on GitHub?)
- Catalog.yml status (exists? book count)
- Release status (exists?)

### Examples

```bash
# View all shelves with validation
shelfctl shelves

# Validate and auto-repair issues
shelfctl shelves --fix
```

### Example output

```
Shelf: programming  (user/shelf-programming)
  repo:        ok
  catalog.yml: ok (15 books)
  release(library): ok  id=123456789

Shelf: history  (user/shelf-history)
  repo:        ok
  catalog.yml: ok (empty)
  release(library): ok  id=987654321
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
- `--force`: Skip duplicate checks and overwrite existing assets

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

# Force overwrite existing asset
shelfctl shelve ~/Downloads/sicp-v2.pdf \
  --shelf programming \
  --id sicp \
  --title "SICP (Updated Edition)" \
  --force
```

### What it does

1. Interactive mode: Launches shelf/file pickers and metadata form
2. Downloads/reads the source file
3. Computes SHA256 checksum and size
4. Extracts PDF cover thumbnail automatically (if pdftoppm installed)
5. Collects metadata (interactively or from flags)
6. Checks for duplicates (SHA256 and asset name, skipped with `--force`)
7. Uploads file as GitHub release asset
8. Updates `catalog.yml` with metadata
9. Commits and pushes catalog changes

**Note:** The `--force` flag bypasses duplicate checks and will overwrite existing assets with the same name.

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

### Flags

- `--shelf`: Specify shelf if book ID exists in multiple shelves

### Examples

```bash
# Find book across all shelves
shelfctl info sicp

# Specify shelf if ID is ambiguous
shelfctl info paper-2024 --shelf research
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
shelfctl open <id> [flags]
```

### Arguments

- `id`: Book ID (searches across all shelves)

### Flags

- `--shelf`: Specify shelf if book ID exists in multiple shelves
- `--app`: Application to open the file with (overrides system default)

### Examples

```bash
# Open with system default viewer
shelfctl open sicp

# Specify shelf if ID is ambiguous
shelfctl open paper-2024 --shelf research

# Open with specific application
shelfctl open sicp --app "Preview"

# Open with command-line viewer
shelfctl open doc --app "less"
```

### What it does

1. Downloads book if not in cache
2. Verifies checksum
3. Opens file with specified application or system default
   - macOS: uses `open` (or specified app)
   - Linux: uses `xdg-open` (or specified app)
   - Windows: uses `start` (or specified app)

---

## move

Move a book between releases or shelves.

**Interactive mode:** Run `shelfctl move` with no arguments to use the guided workflow:
- Pick which book to move
- Choose between different shelf or different release
- Select destination shelf (if moving to different shelf)
- Optionally specify release tag
- Confirm before executing

**Command line mode:**

```bash
shelfctl move <id> [flags]
```

### Flags

- `--shelf`: Source shelf (if book ID is ambiguous)
- `--to-shelf`: Target shelf name
- `--to-release`: Target release tag
- `--keep-old`: Keep the original asset (don't delete after copying)
- `--dry-run`: Show what would happen without making changes

### Examples

```bash
# Interactive mode (pick book, choose destination)
shelfctl move

# Move to different release in same shelf
shelfctl move book-id --to-release 2024

# Move to different shelf
shelfctl move book-id --to-shelf history

# Move to different shelf and release
shelfctl move book-id --to-shelf history --to-release archive

# Keep original asset when moving
shelfctl move book-id --to-shelf backup --keep-old

# Preview move operation without executing
shelfctl move book-id --to-shelf history --dry-run

# Specify source shelf if ID is ambiguous
shelfctl move paper-2024 --shelf research --to-shelf archive
```

### What it does

1. Downloads book from source
2. Uploads to target release
3. Updates target catalog
4. Removes from source catalog
5. Optionally deletes source asset (use `--keep-old` to preserve)
6. Commits both catalog changes

**Note:** Use `--dry-run` to preview changes before executing. Use `--keep-old` to copy rather than move (keeps the original asset).

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

### Browser Compatibility

The index uses `file://` protocol links to open books directly from your filesystem. Browser support varies due to security restrictions:

- **Safari** (macOS): ✅ Generally works - allows file:// navigation from file:// pages
- **Firefox**: ⚠️ May work by default, or may require `security.fileuri.strict_origin_policy=false` in about:config
- **Chrome/Edge**: ❌ Most restrictive - blocks file:// navigation for security reasons

**If clicking doesn't open books:**
1. Use Safari for best compatibility
2. Right-click book card → "Copy Link Address" → open in terminal: `open <paste-path>`
3. Use the TUI browser instead: `shelfctl browse` (always works, no restrictions)

---

## import

Import books from another shelfctl shelf.

```bash
shelfctl import --from owner/repo --shelf TARGET [flags]
```

### Flags

- `--from` (required): Source shelf as owner/repo
- `--shelf` (required): Destination shelf name
- `--release`: Destination release tag (default: shelf's default_release)
- `--dry-run`: Show what would be imported without doing it
- `-n`: Limit number of books to import
- `--no-push`: Update catalog locally only

### Examples

```bash
# Import all books from another user's shelf
shelfctl import --from other-user/shelf-programming --shelf programming

# Import to specific release
shelfctl import --from other-user/shelf-research --shelf papers --release archive

# Preview import without executing
shelfctl import --from other-user/shelf-books --shelf library --dry-run

# Limit import batch size
shelfctl import --from other-user/shelf-prog --shelf cs -n 20
```

### What it does

1. Reads source catalog from owner/repo
2. For each book:
   - Downloads from source
   - Uploads to target release
   - Adds to target catalog
4. Commits target catalog

Note: Source catalog is not modified. This is a copy operation.

---

## completion

Generate shell autocompletion scripts for shelfctl.

```bash
shelfctl completion [bash|zsh|fish|powershell]
```

### Supported Shells

- `bash` - Bourne Again Shell
- `zsh` - Z Shell
- `fish` - Friendly Interactive Shell
- `powershell` - PowerShell

### Setup Instructions

#### Bash

```bash
# Load completions in current session
source <(shelfctl completion bash)

# Install permanently (Linux)
shelfctl completion bash > /etc/bash_completion.d/shelfctl

# Install permanently (macOS with Homebrew)
shelfctl completion bash > $(brew --prefix)/etc/bash_completion.d/shelfctl
```

Requires `bash-completion` package to be installed.

#### Zsh

```bash
# Load completions in current session
source <(shelfctl completion zsh)

# Install permanently
shelfctl completion zsh > "${fpath[1]}/_shelfctl"
```

Then restart your shell or run `compinit`.

#### Fish

```bash
# Install permanently
shelfctl completion fish > ~/.config/fish/completions/shelfctl.fish
```

#### PowerShell

```powershell
# Add to PowerShell profile
shelfctl completion powershell | Out-String | Invoke-Expression

# Save to profile permanently
shelfctl completion powershell >> $PROFILE
```

### What it provides

- Command name completion (`shelfctl bro<tab>` → `shelfctl browse`)
- Flag completion (`shelfctl shelve --sh<tab>` → `shelfctl shelve --shelf`)
- Shelf name completion (completes configured shelf names)
- Book ID completion where applicable

### Examples

```bash
# Generate bash completion script
shelfctl completion bash

# Generate zsh completion script
shelfctl completion zsh > _shelfctl
```
