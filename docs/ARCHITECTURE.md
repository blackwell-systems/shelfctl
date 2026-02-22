# Shelf Architecture Guide

This guide explains how shelfctl organizes your library and helps you make informed decisions about structure.

## Core Concepts

### What is a Shelf?

A shelf is a GitHub repository that stores your books:

- **Repository** - A GitHub repo (e.g., `shelf-programming`)
- **Catalog** - `catalog.yml` file tracked in Git with metadata
- **Assets** - PDF/EPUB files stored as GitHub Release assets (not in Git)

```
shelf-programming/
├── catalog.yml           # Metadata (in Git)
├── README.md            # Auto-generated inventory (in Git)
└── releases/
    └── library/         # Release tag
        ├── sicp.pdf     # Asset (not in Git)
        ├── taocp.pdf    # Asset (not in Git)
        └── ...
```

## Scope and Design Decisions

shelfctl is intentionally narrow: it manages PDF/EPUB libraries using GitHub as storage, and that's it.

### README.md - Human-Readable Inventory

Each shelf automatically gets a README.md that:

- **Documents the shelf** - Title, description, creation date
- **Shows quick stats** - Book count, last updated (auto-updated)
- **Provides usage examples** - How to add/browse books in this shelf
- **Tracks recently added** - Last 10 books added (auto-maintained)
- **Enables curation** - Sections for organizing by topic, reading lists, favorites
- **Links to resources** - Documentation and release assets

The README is created during `shelfctl init` and automatically updated when you add books. You can customize it freely - updates are non-intrusive and only modify the stats sections.

## Organization Philosophy

### Start Broad, Split Later

**Don't over-organize at the start!**

1. **Begin with one shelf** - `shelf-books` or `shelf-library`
2. **Use tags for organization** - Add tags like `programming`, `fiction`, `textbook`
3. **Split when needed** - When you hit 200-300 books, use `shelfctl split`

### When to Create Multiple Shelves

Create separate shelves when you have:

**Different Topics with Distinct Audiences**
```
shelf-work          # Professional books
shelf-personal      # Leisure reading
shelf-research      # Academic papers
```

**Large Collections (200-300+ books)**
```
shelf-programming   # Split from original shelf-books
shelf-history       # Split from original shelf-books
shelf-fiction       # Split from original shelf-books
```

**Different Access Requirements**
```
shelf-public        # Public GitHub repo
shelf-private       # Private GitHub repo
shelf-team          # Shared with organization
```

## Naming Conventions

### Repository Names

Use the `shelf-<topic>` pattern:

**Good:**
- `shelf-programming`
- `shelf-fiction`
- `shelf-research-papers`
- `shelf-textbooks`

**Avoid:**
- `books` (not descriptive)
- `my-library` (unclear)
- `programming-books` (breaks convention)

### Shelf Names (Config)

Shorter than repo names, used in commands:

| Repository Name         | Shelf Name (Config) |
|------------------------|---------------------|
| `shelf-programming`    | `programming`       |
| `shelf-fiction`        | `fiction`           |
| `shelf-research-papers`| `research`          |

## Sub-Organization with Releases

You can create multiple releases within one shelf for sub-categories:

```
shelf-programming/
  release: library        # Default, main collection
  release: textbooks      # Structured learning materials
  release: papers         # Academic papers
  release: references     # Quick references, cheat sheets
```

### When to Use Multiple Releases

- **Sub-topics within a shelf** - Keep related books together
- **Different formats** - Separate PDFs from EPUBs
- **Time-based** - Archive old editions separately
- **Status** - reading-list vs archive vs favorites

### Moving Between Releases

```bash
shelfctl move book-id --to-release textbooks
```

This is cheaper than creating new shelves - keeps everything in one repo.

## Splitting Shelves

When a shelf grows too large, use the interactive split wizard:

```bash
shelfctl split
```

### The Split Wizard

1. **Select source shelf** - Choose which shelf to split
2. **Choose strategy:**
   - By tag - Group books by their tags
   - By size - Split evenly by count
   - Manual - Assign one by one
3. **Preview groupings** - See what goes where
4. **Confirm and execute** - Automatic migration

### What Happens During Split

1. Books are downloaded from source shelf
2. Uploaded to target shelf releases
3. Catalogs are updated automatically
4. Old entries removed from source
5. No data loss - everything is moved, not copied

## Tags vs Shelves vs Releases

Choose the right organization level:

| Use Case | Solution | Example |
|----------|----------|---------|
| Similar books, slight variations | **Tags** | `--tags golang,web,tutorial` |
| Related sub-topics in one area | **Releases** | `release: textbooks` |
| Completely different domains | **Shelves** | `shelf-programming` vs `shelf-fiction` |

### Tags (Lightest)

- Add to any book: `--tags cs,algorithms,textbook`
- Filter: `shelfctl browse --tag algorithms`
- No overhead, infinite flexibility
- **Use first**, before considering releases or shelves

### Releases (Medium)

- Multiple categories within one shelf
- No new repo needed
- Slight overhead (separate asset uploads)
- Good for: sub-topics, formats, status

### Shelves (Heaviest)

- Separate GitHub repos
- Most overhead (separate repos to manage)
- Good for: major topics, different access levels

## Scaling Guidelines

### Small Library (0-100 books)

- **One shelf** - `shelf-books` or `shelf-library`
- **Use tags** - Organize everything with tags
- **One release** - Keep it simple with `library`

### Medium Library (100-300 books)

- **2-3 shelves** - Split by major topic if needed
- **Tags + releases** - Combine for organization
- **Consider splitting** - When approaching 300

### Large Library (300+ books)

- **Multiple shelves** - By topic or domain
- **Multiple releases** - Sub-organize within shelves
- **Split regularly** - Keep shelves under 300 books

## Common Patterns

### Academic Researcher

```
shelf-papers/
  release: reading-list
  release: cited
  release: archive
```

### Software Developer

```
shelf-programming/
  release: library (general)
  release: languages
  release: systems

shelf-references/
  release: library (cheat sheets, docs)
```

### General Reader

```
shelf-books/
  Use tags: fiction, non-fiction, biography, etc.
```

Split later into:
```
shelf-fiction/
shelf-non-fiction/
shelf-technical/
```

## Migration Strategy

### From Monolithic Repo

If you have an existing `books` repo with 500+ PDFs:

1. **Scan the old repo:**
   ```bash
   shelfctl migrate scan --source you/old-books > queue.txt
   ```

2. **Create organized shelves:**
   ```bash
   shelfctl init --repo shelf-programming --name programming --create-repo
   shelfctl init --repo shelf-history --name history --create-repo
   ```

3. **Edit queue.txt** - Assign each file to a shelf

4. **Migrate in batches:**
   ```bash
   shelfctl migrate batch queue.txt --n 20 --continue
   ```

### From Multiple Scattered Repos

Use `shelfctl import` to consolidate:

```bash
shelfctl import --from you/old-prog-books --shelf programming
shelfctl import --from you/random-pdfs --shelf books
```

## Best Practices

### [OK] Do

- Start with one broad shelf
- Use tags liberally
- Let organization emerge naturally
- Split when you feel friction (200-300 books)
- Use descriptive names (`shelf-programming`, not `prog`)
- Keep releases to 2-4 per shelf

### [X] Avoid

- Creating many shelves upfront
- Overly specific shelf names
- Premature optimization
- Deep nesting (no sub-releases within releases)
- Organizing before you have enough books

## Decision Tree

```
Do I need a new shelf?
│
├─ Do I have < 100 books total?
│  └─ No → Use one shelf + tags
│
├─ Is this a completely different topic?
│  └─ Yes → Create new shelf
│
├─ Do I have > 300 books in one shelf?
│  └─ Yes → Run `shelfctl split`
│
└─ Otherwise → Use tags or releases
```

## Advanced: GitHub Limits

### Release Asset Limits (Soft)

- GitHub doesn't publish hard limits
- Release assets avoid Git's 100MB file size limit
- In practice: 100s of assets per release work fine
- If you hit issues: split shelf or use multiple releases

### Repository Limits

- Git repo size: Keep under 5GB
- With shelfctl: Only `catalog.yml` is in Git (tiny)
- Assets don't count toward repo size
- In practice: Shelves can hold many GBs of books

### API Rate Limits

- Authenticated: 5000 requests/hour
- Most operations: 1-3 API calls
- Large migrations: Use `--n` flag to batch
- Rate limit resets hourly

## Getting Help

- Type `help` or `?` during interactive init
- Read `docs/TUTORIAL.md` for walkthrough
- Run `shelfctl split --help` for split options
- See `docs/COMMANDS.md` for all commands

## Summary

**Key Takeaway:** Start simple (one shelf + tags), organize as you go, split when needed.

shelfctl makes reorganization easy, so don't stress about getting it perfect upfront. Your first shelf name matters less than you think - you can always split and restructure later.

---

## Reference: Catalog Schema

The `catalog.yml` file in each shelf stores book metadata as a YAML list:

```yaml
- id: sicp
  title: "Structure and Interpretation of Computer Programs"
  author: "Abelson & Sussman"
  year: 1996
  tags: ["lisp", "cs", "textbook"]
  format: "pdf"

  checksum:
    sha256: "a1b2c3d4..."
  size_bytes: 6498234

  source:
    type: "github_release"
    owner: "your-username"
    repo: "shelf-programming"
    release: "library"
    asset: "sicp.pdf"

  meta:
    added_at: "2024-01-15T10:30:00Z"
```

### Required Fields

- `id` - Unique identifier (URL/CLI friendly: `^[a-z0-9][a-z0-9-]{1,62}$`)
- `title` - Book title
- `format` - File format (pdf, epub, mobi, etc.)
- `source.type` - Always `github_release`
- `source.owner` - GitHub username/org
- `source.repo` - Repository name
- `source.release` - Release tag name
- `source.asset` - Asset filename in release

### Recommended Fields

- `checksum.sha256` - File verification (strongly recommended)
- `author` - Book author(s)
- `tags` - List of tags for filtering
- `year` - Publication year
- `size_bytes` - File size in bytes

### Optional Fields

- `cover` - Path to cover image stored in git repo (e.g., `covers/book.jpg`)
- `meta.added_at` - Timestamp when added
- `meta.migrated_from` - Source if migrated

### Cover Images

shelfctl supports two types of cover images:

**1. Catalog Covers (user-curated):**
- Specified in catalog.yml `cover` field
- Stored in git repo (e.g., `covers/sicp.jpg`)
- Downloaded to cache when browsing: `.covers/<book-id>-catalog.jpg`
- Portable across machines via git
- Higher display priority

**2. Auto-Extracted Thumbnails (automatic):**
- Extracted from first page of PDF during download
- Stored in cache: `.covers/<book-id>.jpg`
- Local only (not in catalog or git)
- Requires `pdftoppm` from poppler-utils

Display priority: catalog cover > extracted thumbnail > none

---

## Reference: Configuration Schema

Config file location: `~/.config/shelfctl/config.yml`

```yaml
github:
  owner: "your-username"           # Your GitHub username/org
  token_env: "GITHUB_TOKEN"        # Environment variable name
  api_base: "https://api.github.com"  # For GitHub Enterprise
  backend: "api"                   # "api" or "gh" (shell out to gh CLI)

defaults:
  release: "library"               # Default release tag name
  cache_dir: "~/.local/share/shelfctl/cache"
  asset_naming: "id"               # "id" or "original"

shelves:
  - name: "programming"            # Shelf name for commands
    repo: "shelf-programming"      # GitHub repository name
    owner: "your-username"         # Override default owner (optional)
    catalog_path: "catalog.yml"    # Path to catalog (optional, default shown)
    default_release: "library"     # Override default release (optional)

  - name: "history"
    repo: "shelf-history"

migration:
  sources:
    - owner: "your-username"
      repo: "old-books-repo"       # Source repository
      ref: "main"                  # Git ref to read from
      mapping:                     # Path prefix → shelf mapping
        programming/: "programming"
        history/: "history"
```

### Environment Variables

You can override config with environment variables:

- `SHELFCTL_GITHUB_TOKEN` - GitHub token (recommended over token_env)
- `SHELFCTL_CACHE_DIR` - Cache directory
- `SHELFCTL_CONFIG` - Custom config file path

### Cache Structure

Downloaded books are stored locally:

```
~/.local/share/shelfctl/cache/
  shelf-programming/
    sicp.pdf
    gopl.pdf
  shelf-history/
    rome.pdf
```

Layout: `<cache>/<shelf_repo>/<asset_filename>`

### Cache Management

shelfctl provides commands to manage local cache without affecting shelf metadata or release assets:

**CLI Commands:**
```bash
# View statistics
shelfctl cache info                    # Overall stats
shelfctl cache info --shelf programming # Per-shelf stats

# Clear cache
shelfctl cache clear book-id-1 book-id-2  # Specific books
shelfctl cache clear --shelf programming  # Entire shelf
shelfctl cache clear --all                # Entire cache
shelfctl cache clear                      # Interactive picker
```

**TUI (in browse):**
- `space` - Toggle selection on books
- `x` - Remove selected books from cache
- Books remain in catalog/release, only local files deleted

**Use cases:**
- Reclaim disk space without affecting library
- Clear corrupted downloads for re-fetch
- Remove books you no longer need locally
- Pre-cache books for offline access, then clear when done

Books automatically re-download when opened or browsed.
