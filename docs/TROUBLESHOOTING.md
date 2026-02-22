# Troubleshooting

Common issues and solutions for shelfctl.

## Authentication issues

### "no GitHub token found"

**Symptom**: Error when running any command except `init`.

**Cause**: GitHub token not set in environment.

**Solution**:

**Option A: Using gh CLI (easier)**

```bash
# If you have gh CLI installed and authenticated
export GITHUB_TOKEN=$(gh auth token)

# Or authenticate now
gh auth login
export GITHUB_TOKEN=$(gh auth token)
```

**Option B: Manual token**

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

Make it permanent by adding to your shell profile:

```bash
# Bash
echo 'export GITHUB_TOKEN=$(gh auth token)' >> ~/.bashrc  # if using gh
# or
echo 'export GITHUB_TOKEN=ghp_...' >> ~/.bashrc          # if using manual token

# Zsh
echo 'export GITHUB_TOKEN=$(gh auth token)' >> ~/.zshrc  # if using gh
# or
echo 'export GITHUB_TOKEN=ghp_...' >> ~/.zshrc           # if using manual token
```

See the [Authentication section](https://github.com/blackwell-systems/shelfctl#authentication) for more details.

### "GitHub API error 401: Unauthorized"

**Cause**: Token is invalid or expired.

**Solution**:

1. Go to https://github.com/settings/tokens
2. Check if your token still exists
3. If expired, generate a new one with `repo` scope
4. Update your `GITHUB_TOKEN` environment variable

### "GitHub API error 403: Forbidden"

**Cause**: Token doesn't have the required `repo` scope.

**Solution**:

1. Go to https://github.com/settings/tokens
2. Click on your shelfctl token
3. Check "repo" (Full control of private repositories)
4. Update token and save

## Repository issues

### "repository not found"

**Cause**: Repository doesn't exist or token doesn't have access.

**Solution**:

If the repo should exist:
- Check repo name spelling in config
- Verify repo owner is correct
- Ensure your token has access (make repo public or grant collaborator access)

If you need to create it:

```bash
shelfctl init --repo shelf-name --name name --create-repo --create-release
```

### "catalog.yml not found"

**Cause**: Shelf repo exists but has no catalog file.

**Solution**:

```bash
# Re-run init (safe, won't duplicate config entry)
shelfctl init --repo shelf-name --name name
```

This creates an empty catalog if one doesn't exist.

## Upload issues

### "asset already exists"

**Symptom**: Error when adding a book with `--asset-name` that already exists.

**Cause**: GitHub release assets must have unique names within a release.

**Solutions**:

1. Use different asset name:
   ```bash
   shelfctl shelve book.pdf --shelf prog --asset-name book-v2.pdf --title "..."
   ```

2. Use default `--asset-naming=id` (prevents collisions):
   ```bash
   shelfctl shelve book.pdf --shelf prog --id unique-id --title "..."
   ```

3. Delete the old asset from GitHub releases UI first

### Upload times out

**Symptom**: Large file upload fails with timeout.

**Cause**: Default 5-minute timeout may not be enough for very large files or slow connections.

**Solution**:

Files over ~500MB may hit timeouts. Consider:
- Splitting large files
- Using a faster connection
- Compressing PDFs (many PDFs can be optimized)

### Checksum mismatch

**Symptom**: Download succeeds but checksum verification fails.

**Cause**: File was corrupted during upload or download, or asset was replaced on GitHub.

**Solution**:

```bash
# Clear cache and re-download
shelfctl cache clear book-id
shelfctl open book-id
```

If problem persists, the asset on GitHub may be corrupt. Re-upload:

```bash
# Delete from GitHub releases UI, then:
shelfctl shelve original-file.pdf --shelf name --id book-id --title "..."
```

## Configuration issues

### "shelf not found in config"

**Symptom**: `shelfctl shelve --shelf name` fails with "shelf not found".

**Cause**: Shelf not defined in `~/.config/shelfctl/config.yml`.

**Solution**:

```bash
# Add shelf to config
shelfctl init --repo shelf-name --name name
```

Or edit `~/.config/shelfctl/config.yml` manually:

```yaml
shelves:
  - name: "name"
    repo: "shelf-name"
```

### Config file not found

**Symptom**: Commands fail with "config file not found".

**Cause**: No config at `~/.config/shelfctl/config.yml`.

**Solution**:

```bash
mkdir -p ~/.config/shelfctl
cp config.example.yml ~/.config/shelfctl/config.yml
# Edit with your settings
```

### YAML parse error

**Symptom**: "parsing config YAML: ..." error.

**Cause**: Syntax error in config file.

**Solution**:

Check YAML syntax:
- Consistent indentation (2 spaces)
- Quoted strings with special characters
- No tabs (use spaces only)

Validate with:

```bash
# Install yq if needed: brew install yq
yq eval ~/.config/shelfctl/config.yml
```

## Cache issues

### Cache taking too much space

**Symptom**: `~/.local/share/shelfctl/cache` is large.

**Solution**:

```bash
# Check cache statistics
shelfctl cache info

# Check specific shelf
shelfctl cache info --shelf programming

# Remove specific books from cache
shelfctl cache clear book-id-1 book-id-2

# Clear all books from a shelf
shelfctl cache clear --shelf programming

# Clear entire cache (requires confirmation)
shelfctl cache clear --all

# Files will be re-downloaded when opened or browsed
```

**In browse TUI:**
- Press `space` to select books
- Press `x` to remove selected books from cache
- Books remain in catalog, only local files deleted

### Can't find cached file

**Symptom**: `shelfctl open book-id` claims it's cached but file is missing.

**Cause**: Cache file was deleted manually.

**Solution**:

```bash
# Re-download
shelfctl open book-id
```

## Migration issues

### "file not found in source repo"

**Symptom**: Migration fails for a file in the queue.

**Cause**: File doesn't exist at that path in the source repo/ref.

**Solution**:

- Check the path is correct
- Verify the ref (branch/tag) exists
- File may have been moved or deleted

Use `--continue` flag to skip failed entries:

```bash
shelfctl migrate batch queue.txt --continue
```

### Migration ledger corruption

**Symptom**: Ledger file has bad format or mixed state.

**Cause**: Manual editing or interrupted operations.

**Solution**:

```bash
# Back up ledger
cp .shelfctl-ledger.txt .shelfctl-ledger.txt.bak

# Remove corrupt lines or start fresh
rm .shelfctl-ledger.txt

# Re-run with --continue (will skip already-migrated books)
shelfctl migrate batch queue.txt --continue
```

## Performance issues

### Commands are slow

**Cause**: GitHub API rate limits or network latency.

**Solution**:

- Check GitHub API rate limit: `curl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/rate_limit`
- Wait if rate limited (resets hourly)
- Use `--n` flag for batch operations to limit size

### List command is slow with many books

**Cause**: Loading and parsing many catalogs.

**Solution**:

- Use specific shelf: `shelfctl browse --shelf programming`
- Filter by tag: `shelfctl browse --tag specific-tag`
- Consider splitting large shelves with `shelfctl split`

## GitHub-specific issues

### Rate limit exceeded

**Symptom**: "API rate limit exceeded" error.

**Cause**: Made too many GitHub API requests (5000/hour for authenticated requests).

**Solution**:

Wait for the rate limit to reset (shown in error message), or:

- Use `--n` flag to process in smaller batches
- Space out operations
- Authenticated requests have higher limits than unauthenticated

### GitHub release limit

**Symptom**: Can't upload more assets to a release.

**Cause**: GitHub has soft limits on release asset size and count.

**Solution**:

Use multiple releases:

```bash
# Add books to different releases
shelfctl shelve book1.pdf --shelf prog --release 2024
shelfctl shelve book2.pdf --shelf prog --release 2025
```

Or split the shelf:

```bash
shelfctl split  # Interactive wizard
```

## ID and naming issues

### "invalid ID — must match ^[a-z0-9][a-z0-9-]{1,62}$"

**Cause**: Book ID contains invalid characters.

**Solution**:

IDs must:
- Start with lowercase letter or digit
- Contain only lowercase letters, digits, and hyphens
- Be 2-63 characters long

Good IDs:
- `sicp`
- `algorithm-design`
- `isbn-9780262033848`
- `smith2024-paper`

Bad IDs:
- `SICP` (uppercase)
- `algorithm_design` (underscore)
- `s` (too short)

### Duplicate book ID

**Symptom**: Adding a book with an existing ID.

**Cause**: A book with that ID already exists in the catalog.

**Solution**:

The new entry will replace the old one. If you want both:

```bash
# Use different ID
shelfctl shelve book.pdf --shelf prog --id sicp-v2 --title "SICP 2nd Ed"

# Or use SHA-based ID
shelfctl shelve book.pdf --shelf prog --id-sha12 --title "..."
```

## Platform-specific issues

### "open" command doesn't work (Linux)

**Cause**: `xdg-open` not installed or not in PATH.

**Solution**:

```bash
# Install xdg-utils
sudo apt-get install xdg-utils  # Debian/Ubuntu
sudo yum install xdg-utils      # RHEL/CentOS
```

Or use `open` and open manually:

```bash
shelfctl open book-id
# Opens in default PDF viewer
```

### Permission denied (macOS)

**Symptom**: "operation not permitted" when opening files.

**Cause**: macOS security settings.

**Solution**:

- System Preferences → Security & Privacy → Files and Folders
- Grant terminal/shelfctl permission to access files

## Cover thumbnails not appearing

**Symptom**: No 📷 camera emoji showing in browser, or covers not displaying in details pane.

**Cause**: `pdftoppm` command not installed (required for PDF cover extraction).

**Solution**:

Install poppler-utils which provides pdftoppm:

**macOS:**
```bash
brew install poppler
```

**Ubuntu/Debian:**
```bash
sudo apt-get install poppler-utils
```

**Fedora/RHEL:**
```bash
sudo dnf install poppler-utils
```

**Arch Linux:**
```bash
sudo pacman -S poppler
```

After installing, covers will be extracted automatically next time you download or open a PDF.

**Verify covers are being extracted:**

```bash
# Check if pdftoppm is installed
which pdftoppm

# If not installed, install poppler (see above)

# Download a PDF to trigger extraction
shelfctl open <book-id>

# Check if cover was created
ls ~/.local/share/shelfctl/cache/<repo>/.covers/
```

**Notes:**
- Cover extraction is optional - books work fine without it
- Only PDFs are supported (EPUB covers not yet implemented)
- Covers are stored in `~/.local/share/shelfctl/cache/<repo>/.covers/`
- Inline images work in Kitty, Ghostty, or iTerm2 terminals (other terminals show 📷 emoji)
- **tmux limitation**: Image protocols don't work through tmux multiplexers - run shelfctl directly in Ghostty/Kitty/iTerm2 for inline image display
- Even without inline images, the 📷 emoji indicates a cover exists

---

## HTML index: clicking books doesn't open files

**Symptom**: Clicking books in the HTML index (`shelfctl index`) doesn't open them.

**Cause**: Browser security restrictions on `file://` protocol links. Modern browsers limit file:// navigation to prevent malicious pages from accessing your filesystem.

**Browser Compatibility**:

- ✅ **Safari** (macOS): Works - allows file:// navigation from file:// pages
- ⚠️ **Firefox**: May work by default, or may require security setting change
  - Navigate to `about:config`
  - Search for `security.fileuri.strict_origin_policy`
  - Set to `false` (warning: reduces security slightly)
  - Restart Firefox
- ❌ **Chrome/Edge**: Blocked - these browsers prevent file:// links for security

**Solutions**:

1. **Use Safari** (recommended on macOS) - no configuration needed

2. **Copy and open manually**:
   - Right-click any book card
   - Select "Copy Link Address"
   - Open in terminal:
   ```bash
   # macOS
   open <paste-file-path>

   # Linux
   xdg-open <paste-file-path>
   ```

3. **Use TUI browser instead** (no restrictions):
   ```bash
   shelfctl browse
   # Press 'o' on any book to open it
   ```

The TUI browser always works because it uses direct file system operations rather than browser security policies.

---

## Still stuck?

1. Check if issue already exists: https://github.com/blackwell-systems/shelfctl/issues
2. Open a new issue with:
   - shelfctl version: `shelfctl --version` (or commit hash)
   - Operating system
   - Full error message
   - Steps to reproduce
3. For sensitive issues (tokens, private repos), email maintainers directly

## Debug mode

For more verbose output, set environment variable:

```bash
export SHELFCTL_DEBUG=1
shelfctl <command>
```

This shows API requests and responses (tokens are redacted).
