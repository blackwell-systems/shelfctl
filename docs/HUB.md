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

### Available Operations

The hub currently provides access to:

**Browse Library**
- Launch the interactive book browser
- View all books across shelves
- Navigate with keyboard (↑/↓ or j/k)
- Filter and search (press `/`)
- See cache status

**Add Book**
- Add a new book with guided workflow
- Shelf picker (if multiple shelves)
- File browser starting in ~/Downloads
- Metadata form with smart defaults
- Automatic upload and cataloging

**Quit**
- Exit shelfctl cleanly

### Additional Commands

All other shelfctl commands remain available via direct invocation:

```bash
shelfctl info <id>       # View book details
shelfctl open <id>       # Open a book
shelfctl move <id>       # Move books between shelves
shelfctl split           # Split a large shelf
shelfctl migrate         # Import from old repos
shelfctl import          # Copy from another shelf
shelfctl delete-shelf    # Remove a shelf
```

Run `shelfctl --help` to see all available commands.

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
│  › Browse Library    View and search your books         │
│                                                          │
│    Add Book         Add a new book to your library      │
│                                                          │
│    Quit             Exit shelfctl                       │
│                                                          │
│                                                          │
│  ↑/↓: navigate  enter: select  /: filter  q: quit       │
└─────────────────────────────────────────────────────────┘
```

Clean, focused, and functional. Additional commands are available via `shelfctl <command>`.

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
- ✅ Smart welcome with status checks
- ✅ Interactive init workflow
- ✅ Architecture help system
- ✅ Clean, focused menu (no "coming soon" clutter)
- ✅ Auto-generated shelf READMEs
- ✅ Delete shelf command

**Future Enhancements:**

As we build out Phase 2, new operations will be added to the hub menu:
- Open book picker (fuzzy search)
- Info viewer (book details)
- Shelves dashboard (status overview)
- Move wizard (reorganize books)
- Import wizard (copy from shelves)
- Split wizard (organize large shelves)
- Migrate wizard (import from old repos)

For now, these operations remain available via direct command invocation.

## Feedback

As you use the hub, consider what would make it more useful:
- Which operations do you use most?
- What information should the status bar show?
- What keyboard shortcuts would be helpful?
- What's confusing or could be clearer?

Open issues at: https://github.com/blackwell-systems/shelfctl/issues
