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
- **Help** - Shows keybindings at the bottom

### Current Operations (Available Now)

✅ **Browse Library** - Launch the interactive book browser
- View all books across shelves
- Navigate with keyboard
- Filter and search
- See cache status

✅ **Add Book** - Add a new book with guided workflow
- Shelf picker (if multiple shelves)
- File browser starting in ~/Downloads
- Metadata form
- Automatic upload and cataloging

✅ **Quit** - Exit shelfctl

### Coming Soon Operations

The following operations are planned and will show "coming soon" when selected:

- 🔜 **View Shelves** - Dashboard showing all configured shelves with status
- 🔜 **Open Book** - Searchable book picker to open files
- 🔜 **Book Info** - View detailed metadata for any book
- 🔜 **Move Book** - Wizard to move books between shelves/releases
- 🔜 **Import Shelf** - Copy books from another shelfctl repository
- 🔜 **Migrate** - Import books from old non-shelfctl repos
- 🔜 **Split Shelf** - Organize large shelves into sub-categories

## First Time Use

If you haven't configured any shelves yet, the hub will show a welcome message and offer to guide you through setup:

```bash
$ shelfctl

⚠ Welcome to shelfctl!

Setup status:

  ✓ GitHub token configured
  ✗ No shelves configured

Next step: Create your first shelf

Would you like to create a shelf now? (y/n): y

📚 Let's set up your first shelf!

Tip: Type 'help' or '?' at any prompt for detailed guidance

Want to learn about shelf architecture first? (y/n/?): y

═══════════════════════════════════════════════════════════
  How Shelves Work
═══════════════════════════════════════════════════════════

Structure:
  Each shelf is a GitHub repository with:
    • catalog.yml (in Git) - Metadata for your books
    • Release assets (not in Git) - The actual PDF/EPUB files

Organization Strategy:
  1. Start broad - One shelf is often enough at first
     Example: shelf-books (general collection)

  2. Use tags - Organize books within a shelf using tags
     Example: --tags programming,golang,textbook

  3. Split later - When a shelf grows large, split it
     Use: shelfctl split (interactive wizard)

[... more help content ...]

Repository name (e.g., shelf-books): shelf-programming
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

- **↑ / ↓** or **j / k** - Navigate menu items
- **/** - Filter/search menu items
- **Enter** - Select highlighted item
- **q** or **Esc** or **Ctrl+C** - Quit

## Visual Design

```
┌─────────────────────────────────────────────────────────┐
│  shelfctl - Personal Library Manager                    │
│  3 shelves · 42 books                                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  › Browse Library          View and search your books   │
│    Add Book               Add a new book to library     │
│    View Shelves           List all shelves (coming)     │
│    Open Book              Search and open (coming)      │
│    Book Info              View metadata (coming)        │
│    Move Book              Reorganize books (coming)     │
│    Import Shelf           Copy from shelf (coming)      │
│    Migrate                Import from repo (coming)     │
│    Split Shelf            Organize shelf (coming)       │
│    Quit                   Exit shelfctl                 │
│                                                          │
│  ↑/↓: navigate  enter: select  /: filter  q: quit       │
└─────────────────────────────────────────────────────────┘
```

## Advantages of the Hub

1. **Discoverability** - See all available operations at a glance
2. **Guidance** - No need to remember command names or flags
3. **Visual feedback** - See shelf/book counts in real-time
4. **Consistent experience** - All operations follow similar patterns
5. **Faster** - No need to type commands repeatedly

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

## Implementation Status

**Phase 1 (Complete):**
- ✅ Hub menu with navigation
- ✅ Status bar showing shelf/book counts
- ✅ Integration with browse command
- ✅ Integration with shelve command
- ✅ Welcome message for first-time users

**Phase 2 (Planned):**
- 🔜 Open book picker
- 🔜 Info viewer
- 🔜 Shelves dashboard

**Phase 3 (Planned):**
- 🔜 Move wizard
- 🔜 Import wizard
- 🔜 Split wizard
- 🔜 Migrate wizard

## Feedback

As you use the hub, consider what would make it more useful:
- Which operations do you use most?
- What information should the status bar show?
- What keyboard shortcuts would be helpful?
- What's confusing or could be clearer?

Open issues at: https://github.com/blackwell-systems/shelfctl/issues
