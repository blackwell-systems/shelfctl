Makes sense: **repos are shelves → name them like shelves**, and **`shelfctl` is a Go CLI**.

Below is an updated, Go-first spec with **Cobra + Viper** and optional **fatih/color** for output.

---

# shelfctl Project Spec (Go, GitHub Releases, “Shelf repos”)

## One-liner

`shelfctl` is a Go CLI that manages a personal document library where:

* **Shelf repos** (GitHub repos) store **metadata** (`catalog.yml`, covers, notes)
* **GitHub Release assets** store the **actual files** (PDF/EPUB/etc.)
* Users can **get/open one book at a time**, **ingest locally**, and **migrate incrementally** from an old monorepo — with **zero infrastructure**

This tool does **not ship content**; it manages *your* files under *your* GitHub auth.

---

## Naming conventions

### Shelf repos

Use `shelf-<topic>` as the repo pattern:

* `shelf-programming`
* `shelf-philosophy`
* `shelf-history`

(If one shelf gets too big later: `shelf-history-ancient`, `shelf-history-modern`, etc.)

### Releases as sub-shelves

Each shelf repo has one or more release tags. The default is `library`, but you can use multiple releases within the same repo as **logical sub-shelves** — no new repo required:

```
shelf-programming/
  release: library         ← default, start here
  release: languages       ← Python, Rust, Go, JS books
  release: systems         ← OS, networking, compilers
  release: theory          ← algorithms, math, CS fundamentals
```

**Two kinds of splits:**

| Situation | Solution |
|-----------|----------|
| Too many books, want sub-topics | Add a new release to the same repo (`shelfctl move --to-release <tag>`) |
| Repo approaching 5 GB | Create a new repo (`shelfctl move --to-shelf <name>`) |

Use **releases** for organization. Use **new repos** only when forced by size.

---

## Storage model

### In Git (small files)

* `catalog.yml` (source of truth)
* `covers/` (optional thumbnails)
* `notes/` (optional)
* `README.md` (optional curated lists)

### Not in Git (large binaries)

* PDFs/EPUBs stored as **Release assets** under release `library`

### Local cache

Default cache:

* macOS/Linux: `~/.local/share/shelfctl/cache`

Cache path:

```
<cache>/<shelf_repo>/<asset_filename>
```

---

## Catalog schema (v1)

`catalog.yml` = YAML list.

```yaml
- id: ostep
  title: "Operating Systems: Three Easy Pieces"
  author: "Remzi & Andrea Arpaci-Dusseau"
  year: 2018
  tags: ["os", "systems"]
  format: "pdf"
  cover: "covers/ostep.jpg"   # optional

  checksum:
    sha256: "..."             # strongly recommended
  size_bytes: 31457280        # optional

  source:
    type: "github_release"
    owner: "YOUR_GH_OWNER"
    repo: "shelf-programming"
    release: "systems"       # release tag; defaults to "library"
    asset: "ostep.pdf"

  meta:
    added_at: "2026-02-20T00:00:00Z"         # optional
    migrated_from: "oldbooks:programming/..." # optional
```

### Required

* `id`, `title`, `format`
* `source.{owner,repo,release,asset}` (with `type=github_release`)

### Recommended

* `checksum.sha256`
* `tags`

### ID rules

* `^[a-z0-9][a-z0-9-]{1,62}$` (URL/CLI friendly)
* stable once created
* optionally support deterministic IDs: `--id sha12` (sha256 prefix)

---

## Config (Viper)

### Config file

Default path:

* `~/.config/shelfctl/config.yml`

Example:

```yaml
github:
  owner: "dayna"
  token_env: "GITHUB_TOKEN"   # do not store token in file
  api_base: "https://api.github.com"  # overridable for GH Enterprise later

defaults:
  release: "library"
  cache_dir: "~/.local/share/shelfctl/cache"
  asset_naming: "id"          # "id" or "original"

shelves:
  - name: "programming"
    owner: "dayna"
    repo: "shelf-programming"
    catalog_path: "catalog.yml"
    default_release: "library"   # overrides defaults.release for this shelf
  - name: "history"
    owner: "dayna"
    repo: "shelf-history"
    catalog_path: "catalog.yml"
  - name: "philosophy"
    owner: "dayna"
    repo: "shelf-philosophy"
    catalog_path: "catalog.yml"

migration:
  sources:
    - owner: "dayna"
      repo: "books"           # old monorepo
      ref: "main"
      mapping:
        programming/: "programming"
        history/: "history"
        philosophy/: "philosophy"
    - owner: "dayna"
      repo: "papers"          # second source repo (optional)
      ref: "master"
      mapping:
        cs/: "programming"
        misc/: "reading"
```

### Env overrides

* `SHELFCTL_GITHUB_TOKEN` (or read env named by `token_env`)
* `SHELFCTL_CACHE_DIR`
* `SHELFCTL_CONFIG`

---

## CLI UX and commands (Cobra)

### `shelfctl init`

Creates config skeleton and optionally bootstraps a shelf repo + release.

Flags:

* `--owner`, `--repo shelf-programming`, `--name programming`
* `--create-repo` (GitHub API)
* `--private` (default true)
* `--create-release` (default true)

### `shelfctl shelves`

Validate shelves:

* repo exists
* `catalog.yml` exists (or offer `--fix` to create)
* `library` release exists (or offer `--fix` to create)

### `shelfctl browse`

Browse books in interactive TUI (when in terminal) or list as text (when piped/scripted).

**Interactive mode** (default in terminal):
- Visual browser with keyboard navigation
- Live filtering with `/`
- Color-coded cache indicators

**Text mode** (when piped or with `--no-interactive`):
- Plain text output for scripts and automation

Flags:

* `--shelf <name>`
* `--tag <tag>` (repeatable)
* `--search <text>` (title/author/tags)
* `--format <pdf|epub|...>`
* `--no-interactive` (disable TUI)

### `shelfctl info <id>`

Show metadata and local cache status.

### `shelfctl open <id>`

Download **exactly one** asset to cache.

Flags:

* `--shelf <name>` if collisions
* `--force` redownload
* `--to <path>` copy to destination

Behavior:

* download release asset by name
* verify sha256 if present
* write to cache

### `shelfctl open <id>`

Ensure cached, then open:

* macOS: `open`
* Linux: `xdg-open`

Flags:

* `--app <app>`

### `shelfctl shelve [file-or-url]`

Ingest a local file **or a URL** into release-assets + catalog. This is the primary day-to-day command for adding new books.

**Interactive mode** (when no arguments provided):
1. Shelf picker (if multiple shelves configured)
2. File browser (starts in ~/Downloads, filters for documents)
3. Metadata form (title, author, tags, ID)
4. Automatic upload and catalog update

**Input types:**

| Argument | Behaviour |
|----------|-----------|
| `/path/to/book.pdf` | read from disk |
| `https://example.com/book.pdf` | stream download → sha256 in-flight → pipe to upload (no temp file) |
| `github:owner/repo@ref:path/to/file.pdf` | download via GitHub Contents API (authenticated) |

Steps:

1. resolve input (file, HTTP URL, or GitHub path) as a readable stream
2. compute sha256 + size in-flight while buffering for upload
3. choose `id` (from `--id` or prompt; optionally `sha12`)
4. determine asset filename:

   * if `asset_naming=id`: `<id>.<ext>`
   * else keep normalized original name
5. ensure target release exists (default `library`, or `--release <tag>`)
6. upload asset
7. update `catalog.yml` (append + sort optional)
8. commit + push catalog changes

Flags:

* `--shelf <name>` (interactive picker if omitted)
* `--id`, `--title`, `--author`, `--year`
* `--tags a,b,c`
* `--release <tag>` target sub-shelf release (default: shelf's `default_release`)
* `--asset-name <filename>`
* `--no-push` (local edit only)
* `--no-interactive` (disable TUI)
* `--force` (skip duplicate checks and overwrite existing assets)

### `shelfctl move <id> --to-release <tag>`

Move a book to a different release within the same repo (logical sub-shelf split).

Steps:

1. download asset from old release (uses cache if available)
2. upload to new release (create release if it doesn't exist)
3. update `source.release` in `catalog.yml`
4. delete old asset from old release
5. commit + push catalog

Flags:

* `--to-release <tag>` move to different release, same repo
* `--to-shelf <name>` move to different shelf repo entirely (new repo must exist in config)
* `--dry-run` show what would happen without doing it
* `--keep-old` skip deleting the old asset (useful if unsure)

> **Note:** Moving between repos requires re-uploading the asset (GitHub has no cross-repo move API). The CLI streams directly from old release URL to new upload — no intermediate disk write unless the file is already cached.

### `shelfctl split --shelf <name>`

Interactive wizard to split a shelf. Groups books by tag, proposes release or repo assignments, then calls `move` in batch.

Flags:

* `--by-tag` group books by tag and propose sub-releases
* `--dry-run`
* `--n <max>` limit books processed per run (resume-safe)

---

### `shelfctl migrate one <old_path>`

Incremental migration without cloning old repo.

Steps:

1. download file from old repo via GitHub Contents API (raw)
2. route to shelf via `migration.mapping` (prefix match)
3. call `shelve` path from temp
4. append record to a local ledger

Ledger:

* `~/.local/share/shelfctl/migrated.jsonl`

### `shelfctl migrate batch <queue_file>`

Migrate N lines from a queue file.

Flags:

* `--n 5`
* `--continue` (skip already migrated)
* `--dry-run`

### `shelfctl migrate scan [--source <owner/repo>]`

List all files in a configured migration source repo and emit a queue file suitable for `migrate batch`. Lets you see what's there before committing to migrate it.

Flags:

* `--source <owner/repo>` override config (useful for one-off sources)
* `--ext pdf,epub,mobi` filter by extension
* `--out <file>` write queue to file (default: stdout)

### `shelfctl import --from <owner/repo>`

Import from an existing `shelfctl`-structured repo (reads `catalog.yml`, downloads each asset, re-uploads to your shelf). Use this to absorb another user's shelf or a second account's shelf.

Steps:

1. fetch `catalog.yml` from source repo
2. for each entry: stream asset from source release → sha256 in-flight → upload to your target shelf release
3. merge entries into your `catalog.yml` (skip duplicates by sha256)
4. commit + push

Flags:

* `--shelf <name>` target shelf in your config
* `--release <tag>` target release (default: shelf's `default_release`)
* `--dry-run`
* `--n <max>` limit per run (resume-safe via ledger)

---

## GitHub integration (Go)

### Auth

* Token from env var (Viper-configurable key)
* Never print token
* Token scope: `repo` for private repos

### API operations needed

* Get/Create release by tag (`library`)
* List assets for release
* Download asset (authenticated, streaming)
* Upload asset (multipart upload endpoint)
* Read/update `catalog.yml`

  * MVP: easiest is **local clone** of shelf repos (small) and `git commit/push`
  * Later: can support GitHub Contents API updates to avoid cloning

### Backends

Support two interchangeable backends:

* `github-api` (native HTTP client in Go) **default**
* `gh-cli` (shell out to `gh`) optional fallback for weird auth setups

Config:

```yaml
github:
  backend: "api"  # or "gh"
```

---

## Output / colors

### Minimal dependency (recommended)

Use `fatih/color` for colored status lines:

* green: success
* yellow: warning
* red: error
* cyan: headings

Optional:

* `--no-color` / auto-disable if not TTY

If you meant something else by “faith”, I’m assuming you meant **`github.com/fatih/color`**.

---

## Repo layout (Go)

```
shelfctl/
  README.md
  LICENSE
  go.mod
  go.sum

  cmd/
    shelfctl/
      main.go

  internal/
    app/
      root.go           # cobra root
      init.go
      shelves.go
      browse.go         # (formerly list.go)
      info.go
      open.go           # (inlines former get.go)
      shelve.go         # (formerly add.go)
      move.go
      split.go
      migrate.go        # migrate one / batch / scan
      import.go

    tui/
      detection.go      # TUI mode detection
      common.go         # shared styles
      list_browser.go   # interactive book browser
      shelve_form.go    # metadata entry form
      file_picker.go    # filesystem browser
      shelf_picker.go   # shelf selector

    config/
      config.go         # viper load/validate
      schema.go

    catalog/
      model.go          # structs
      load.go           # YAML load
      save.go           # YAML write
      search.go         # filter/search helpers

    github/
      client.go         # http client + auth
      releases.go
      assets.go
      contents.go       # read/write catalog via API (future)
      errors.go

    cache/
      paths.go
      verify.go         # sha256 verify
      store.go

    migrate/
      ledger.go         # jsonl ledger
      mapping.go
      scan.go           # list files in source repo

    ingest/
      resolver.go       # detect file / http url / github:// input type
      stream.go         # unified streaming reader with in-flight sha256

    util/
      hash.go
      io.go
      term.go           # tty detection, color enable/disable
```

---

## Workflows

### Drain a monorepo into shelves (one-time)

This is the primary migration path: you have an existing GitHub repo full of books in flat or folder structure, and you want to restructure it into shelves.

```bash
# 1. Create shelves for each topic
shelfctl init --repo shelf-programming --name programming --create-repo --create-release
shelfctl init --repo shelf-history --name history --create-repo --create-release

# 2. Configure migration sources in ~/.config/shelfctl/config.yml
#    (set migration.sources with owner/repo/ref/mapping)

# 3. Scan what's in the old repo, write a queue file
shelfctl migrate scan --source dayna/books --ext pdf,epub > queue.txt

# 4. Review queue.txt, edit if needed, then drain in batches
shelfctl migrate batch queue.txt --n 10 --continue

# 5. Verify
shelfctl shelves
shelfctl browse --shelf programming

# 6. Repeat until queue is empty, then archive old repo
```

The ledger at `~/.local/share/shelfctl/migrated.jsonl` tracks every completed migration so `--continue` is safe to run repeatedly.

### Ongoing ingestion (day-to-day)

After migration, the same `shelve` command handles all new books regardless of source:

```bash
# From disk
shelfctl shelve ~/Downloads/newbook.pdf --shelf programming --title "New Book" --tags go

# Stream from a URL (no temp file)
shelfctl shelve https://example.com/book.pdf --shelf history --title "..." --tags ancient

# From another GitHub repo (authenticated)
shelfctl shelve github:someuser/repo@main:books/title.pdf --shelf philosophy

# Batch import from another shelf repo
shelfctl import --from someuser/shelf-philosophy --shelf philosophy
```

---

## MVP acceptance criteria

MVP is done when, for private repos:

1. `shelfctl shelves` validates all shelves and creates missing `library` release with `--fix`
2. `shelfctl shelve <file> --shelf history --title ... --tags ...` uploads to release + updates `catalog.yml` and pushes
3. `shelfctl open <id>` downloads exactly one asset, verifies sha256, caches it
4. `shelfctl open <id>` opens cached file (or downloads then opens)
5. `shelfctl migrate one <old_path>` pulls one file from old monorepo via API and ingests to correct shelf

---

## Enhancements (post-MVP)

### `shelfctl serve` — local web frontend

**Non-blocking:** the CLI core builds and ships independently. This is purely additive. All business logic stays in `internal/` packages; the HTTP layer just calls the same functions.

#### Rationale

The GitHub token must never reach a browser. A local server is the cleanest solution: the Go process holds the token, exposes a localhost API, and serves a SPA from the same binary. No extra infrastructure, no OAuth dance.

#### Command

```
shelfctl serve [--port 8080] [--host 127.0.0.1]
```

Starts a local HTTP server. Browser opens automatically (or user navigates to `http://localhost:8080`).

Config key:

```yaml
serve:
  port: 8080
  host: "127.0.0.1"   # never bind to 0.0.0.0 by default
```

#### API surface (REST, JSON)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/shelves` | list configured shelves with book count |
| GET | `/api/books` | list books, same filter params as `shelfctl browse` |
| GET | `/api/books/:id` | single book metadata + cache status |
| POST | `/api/books` | ingest (wraps `shelve` logic) |
| GET | `/api/books/:id/open` | ensure cached + open with system viewer |
| GET | `/api/books/:id/download` | stream asset to browser |
| GET | `/api/migrate/status` | ledger summary |

All handlers are thin wrappers over existing `internal/` packages — no business logic lives in `internal/api/`.

#### SPA

A minimal browser UI (shelf browser, book grid/list with covers, search, book detail, add form) embedded in the binary via `//go:embed`:

```
cmd/shelfctl/ui/dist/   ← compiled frontend assets (gitignored build output)
```

Source lives in `ui/` at repo root (e.g. Vite + vanilla TS or a small React app). The Go binary embeds the compiled `dist/` so `go install` produces a single self-contained binary.

#### Repo layout additions

```
shelfctl/
  ui/                   ← frontend source (Vite/TS, not compiled into git)
    src/
    index.html
    vite.config.ts
    package.json

  cmd/
    shelfctl/
      ui/
        dist/           ← built assets embedded by go:embed (gitignored)

  internal/
    api/
      server.go         ← http.Server setup, embed.FS mount
      handlers.go       ← REST handlers (call internal/* packages)
      middleware.go     ← logging, CORS (localhost only)

    app/
      serve.go          ← cobra command: starts api.Server
```

#### Build notes

* `make build` (or `go build`) skips the UI — CLI works without it
* `make build-ui` runs `npm run build` in `ui/`, then `go build` embeds the result
* CI produces two artifacts: `shelfctl` (CLI-only) and `shelfctl-ui` (with embedded frontend) — or a single binary with the UI always embedded, depending on preference

---

## Open-source safety wording (README requirement)

* “Manages user-provided files in user-owned GitHub repos/releases.”
* “Does not distribute content.”
* “Designed for personal document libraries (PDF/EPUB/etc.).”

---



