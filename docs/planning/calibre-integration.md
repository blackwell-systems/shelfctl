# Calibre Integration

> This is an exploratory design document, not a final spec. The goal is to think through what Calibre integration could look like, where the real complexity lives, and what a sensible phased approach might be. Nothing here is committed.

---

## Overview

Calibre is the most popular open-source ebook manager — the de facto tool for readers who take their libraries seriously. If shelfctl's target user is someone who already organizes books with care, there's a reasonable chance they already have a Calibre library sitting on their machine.

The natural question is: can shelfctl and Calibre coexist without friction, or even work together? They're solving adjacent but distinct problems. Calibre is a local-first, full-featured library manager with format conversion, metadata editing, and device sync. shelfctl is a GitHub-backed, cloud-native library manager focused on storage, organization, and portability. They don't have to compete — they can complement.

Integration would bridge local Calibre workflows with shelfctl's GitHub-backed storage, letting users who prefer Calibre's reading experience still benefit from shelfctl's distribution and backup model. Think of it as: Calibre as your local reading environment, shelfctl as your cloud shelf.

---

## Calibre Internals

Understanding how Calibre stores data is the foundation for any integration. The good news: it's well-structured and documented.

**Library directory**
Calibre stores its library in a single root directory, defaulting to `~/Calibre Library`. The location is configurable and most power users have moved it.

**`metadata.db`**
This is the heart of a Calibre library — a SQLite database at the root of the library directory. It contains all book metadata: title, author(s), tags, formats, ratings, comments, series, publisher, publication date, and more. The schema is stable across Calibre versions and well-understood by the community.

**Book file storage**
Actual ebook files live in `Author Name/Book Title/` subdirectories within the library. Each book can have multiple format files (e.g., both `book.epub` and `book.pdf`) alongside a cover image and an OPF metadata file.

**`calibredb` CLI**
Calibre ships a command-line tool, `calibredb`, that provides programmatic access to a library: `list`, `add`, `remove`, `set_metadata`, `export`, and more. It works against a library path and can output JSON. This is the official external interface.

**`ebook-convert` CLI**
Also ships with Calibre. Handles format conversion between epub, pdf, mobi, azw3, txt, and others. The key detail: it's a local tool, so any conversion workflow requires Calibre installed on the user's machine.

---

## Integration Directions

There are four meaningful directions this could go. They're not mutually exclusive, but they have very different complexity profiles.

### 1. Import from Calibre

The most obviously useful direction: a user has an existing Calibre library and wants to move some or all of it into shelfctl.

The workflow would look something like:
- Scan `metadata.db` directly (or shell out to `calibredb list`) to enumerate available books
- Map Calibre fields to shelfctl metadata (see the mapping table below)
- Let the user filter and select which books to import — by tag, author, format, or just "everything"
- Upload selected books as Release assets to a target shelf
- Write entries to `catalog.yml` with the mapped metadata

From the CLI, this could be:

```bash
shelfctl import --from-calibre ~/Calibre\ Library
shelfctl import --from-calibre ~/Calibre\ Library --tag "to-read"
shelfctl import --from-calibre ~/Calibre\ Library --author "Le Guin"
```

Or as a subcommand group:

```bash
shelfctl calibre import
shelfctl calibre import --filter tag=science-fiction
```

From the TUI, this would be an "Import from Calibre" option in the hub menu, launching an interactive picker over the Calibre library.

This direction is read-only from Calibre's perspective — we're only reading its data, never writing back. That keeps the risk low: the worst that happens is a bad import, not a corrupted Calibre library.

### 2. Export to Calibre

The reverse: a user has books in shelfctl and wants them available in their local Calibre library. Maybe they prefer Calibre's reading interface, or they want Calibre's device sync.

The workflow:
- Download the book file(s) from shelfctl
- Use `calibredb add` to add them to the local Calibre library, passing metadata flags for title, author, and tags
- Map shelfctl's `shelf` name to a Calibre tag or custom column (so the provenance is preserved)

```bash
shelfctl calibre export sicp --library ~/Calibre\ Library
shelfctl calibre export --shelf programming --library ~/Calibre\ Library
```

This requires Calibre installed (for `calibredb`) and the library path either specified or auto-detected. It's mostly a wrapper around `calibredb add` with some metadata massaging.

### 3. Two-way Sync

The most ambitious direction: keep a Calibre library and a set of shelfctl shelves in sync, detecting changes on either side and reconciling them.

This would need:
- A state snapshot or hash-based comparison to detect what changed since last sync
- A conflict resolution strategy when the same book is modified in both places (timestamp-based? prompt user?)
- Some durable identifier to correlate books across systems (Calibre's internal ID? title+author hash?)

This is genuinely hard to do well. The failure modes are subtle — duplicate entries, lost annotations, metadata drift — and users tend to trust their Calibre library deeply. Getting this wrong would be worse than not having it at all.

The honest assessment: this is probably a v2 feature at the earliest, and only worth building if Phase 1 and Phase 2 demonstrate real demand. Starting here would be a mistake.

### 4. Format Conversion

Calibre's `ebook-convert` is the best free format conversion tool available. It would be useful to surface this through shelfctl:

```bash
shelfctl convert sicp --format epub
```

The workflow: download the book, convert via `ebook-convert`, then upload the new format back to shelfctl. The question is whether to store multiple formats per book (a `pdf` and an `epub` as separate assets) or replace the original. Currently shelfctl assumes one file per book — storing multiple formats would require a metadata schema change.

This could also be useful independently of Calibre import/export: any shelfctl user with Calibre installed could convert formats on books already in their shelves.

---

## Technical Considerations

### Calibre Detection

Any feature that requires Calibre should degrade gracefully when it's not installed. The detection approach:

```go
// Check for calibredb in PATH
calibredbPath, err := exec.LookPath("calibredb")

// Check for ebook-convert in PATH
ebookConvertPath, err := exec.LookPath("ebook-convert")
```

On macOS, Calibre installs to `/Applications/calibre.app` and symlinks `calibredb` and `ebook-convert` into `/usr/bin` by default — but users sometimes have non-standard installations. Worth checking both PATH and common app bundle locations.

**Auto-detecting the library path** is trickier. Calibre stores its preferences in:
- Linux: `~/.config/calibre/global.py`
- macOS: `~/Library/Preferences/calibre/`
- Windows: `%APPDATA%\calibre\`

The `global.py` file contains the library path as a Python literal. Parsing it is doable but fragile — Python literal syntax is not JSON. Alternatively, we could just ask the user to provide `--library <path>` and only auto-detect as a convenience, not a requirement.

### Metadata Mapping

Calibre's schema is richer than shelfctl's current metadata model. Here's how the fields map:

| Calibre Field | shelfctl Field | Notes |
|---------------|----------------|-------|
| `title` | `title` | Direct map |
| `authors` | `author` | Calibre supports multiple authors; join with ` & ` |
| `tags` | `tags` | Direct map — both are string lists |
| `pubdate` | `year` | Extract year from the datetime |
| `formats` | (file) | Calibre stores multiple formats; we pick one (user choice?) |
| `comments` | — | No shelfctl equivalent currently; could drop or warn |
| `series` | — | Could map to a tag like `series:Foundation` |
| `series_index` | — | Could include in series tag: `series:Foundation:2` |
| `publisher` | — | No shelfctl equivalent; drop silently or warn |
| `identifiers` | — | Calibre stores ISBN, ASIN, etc.; no shelfctl equivalent |
| `rating` | — | No shelfctl equivalent |
| `cover` | (cover) | Could import as a cover image if shelfctl gains that field |

The `comments`, `publisher`, `rating`, and `identifiers` fields are the lossy parts. Users who care about those fields might be unhappy if they're silently dropped. A `--verbose` flag that lists what was dropped could help.

### Deduplication

When importing from Calibre, some books might already be in shelfctl. The matching strategy has tradeoffs:

- **Title + author match**: Fuzzy, but works without file access. Prone to false positives with common titles.
- **File hash match**: Reliable, but requires downloading from shelfctl (expensive) or computing the hash at upload time and storing it.
- **Calibre internal ID**: Only useful if we stored it during a previous import (implies state we're not tracking today).

For a first pass, title + author match with a prompt on collision seems like the right tradeoff: explicit, no hidden state, user stays in control.

Collision resolution options to offer:
- `--skip-existing` — silently skip books that appear to already be in the target shelf
- `--overwrite` — replace the existing shelfctl entry
- Default: prompt the user interactively for each collision

### Multiple Formats Per Book

This is probably the biggest schema question. Calibre happily stores `book.epub`, `book.pdf`, and `book.mobi` as three formats for the same logical book. shelfctl currently assumes one file per book.

Options:
1. **Pick one format at import time** — ask user which format to prefer (epub over pdf, etc.), import only that one. Simple, but loses formats.
2. **Import multiple formats as separate entries** — each format becomes a separate book record with a format tag. Ugly; search results get noisy.
3. **Extend shelfctl's schema to support multiple assets per book** — the right long-term answer, but a non-trivial change. Worth noting as a prerequisite for full-fidelity Calibre import.

For Phase 1, option 1 is probably fine with a `--prefer-format epub` flag.

### Direct DB vs. calibredb CLI

Two approaches for reading Calibre's library:

**Direct SQLite read**
- Faster — no subprocess overhead
- Can be done without Calibre installed
- Requires a Go SQLite driver (e.g., `modernc.org/sqlite` or `mattn/go-sqlite3`)
- Couples to Calibre's internal schema, which could theoretically change between Calibre versions (though it's been stable for years)

**Shell out to `calibredb list`**
- Simpler implementation — no new Go dependencies
- Always uses Calibre's own parsing, so schema compatibility is guaranteed
- Requires Calibre installed just to read the library (even for a dry-run preview)
- Slightly slower for large libraries

The direct DB approach is more robust for users who want to preview what's in their Calibre library before committing to an import. The CLI approach is safer against schema drift. A reasonable choice: direct DB for read operations (import preview, listing), `calibredb` for write operations (export to Calibre).

---

## Phased Approach

### Phase 1: Import from Calibre

Read-only access to Calibre library data. This covers the most common use case (migrating an existing Calibre collection into shelfctl) with the lowest risk profile.

Scope:
- Read `metadata.db` directly via SQLite
- List available books with metadata
- Interactive picker for selection (TUI) or `--filter` flags (CLI)
- Format preference selection when a book has multiple formats
- Upload selected books as Release assets to a target shelf
- Deduplication with skip/overwrite/prompt options
- Clear warnings for fields that don't map to shelfctl metadata

This alone is a complete feature. A user can onboard their entire Calibre library into shelfctl in one command.

### Phase 2: Export to Calibre + Format Conversion

Both of these are Calibre CLI wrappers, lower-risk than direct DB writes.

Export scope:
- `shelfctl calibre export` — download books from shelfctl, add to local Calibre library via `calibredb add`
- Map shelfctl shelf name to a Calibre tag for provenance tracking
- Support exporting one book, a selection, or an entire shelf

Format conversion scope:
- `shelfctl convert <id> --format <fmt>` — download, convert via `ebook-convert`, re-upload
- Decide on single-format vs. multi-format storage before implementing
- Clear error if `ebook-convert` is not in PATH

### Phase 3: Sync (if there's demand)

Bidirectional sync between a Calibre library and shelfctl shelves. Only worth building if Phase 1 and Phase 2 see real usage. The complexity cost is high enough that building it speculatively would be a mistake.

If it does get built:
- State tracking file (e.g., `.shelfctl-sync-state.json` in the library root)
- Hash-based change detection on both sides
- Explicit conflict resolution — never silently overwrite
- Dry-run mode before any destructive operation

---

## Open Questions

**Command structure**: Should Calibre features live under a `shelfctl calibre` subcommand group, or should they be flags on existing commands (`shelfctl import --from-calibre`, `shelfctl shelve --convert-via-calibre`)? A dedicated subcommand group is more discoverable and keeps the surface area clean. Flags feel more ergonomic for one-offs but scatter the feature.

**Direct DB vs. CLI**: The tradeoff is covered above. Worth deciding early since it affects the dependency footprint. No new Go deps is a meaningful advantage at this stage.

**Multiple formats per book**: This needs a decision before Phase 1 ships. The answer shapes the import UX and potentially the `catalog.yml` schema. Deferring it creates a compatibility cliff.

**Calibre internal ID preservation**: If we want Phase 3 sync to ever be feasible, we'd need to store Calibre's book ID somewhere during import (probably in a shelfctl custom metadata field or a separate state file). If we don't store it now, correlating books later requires fuzzy matching. Worth storing even in Phase 1 if sync is on the roadmap.

**Calibre plugin direction**: An alternative framing entirely — instead of shelfctl calling Calibre, a Calibre plugin that talks to shelfctl's GitHub backend. Calibre has a plugin system; a plugin could add a "Send to shelfctl shelf" action in Calibre's UI. This would require a different kind of work (Python, Calibre's plugin API) but might deliver a better UX for Calibre-native users. Not an either/or — but worth noting as a complementary approach.

**Scope creep risk**: Calibre is a large, complex application. Integration could easily grow to consume disproportionate development effort. The phased approach is intended to guard against this, but it's worth being explicit: Phase 1 should ship as a small, contained feature. If it requires significant changes to shelfctl's core data model (e.g., multi-format support, new metadata fields), those changes should be scoped and reviewed separately.

---

## Prior Art

**calibre-web** ([Janeczku/calibre-web](https://github.com/janeczku/calibre-web)): A web application that provides a browser interface to a Calibre library. Read-only view of a local Calibre database over HTTP. Popular self-hosting option. Not bidirectional — it reads from Calibre but doesn't modify it.

**COPS** ([seblucas/cops](https://github.com/seblucas/cops)): A lightweight OPDS server that serves a Calibre library over the OPDS catalog protocol. Useful for ereaders that support OPDS. Again, read-only.

Both projects treat the Calibre library as a data source rather than a sync target. shelfctl's integration would be different in a meaningful way: it's the first direction (Calibre → shelfctl) that overlaps with what these tools do, but the second direction (shelfctl → Calibre) and the conversion features have no equivalent in the existing ecosystem.

The fact that calibre-web and COPS are popular suggests there's real appetite for tools that bridge Calibre to other workflows. The gap they leave — getting books *out* of Calibre and *into* a cloud-backed system — is exactly where shelfctl could add value.
