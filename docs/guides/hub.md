# Interactive Hub

The `shelfctl` hub is an interactive menu that provides a visual interface to all shelfctl operations.

## Launching the Hub

Simply run `shelfctl` with no arguments in a terminal:

```bash
shelfctl
```

This will display an interactive menu showing all available operations.

## Features

### Visual Menu

The hub displays:
- **Main menu** - All available operations (Browse, Add Book, Move, etc.)
- **Status bar** - Number of configured shelves and total books
- **Keyboard navigation** - Use ↑/↓ or j/k to navigate
- **Search** - Press `/` to filter menu items
- **Command palette** - Press `ctrl+p` to fuzzy-search all actions
- **Help** - Shows keybindings at the bottom

### Scrollable Details Panel

When highlighting "View Shelves" or "Cache Info" menu items, the hub displays a details panel on the right side. For large libraries with many shelves or modified books:

- Press `tab` or `→` to focus the details panel
- Use `↑`/`↓` or `j`/`k` to scroll through content
- Press `←` or `esc` to return focus to the menu
- Focused panel shows thick cyan border and scroll position indicator
- Menu border dims when details panel is focused

This allows you to review all your shelves or cache statistics without truncation, even if the list extends beyond one screen.

### Available Operations

The hub provides access to all core operations and loops continuously until you quit:

**Browse Library**
- Launch the interactive book browser
- View all books across shelves with metadata
- Navigate with keyboard (↑/↓ or j/k)
- Filter and search in real-time (press `/`)
- See cache status (green ✓ for cached books)
- **Interactive Actions:**
  - `enter` - Show detailed book information with cover image (Kitty/Ghostty/iTerm2)
  - `o` - Open book (downloads if needed, opens with system viewer)
  - `space` - Toggle selection (checkboxes appear for multi-select)
  - `g` - Download selected books to cache (or current if none selected)
  - `x` - Remove selected books from cache (or current if none selected)
  - `s` - Sync modified books to GitHub (uploads annotations/highlights)
  - `c` - Clear all selections
  - `tab` - Toggle details panel
  - `q` - Return to hub menu
- **Multi-select workflow:**
  - Press `space` to check books for batch operations
  - Press `g` to download all selected (useful for pre-caching for offline)
  - Press `x` to remove selected from cache (frees disk space, keeps in library)
  - Press `s` to sync modified books back to GitHub (replaces release assets)
- Downloads happen in background with progress bar at bottom
- Books marked as cached when download complete
- Extracts PDF cover thumbnails automatically during download

**Generate HTML Index**
- Creates a static HTML file for browsing your library in a web browser
- Generated at `~/.local/share/shelfctl/cache/index.html`
- Features:
  - Visual book grid with covers and metadata
  - Real-time search/filter by title, author, or tags (JavaScript)
  - Clickable tag filters with word cloud interface
  - Sort options: Recently Added, Title, Author, Year
  - Organized by shelf sections
  - Click books to open with system viewer (file:// links)
  - Responsive layout for mobile/desktop
  - Dark theme matching shelfctl aesthetic
- Shows only cached books (download books first to include them)
- Works without running shelfctl - just open in any browser
- Perfect for offline browsing or sharing your library locally
- Returns to hub menu after generation

**Add Book**
- Add a new book with guided workflow
- Shelf picker (if multiple shelves)
- File browser starting in current directory
- Filters for supported formats (.pdf, .epub, .mobi, .djvu)
- Metadata form with smart defaults (auto-extracts title/author from PDFs)
- "Cache locally" checkbox (enabled by default) - book available immediately after upload
- Automatic upload and cataloging
- Returns to hub menu after completion

**Edit Book**
- Interactive book picker followed by metadata form
- Pre-populates form with current values
- Update title, author, year, and tags
- Only updates catalog metadata (asset file unchanged)
- Returns to hub menu after completion
- Hidden when no books exist

**Delete Book**
- Interactive book picker showing all books
- Shows book details before deletion
- Safety confirmation (type book ID to confirm)
- Removes from catalog.yml and deletes GitHub release asset
- Clears from local cache automatically
- Returns to hub menu after completion
- Hidden when no books exist

**Create Shelf**
- Interactive form for adding a new shelf repository
- Collects shelf name and repository name
- Checkboxes for configuration options:
  - Create GitHub repository (default: yes)
  - Make repository private (default: yes)
- Creates repo via GitHub API with 'library' release, generates README.md, adds to config
- No need to exit to CLI or remember `init` command syntax
- Returns to hub menu after completion

**Delete Shelf**
- Interactive shelf picker
- Clear numbered choices:
  - Keep repository (remove from config only)
  - Delete permanently (repository and all books)
- Safety confirmation (type shelf name to confirm)
- Shows exactly what will be deleted
- Returns to hub menu after completion
- Hidden when no shelves exist

**Quit**
- Exit shelfctl cleanly

### Additional Commands

All other shelfctl commands remain available via direct invocation:

```bash
shelfctl status          # Library sync status overview
shelfctl search <query>  # Search by title, author, or tags
shelfctl tags            # List tags with counts
shelfctl info <id>       # View book details
shelfctl open <id>       # Open a book
shelfctl move <id>       # Move books between shelves
shelfctl split           # Split a large shelf
shelfctl migrate         # Import from old repos
shelfctl import          # Copy from another shelf
```

Run `shelfctl --help` to see all available commands.

## First Time Use

If you haven't configured any shelves yet, the hub will show a welcome message and offer to guide you through setup:

```bash
$ shelfctl

Welcome to shelfctl!

Setup status:

  ✓ GitHub token configured
  ✗ No shelves configured

Next step: Create your first shelf

Would you like to create a shelf now? (y/n): y

📚 Let's set up your first shelf!

Tip: Type 'help' or '?' at any prompt for detailed guidance

Want to learn about shelf architecture first? (y/n/?): n

This will create a GitHub repository to store your books.

GitHub repository name (e.g., shelf-books) [?=help]: shelf-programming

The shelf name is a short nickname used in commands like:
  shelfctl shelve book.pdf --shelf programming

Shelf name for commands (default: programming) [?=help]:

Summary:
  GitHub repository:  your-username/shelf-programming
  Release tag:        library
  Shelf name (config): programming

You'll use the shelf name in commands: shelfctl shelve --shelf programming

Proceed? (y/n): y
Shelf name (e.g., books): programming

This will create:
  • GitHub repository: your-username/shelf-programming
  • Release tag: library
  • Shelf name: programming

Proceed? (y/n): y

Creating repo your-username/shelf-programming …
✓ Created https://github.com/your-username/shelf-programming
Creating release 'library' in your-username/shelf-programming …
✓ Release ready: https://github.com/your-username/shelf-programming/releases/tag/library
Creating README.md …
✓ README.md created
✓ Added shelf "programming" to config
  config: /Users/you/.config/shelfctl/config.yml
  owner:  your-username
  repo:   shelf-programming

✓ Shelf created successfully!

What's next?
  1. Add your first book:
     shelfctl shelve

  2. Or run the interactive menu:
     shelfctl
```

### Setup Status Indicators

The welcome message shows visual status for each requirement:

- ✓ **GitHub token configured** - Token is set and ready
- ✗ **GitHub token not found** - Need to `export GITHUB_TOKEN=...`
- ✓ **N shelf(s) configured** - Shelves are set up
- ✗ **No shelves configured** - Need to run `shelfctl init` or accept the guided workflow

## Non-Interactive Mode

The hub requires a terminal (TTY). If you:
- Pipe output: `shelfctl | grep ...`
- Redirect output: `shelfctl > file.txt`
- Use `--no-interactive` flag: `shelfctl --no-interactive`

It will display the standard CLI help instead.

## Keyboard Controls

**Main Menu:**
- **↑ / ↓** or **j / k** - Navigate menu items
- **/** - Filter/search menu items
- **Ctrl+P** - Open command palette (fuzzy-search all actions)
- **Enter** - Select highlighted item
- **Tab** or **→** - Focus details panel (when available)
- **q** or **Esc** or **Ctrl+C** - Quit

**Details Panel** (when focused):
- **↑ / ↓** or **j / k** - Scroll content up/down
- **←** or **Esc** - Return focus to main menu

## Visual Design

```
┌─────────────────────────────────────────────────────────────────────┐
│  shelfctl - Personal Library Manager                                │
│  3 shelves · 42 books                                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  › Browse Library              View and search your books           │
│                                                                      │
│    Add Book                    Add a new book to your library       │
│                                                                      │
│    Edit Book                   Update metadata for a book           │
│                                                                      │
│    Cache Info                  View cache statistics and usage      │
│                                                                      │
│    Clear Cache                 Remove books from local cache        │
│                                                                      │
│    View Shelves                Show all configured shelves          │
│                                                                      │
│    Generate HTML Index         Create web page for browsing         │
│                                                                      │
│    Add from URL                Download and add a book from URL     │
│                                                                      │
│    Move Book                   Transfer a book to another shelf     │
│                                                                      │
│    Delete Book                 Remove a book from your library      │
│                                                                      │
│    Import from Repository      Migrate books from another repo      │
│                                                                      │
│    Create Shelf                Add a new shelf repository           │
│                                                                      │
│    Delete Shelf                Remove a shelf from configuration    │
│                                                                      │
│    Quit                        Exit shelfctl                        │
│                                                                      │
│  ↑/↓: navigate  enter: select  ctrl+p: palette  /: filter  q: quit    │
└─────────────────────────────────────────────────────────────────────┘
```

Clean, focused, and functional. The menu dynamically shows only available operations (e.g., "Delete Book" is hidden when no books exist). Menu items are ordered by frequency of use. Additional commands are available via `shelfctl <command>`.

## CLI Compatibility

All operations remain available as direct commands:

```bash
# These work exactly as before
shelfctl browse
shelfctl shelve ~/book.pdf --shelf programming --title "..."
shelfctl info book-id
shelfctl move book-id --to-shelf history

# But now you can also use the hub for a guided experience
shelfctl  # launches interactive menu
```

## Feedback

As you use the hub, consider what would make it more useful:
- Which operations do you use most?
- What information should the status bar show?
- What keyboard shortcuts would be helpful?
- What's confusing or could be clearer?

Open issues at: https://github.com/blackwell-systems/shelfctl/issues
