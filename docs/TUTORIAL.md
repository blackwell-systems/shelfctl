# Tutorial

A step-by-step guide to setting up and using shelfctl.

## Prerequisites

- Go 1.21+ installed
- GitHub account
- GitHub personal access token (see Step 1 below)

## Step 1: Authenticate with GitHub

shelfctl requires a GitHub personal access token (PAT) set in the `GITHUB_TOKEN` environment variable.

**Classic PAT scopes:**
- `repo` - for private shelves
- `public_repo` - for public-only shelves

**Fine-grained PAT permissions:**
Grant **Contents** (Read/Write) and **Releases** (Read/Write) on the shelf repos you manage.

**Note**: GitHub CLI (`gh`) is not required - shelfctl uses the GitHub REST API directly.

### Option A: Using gh CLI (optional convenience)

If you already have [GitHub CLI](https://cli.github.com/) installed and authenticated:

```bash
gh auth login
export GITHUB_TOKEN=$(gh auth token)
```

Add to shell profile to persist:

```bash
# Bash
echo 'export GITHUB_TOKEN=$(gh auth token)' >> ~/.bashrc

# Zsh
echo 'export GITHUB_TOKEN=$(gh auth token)' >> ~/.zshrc
```

### Option B: Manual Token

1. Visit https://github.com/settings/tokens
2. Generate new token (classic or fine-grained)
3. Select required scopes/permissions (see above)
4. Copy the token (starts with `ghp_` or `github_pat_`)

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

Add to shell profile to persist (`~/.bashrc` or `~/.zshrc`).

**API Rate Limits**: GitHub's authenticated API allows 5,000 requests/hour. shelfctl caches downloaded book files locally; metadata is fetched from GitHub as needed. For typical personal library usage, you're unlikely to hit rate limits.

## Step 2: Install shelfctl

```bash
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest
```

Verify installation:

```bash
shelfctl --help
```

## Step 3: Set up configuration

Create the config directory:

```bash
mkdir -p ~/.config/shelfctl
```

Copy the example config:

```bash
curl -o ~/.config/shelfctl/config.yml \
  https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/config.example.yml
```

Or download from the repo and copy manually.

Edit `~/.config/shelfctl/config.yml`:

```yaml
github:
  owner: "your-github-username"  # Change this!
  token_env: "GITHUB_TOKEN"      # Environment variable to read token from

defaults:
  release: "library"
  cache_dir: "~/.local/share/shelfctl/cache"
  asset_naming: "id"

shelves: []  # Will populate via init command
```

**Security**: The token itself is never stored in the config file - only the environment variable name. shelfctl reads the token from your environment at runtime.

## Step 4: Create your first shelf

Let's create a shelf for programming books:

```bash
shelfctl init \
  --repo shelf-programming \
  --name programming \
  --create-repo
```

This will:
- Create `shelf-programming` repository on GitHub
- Create the `library` release tag
- Add an empty `catalog.yml` to the repo
- Add the shelf to your config file

Verify it worked:

```bash
shelfctl shelves
```

Output:
```
[OK] programming (shelf-programming)
  catalog: catalog.yml (0 books)
```

## Step 5: Add your first book

### Interactive mode (easiest)

Just run `shelve` with no arguments for a fully guided workflow:

```bash
shelfctl shelve
```

This launches an interactive TUI that:
1. Lets you pick the shelf (if you have multiple)
2. Shows a file browser to select your book
3. Provides a form to fill in metadata (title, author, tags, ID)
4. Uploads and catalogs automatically

### Command-line mode

Or provide all details via flags:

```bash
shelfctl shelve ~/Downloads/some-book.pdf \
  --shelf programming \
  --title "Introduction to Algorithms" \
  --author "CLRS" \
  --tags algorithms,textbook,cs \
  --year 2009
```

If you don't provide all metadata, shelfctl will prompt you.

What happened:
1. The PDF was uploaded to GitHub as a release asset
2. A catalog entry was created with metadata and SHA256 checksum
3. The catalog was committed and pushed

## Step 6: Browse your books

### Interactive mode (TUI)

Run `browse` in a terminal for an interactive browser:

```bash
shelfctl browse --shelf programming
```

This shows a visual browser with:
- Keyboard navigation (↑/↓ or j/k)
- Live filtering (press `/` to search)
- Color-coded indicators ([OK] = cached)

### Text mode

Use `--no-interactive` or pipe output for scripts:

```bash
shelfctl browse --shelf programming --no-interactive
```

Output:
```
programming/introduction-to-algorithms
  Introduction to Algorithms
  CLRS
  tags: algorithms, textbook, cs
  format: pdf, 12.4 MB
```

## Step 7: View book details

```bash
shelfctl info introduction-to-algorithms
```

This shows full metadata, source location, cache status, and checksum.

## Step 8: Open a book

```bash
shelfctl open introduction-to-algorithms
```

This will:
1. Download the book to your cache (if not already cached)
2. Verify the checksum
3. Open it with your system's default PDF viewer

The next time you open it, it will use the cached copy (no download).

You can also browse from the interactive TUI:

```bash
shelfctl browse
# Press 'o' on a book to open it
```

### Generate HTML index for web browsing

Create a visual web page to browse your library in any browser:

```bash
shelfctl index
```

This generates `~/.local/share/shelfctl/cache/index.html` with:
- Visual book grid with covers and metadata
- Real-time search/filter (no server needed)
- Click books to open them
- Works offline, no shelfctl needed

Open the index:

```bash
# macOS
open ~/.local/share/shelfctl/cache/index.html

# Linux
xdg-open ~/.local/share/shelfctl/cache/index.html
```

The index shows only cached books, so download books first to include them.

### Managing cache

Your local cache stores downloaded books at `~/.cache/shelfctl/` (or `~/.local/share/shelfctl/cache`). You can manage cache without affecting your shelf catalog or release assets:

```bash
# View cache statistics
shelfctl cache info

# View stats for specific shelf
shelfctl cache info --shelf programming

# Remove specific books from cache
shelfctl cache clear introduction-to-algorithms sicp

# Clear all cached books from a shelf
shelfctl cache clear --shelf programming

# Interactive picker (multi-select with spacebar)
shelfctl cache clear

# Clear entire cache (requires confirmation)
shelfctl cache clear --all
```

**In browse TUI:**
- Press `space` to select multiple books
- Press `x` to remove selected books from cache
- Books remain in your library, only local copies deleted
- Useful for reclaiming disk space

Books will automatically re-download when opened or browsed.

### Syncing annotations and highlights

When you annotate or highlight PDFs in your reader, those changes are saved to your local cache. Sync them back to GitHub to preserve your work:

```bash
# Sync specific book (CLI)
shelfctl sync sicp

# Sync all modified books (CLI)
shelfctl sync --all
```

**In browse TUI:**
- Press `s` on a modified book to sync it
- Or press `space` to select multiple modified books, then press `s` to sync all
- Status messages show progress during upload
- Books marked with modified indicator in cache info

Modified files are protected by default when clearing cache - you'll get a warning to sync first or use `--force` to delete anyway.

## Step 9: Add more shelves

Create shelves for different topics:

```bash
# History shelf
shelfctl init --repo shelf-history --name history --create-repo --create-release

# Fiction shelf
shelfctl init --repo shelf-fiction --name fiction --create-repo --create-release

# Research papers
shelfctl init --repo shelf-research --name research --create-repo --create-release
```

Verify:

```bash
shelfctl shelves
```

## Step 10: Add books from URLs

You can add books directly from URLs:

```bash
shelfctl shelve https://example.com/paper.pdf \
  --shelf research \
  --title "Important Research Paper" \
  --tags ai,ml \
  --year 2024
```

## Step 11: Organize with tags

Use tags to organize books within a shelf:

```bash
# Add a book about Lisp
shelfctl shelve ~/Downloads/sicp.pdf \
  --shelf programming \
  --title "Structure and Interpretation of Computer Programs" \
  --author "Abelson & Sussman" \
  --tags lisp,functional,textbook

# Add a book about algorithms
shelfctl shelve ~/Downloads/algo-design.pdf \
  --shelf programming \
  --title "Algorithm Design" \
  --author "Kleinberg & Tardos" \
  --tags algorithms,graphs,textbook
```

Filter by tag:

```bash
shelfctl browse --shelf programming --tag lisp
shelfctl browse --shelf programming --tag algorithms
```

## Common workflows

### Workflow 1: Adding multiple books at once

Create a shell script:

```bash
#!/bin/bash
for file in ~/Downloads/books/*.pdf; do
  shelfctl shelve "$file" \
    --shelf programming \
    --title "$(basename "$file" .pdf)" \
    --tags imported
done
```

Then review and update metadata later by editing `catalog.yml` directly.

### Workflow 2: Moving books between shelves

Realized a book belongs in a different shelf?

```bash
shelfctl move book-id --to-shelf different-shelf
```

### Workflow 3: Using releases as sub-categories

You can use releases within a shelf for sub-organization:

```bash
# Add to a specific year
shelfctl shelve book.pdf \
  --shelf research \
  --release 2024 \
  --title "Recent Paper"

# Add to archive
shelfctl shelve old-book.pdf \
  --shelf programming \
  --release archive \
  --title "Old Book"
```

### Workflow 4: Migrating from an existing collection

If you have books in another GitHub repo:

```bash
# Scan the old repo
shelfctl migrate scan --source your-username/old-books > queue.txt

# Edit queue.txt to add shelf mappings and metadata

# Migrate in batches
shelfctl migrate batch queue.txt --n 20 --continue
```

The `--continue` flag lets you resume if interrupted.

### Workflow 5: Sharing books between users

If a friend also uses shelfctl:

```bash
# Import their programming shelf into yours
shelfctl import \
  --from their-username/shelf-programming \
  --shelf imported-books
```

This copies books from their catalog to yours (requires their repos to be public or you to have access).

## Tips

### Tip 1: Use consistent IDs

The `--id` flag lets you control book IDs. Good practices:

- Use lowercase with hyphens: `effective-go`
- Keep them short but meaningful
- For academic papers, use citation keys: `smith2024-neural-nets`
- For ISBNs: `isbn-9780262033848`

### Tip 2: Enable shell completion

Set up autocompletion for faster command typing:

```bash
# Bash (add to ~/.bashrc)
source <(shelfctl completion bash)

# Zsh (add to ~/.zshrc)
source <(shelfctl completion zsh)

# Fish
shelfctl completion fish > ~/.config/fish/completions/shelfctl.fish
```

This enables tab completion for commands, flags, and shelf names.

### Tip 3: Automate with shell aliases

Add to your shell profile:

```bash
alias shelfadd='shelfctl shelve'
alias shelflist='shelfctl browse'
alias shelfopen='shelfctl open'
```

### Tip 4: Keep catalogs readable

Edit `catalog.yml` directly for bulk updates. It's just YAML:

```yaml
- id: book-id
  title: "The Book Title"
  author: "Author Name"
  year: 2024
  tags:
    - tag1
    - tag2
  format: pdf
  checksum:
    sha256: abc123...
  size_bytes: 1048576
  source:
    type: github_release
    owner: you
    repo: shelf-programming
    release: library
    asset: book-id.pdf
  meta:
    added_at: "2024-01-15T10:30:00Z"
```

After editing, commit and push manually or use shelfctl to add more books (it will merge correctly).

### Tip 4: Back up your config

Your `~/.config/shelfctl/config.yml` is small and important:

```bash
cp ~/.config/shelfctl/config.yml ~/Dropbox/shelfctl-config-backup.yml
```

### Tip 5: Use private repos for sensitive content

By default, repos can be private. Just ensure your GitHub token has access.

```bash
# On GitHub, make shelf-private-docs private via repo settings
```

shelfctl works identically with private repos.

## Next steps

- Read [COMMANDS.md](COMMANDS.md) for complete command reference
- Read [ARCHITECTURE.md](ARCHITECTURE.md) for schemas and configuration reference
- Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) if you hit issues
- Contribute improvements via [contributing.md](contributing.md)

## Example: Complete setup from scratch

```bash
# 1. Authenticate with GitHub (choose one)
# Option A: Using gh CLI (easier)
gh auth login
export GITHUB_TOKEN=$(gh auth token)

# Option B: Manual token
# export GITHUB_TOKEN=ghp_your_token_here

# 2. Install
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest

# 3. Configure
mkdir -p ~/.config/shelfctl
cat > ~/.config/shelfctl/config.yml <<EOF
github:
  owner: "myusername"
  token_env: "GITHUB_TOKEN"
defaults:
  release: "library"
  cache_dir: "~/.local/share/shelfctl/cache"
  asset_naming: "id"
shelves: []
EOF

# 4. Create shelf
shelfctl init --repo shelf-books --name books --create-repo --create-release

# 5. Add a book
shelfctl shelve ~/book.pdf --shelf books --title "My Book" --tags reading

# 6. Open it
shelfctl open my-book

# Done!
```
