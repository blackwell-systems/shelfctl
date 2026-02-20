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

### Rolling release per shelf repo

Each shelf repo has a single rolling release tag:

* `library` (default tag/name)

Assets live there.

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
<cache>/<shelf_repo>/<book_id>/<asset_filename>
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
    release: "library"
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
  - name: "history"
    owner: "dayna"
    repo: "shelf-history"
    catalog_path: "catalog.yml"
  - name: "philosophy"
    owner: "dayna"
    repo: "shelf-philosophy"
    catalog_path: "catalog.yml"

migration:
  source_owner: "dayna"
  source_repo: "books"        # your old monorepo
  source_ref: "main"
  mapping:
    programming/: "programming"
    history/: "history"
    philosophy/: "philosophy"
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

### `shelfctl list`

List books, filterable.

Flags:

* `--shelf <name>`
* `--tag <tag>` (repeatable)
* `--search <text>` (title/author/tags)
* `--format <pdf|epub|...>`

### `shelfctl info <id>`

Show metadata and local cache status.

### `shelfctl get <id>`

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

### `shelfctl add <file> --shelf <name>`

Ingest a local file into release-assets + catalog.

Steps:

1. compute sha256 + size
2. choose `id` (from `--id` or prompt; optionally `sha12`)
3. determine asset filename:

   * if `asset_naming=id`: `<id>.<ext>`
   * else keep normalized original name
4. ensure `library` release exists
5. upload asset
6. update `catalog.yml` (append + sort optional)
7. commit + push catalog changes (using local clone of shelf repo OR GitHub Contents API)

Flags:

* `--id`, `--title`, `--author`, `--year`
* `--tags a,b,c`
* `--asset-name <filename>`
* `--no-push` (local edit only)

### `shelfctl migrate one <old_path>`

Incremental migration without cloning old repo.

Steps:

1. download file from old repo via GitHub Contents API (raw)
2. route to shelf via `migration.mapping` (prefix match)
3. call `add` path from temp
4. append record to a local ledger

Ledger:

* `~/.local/share/shelfctl/migrated.jsonl`

### `shelfctl migrate batch <queue_file>`

Migrate N lines from a queue file.

Flags:

* `--n 5`
* `--continue` (skip already migrated)
* `--dry-run`

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
      list.go
      info.go
      get.go
      open.go
      add.go
      migrate.go

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

    util/
      hash.go
      io.go
      term.go           # tty detection, color enable/disable
```

---

## MVP acceptance criteria

MVP is done when, for private repos:

1. `shelfctl shelves` validates all shelves and creates missing `library` release with `--fix`
2. `shelfctl add <file> --shelf history --title ... --tags ...` uploads to release + updates `catalog.yml` and pushes
3. `shelfctl get <id>` downloads exactly one asset, verifies sha256, caches it
4. `shelfctl open <id>` opens cached file (or downloads then opens)
5. `shelfctl migrate one <old_path>` pulls one file from old monorepo via API and ingests to correct shelf

---

## Open-source safety wording (README requirement)

* “Manages user-provided files in user-owned GitHub repos/releases.”
* “Does not distribute content.”
* “Designed for personal document libraries (PDF/EPUB/etc.).”

---



