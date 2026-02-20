# Tutorial

A step-by-step guide to setting up and using shelfctl.

## Prerequisites

- Go 1.21+ installed
- GitHub account
- GitHub personal access token with `repo` scope

## Step 1: Create a GitHub token

1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Give it a descriptive name: "shelfctl"
4. Select scope: **repo** (Full control of private repositories)
5. Click "Generate token"
6. Copy the token (starts with `ghp_`)

Save it securely. You won't be able to see it again.

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
  token_env: "GITHUB_TOKEN"

defaults:
  release: "library"
  cache_dir: "~/.local/share/shelfctl/cache"
  asset_naming: "id"

shelves: []  # Will populate via init command
```

## Step 4: Set your GitHub token

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

Add to your shell profile to persist:

```bash
# For bash
echo 'export GITHUB_TOKEN=ghp_your_token_here' >> ~/.bashrc

# For zsh
echo 'export GITHUB_TOKEN=ghp_your_token_here' >> ~/.zshrc
```

## Step 5: Create your first shelf

Let's create a shelf for programming books:

```bash
shelfctl init \
  --repo shelf-programming \
  --name programming \
  --create-repo \
  --create-release
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

## Step 6: Add your first book

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

## Step 7: Browse your books

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

## Step 8: View book details

```bash
shelfctl info introduction-to-algorithms
```

This shows full metadata, source location, cache status, and checksum.

## Step 9: Open a book

```bash
shelfctl open introduction-to-algorithms
```

This will:
1. Download the book to your cache (if not already cached)
2. Verify the checksum
3. Open it with your system's default PDF viewer

The next time you open it, it will use the cached copy (no download).

## Step 10: Add more shelves

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

## Step 11: Add books from URLs

You can add books directly from URLs:

```bash
shelfctl shelve https://example.com/paper.pdf \
  --shelf research \
  --title "Important Research Paper" \
  --tags ai,ml \
  --year 2024
```

## Step 12: Organize with tags

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
  --from-shelf programming \
  --from-owner their-username \
  --to-shelf imported-books
```

This copies books from their catalog to yours (requires their repos to be public or you to have access).

## Tips

### Tip 1: Use consistent IDs

The `--id` flag lets you control book IDs. Good practices:

- Use lowercase with hyphens: `effective-go`
- Keep them short but meaningful
- For academic papers, use citation keys: `smith2024-neural-nets`
- For ISBNs: `isbn-9780262033848`

### Tip 2: Automate with shell aliases

Add to your shell profile:

```bash
alias shelfadd='shelfctl shelve'
alias shelflist='shelfctl browse'
alias shelfopen='shelfctl open'
```

### Tip 3: Keep catalogs readable

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
- Read [SPEC.md](SPEC.md) for architecture details
- Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) if you hit issues
- Contribute improvements via [CONTRIBUTING.md](CONTRIBUTING.md)

## Example: Complete setup from scratch

```bash
# 1. Install
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest

# 2. Configure
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

# 3. Set token
export GITHUB_TOKEN=ghp_...

# 4. Create shelf
shelfctl init --repo shelf-books --name books --create-repo --create-release

# 5. Add a book
shelfctl shelve ~/book.pdf --shelf books --title "My Book" --tags reading

# 6. Open it
shelfctl open my-book

# Done!
```
