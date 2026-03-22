# New User Friction Analysis

## Summary
shelfctl has strong documentation and clear help text, but the first-run experience has critical friction points. New users face a multi-step bootstrapping process (config file creation, token setup, owner configuration, shelf initialization) with no guided wizard to help them through it. The README jumps quickly to advanced features (migration, splitting) before establishing basics, and error messages don't always provide actionable next steps. The terminology (shelf/repo/release/catalog) requires understanding GitHub's release asset model, which isn't intuitive for users who haven't worked with this pattern before.

## First-time Setup

- **[CRITICAL] No automatic config initialization**: When a user runs `shelfctl` for the first time without a config file at `~/.config/shelfctl/config.yml`, the tool shows help text but doesn't offer to create the config. The user must manually create the directory structure and file, or know to run `init` first.
  - **Severity**: High
  - **Suggested fix**: Add an interactive setup wizard that runs on first launch, prompting for GitHub username and offering to create the config file. Or at minimum, detect missing config and print: "Config file not found. Run `shelfctl init --help` to get started, or manually create ~/.config/shelfctl/config.yml"

- **[CRITICAL] GitHub owner must be set before init works**: Running `shelfctl init --repo shelf-test --name test --create-repo` fails with "error: --owner is required (or set github.owner in config)" but there's no config file yet. This is a chicken-and-egg problem.
  - **Severity**: High
  - **Suggested fix**: Allow `init` to prompt for owner on first run and create the config file automatically, or make `--owner` mandatory in the help text for first-time users

- **[HIGH] Token setup is documented but not enforced progressively**: The README explains token creation well, but when a user runs any command without `GITHUB_TOKEN` set, they get a terse error: "no GitHub token found — set GITHUB_TOKEN or SHELFCTL_GITHUB_TOKEN". This doesn't link back to setup instructions.
  - **Severity**: High
  - **Suggested fix**: Include a link in the error message: "No GitHub token found. See https://github.com/blackwell-systems/shelfctl#authentication or run with GITHUB_TOKEN=your_token"

- **[MEDIUM] Missing prerequisites check**: The tool doesn't check if Git is installed (required for repo operations) or warn about optional dependencies like poppler (for PDF cover extraction).
  - **Severity**: Medium
  - **Suggested fix**: Add a `shelfctl doctor` command that checks prerequisites and reports status

- **[LOW] README structure front-loads complexity**: The "Quick start" section shows migration commands before showing the simplest flow (add one book). A brand-new user without existing PDFs in repos will skip past the migration examples but they occupy prime real estate.
  - **Severity**: Low
  - **Suggested fix**: Restructure README: move "Starting fresh? Add books directly" section before "Already have PDFs in GitHub repos?"

## CLI Discoverability

- **[HIGH] Critical flags not shown in examples**: The `shelve` command help shows examples but doesn't emphasize that `--shelf` is effectively required on first use (or the tool will prompt). Similarly, `init` examples don't show that you need `--create-repo` to actually create the repo.
  - **Severity**: High
  - **Suggested fix**: Add "(required)" annotations in help text for effectively-required flags, or show interactive vs non-interactive paths more clearly

- **[MEDIUM] Terminology gap in help text**: Commands use "shelf" and "release" as if the user already understands the architecture. `open` help says "downloads to cache first if needed" but doesn't explain what the cache is or where it lives.
  - **Severity**: Medium
  - **Suggested fix**: Add a brief terminology section to `--help` output, or link to docs: "Run 'shelfctl --help' for concepts, or see docs/reference/architecture.md"

- **[MEDIUM] No clear "getting started" command**: `shelfctl --help` lists 20+ commands with no indication of which ones a new user needs first (hint: `init`, then `shelve`, then `browse`).
  - **Severity**: Medium
  - **Suggested fix**: Add a "Getting Started" section at the top of help output: "New user? Start with: shelfctl init, then shelfctl shelve"

- **[LOW] Default flag values sometimes confusing**: `init --private` defaults to true, which is good for security, but the help text shows it as a flag without explaining that private is the default.
  - **Severity**: Low
  - **Suggested fix**: Change help text to: "--private    Make the repo private (default: true, use --private=false for public)"

## TUI Navigation

- **[HIGH] TUI requires existing config and shelves**: Running `shelfctl` (TUI mode) without a config or without any shelves configured shows help text instead of launching an interactive setup wizard. The code has logic to detect this (hub_runner.go:24-28) but falls back to a legacy `runHub()` that isn't fully implemented.
  - **Severity**: High
  - **Suggested fix**: Complete the interactive setup wizard in `runHub()` to guide users through config creation, token setup, and first shelf creation

- **[MEDIUM] No TUI preview in README**: The README shows a TUI demo GIF but doesn't explain what keys to press or what the workflow is. The hub.md doc is comprehensive but users might not find it.
  - **Severity**: Medium
  - **Suggested fix**: Add a "TUI Quick Reference" callout in README with basic keybindings (↑/↓ navigate, enter select, q quit, / search)

- **[MEDIUM] Error handling in TUI unclear**: Code shows TUI exits on action failures, but there's no indication in docs about what happens if a download fails, GitHub API times out, or the user has no internet connection.
  - **Severity**: Medium
  - **Suggested fix**: Document TUI error handling behavior in docs/guides/hub.md

- **[LOW] Key bindings not mentioned in main help**: The `browse` command launches a TUI but `browse --help` doesn't mention the interactive keybindings (o to open, space to select, g to download). Users have to discover this through trial.
  - **Severity**: Low
  - **Suggested fix**: Add "(press ? in TUI for help)" to browse command description

## Terminology and Concepts

- **[HIGH] "Shelf" metaphor not immediately clear**: The README explains that a shelf is a GitHub repo, but the relationship between shelf → repo → release → catalog.yml → asset isn't explained until deep in the Architecture doc. A new user seeing "shelf-programming" might not realize this creates a public GitHub repo visible to anyone.
  - **Severity**: High
  - **Suggested fix**: Add a "Core Concepts" section early in README with a simple diagram: Shelf (GitHub repo) → Release (library tag) → Assets (your PDFs) + catalog.yml (metadata)

- **[MEDIUM] "Release" vs "release tag" confusion**: The tool defaults to a release tag called "library" but this isn't explained anywhere in first-run docs. A user might think "release" means a versioned release (v1.0) rather than a storage container.
  - **Severity**: Medium
  - **Suggested fix**: Use "storage release" or "asset release" in docs to distinguish from version releases, or explain in Prerequisites: "shelfctl uses GitHub release tags as storage containers (default: 'library')"

- **[MEDIUM] Cache behavior not explained until needed**: The README mentions "on-demand downloads" and "cache" but doesn't explain that the cache lives at `~/.local/share/shelfctl/cache/` or that it can grow large. Users discover this when their disk fills up.
  - **Severity**: Medium
  - **Suggested fix**: Add cache location and `cache info`/`cache clear` commands to Quick Start section

- **[LOW] catalog.yml exposure is inconsistent**: Sometimes the docs say "only metadata is versioned" (true), but it's not clear that users can directly edit catalog.yml or that it's just a YAML file in their repo. Advanced users might want to know this; new users don't need to.
  - **Severity**: Low
  - **Suggested fix**: Add a "For advanced users" section in Architecture doc explaining direct catalog.yml manipulation

## Common Failure Modes

- **[CRITICAL] Empty shelf shows no guidance**: Running `shelfctl browse` on a new shelf with zero books shows an empty list. Code handles this (hub_runner.go:34-44) but only when shelves are unconfigured, not when a shelf exists but has no books.
  - **Severity**: High
  - **Suggested fix**: In TUI browse mode, detect empty shelf and show helpful text: "No books yet. Press 'q' to return to menu and select 'Add Book' to upload your first PDF."

- **[HIGH] Network failures aren't actionable**: When GitHub API is unreachable or rate-limited, errors are technical (401, 403, 500). The troubleshooting doc covers 401/403 for auth, but not rate limiting or general connectivity.
  - **Severity**: High
  - **Suggested fix**: Add network error handling with user-friendly messages: "Unable to reach GitHub API. Check your internet connection or try again later."

- **[HIGH] Invalid shelf config shows generic error**: Running `shelfctl shelves` with a shelf that has an invalid repo name fails with "error: one or more shelves have issues" but doesn't say which shelf or what the issue is.
  - **Severity**: High
  - **Suggested fix**: Make error messages specific: "Shelf 'programming' failed: repository 'username/shelf-programming' not found"

- **[MEDIUM] Token scope errors are cryptic**: If a user creates a token with only `public_repo` scope but tries to create a private shelf, the error is a GitHub API 403. The troubleshooting doc mentions this but users might not connect the dots.
  - **Severity**: Medium
  - **Suggested fix**: Detect 403 on private repo operations and suggest: "Permission denied. Ensure your GitHub token has 'repo' scope (not just 'public_repo')"

- **[LOW] Duplicate book ID not prevented**: If a user tries to `shelve` a book with an ID that already exists, the behavior isn't documented (does it overwrite? fail?).
  - **Severity**: Low
  - **Suggested fix**: Document behavior in commands.md and consider adding a `--force` flag to allow overwrites

## Documentation Gaps

- **[HIGH] No "First 5 Minutes" quickstart**: The tutorial is comprehensive (13KB) but there's no ultra-condensed version for users who just want to add one book. The README Quick Start mixes setup with advanced features.
  - **Severity**: High
  - **Suggested fix**: Create docs/guides/quickstart.md: "1. Set GITHUB_TOKEN. 2. Run `shelfctl init --repo shelf-books --name books --create-repo --owner YOUR_USERNAME`. 3. Run `shelfctl shelve ~/book.pdf --shelf books`. Done."

- **[MEDIUM] GitHub token permissions not in README**: The Authentication section lists required scopes but doesn't explain what happens if you get them wrong, or that fine-grained tokens need both Contents and Releases permissions (not just one).
  - **Severity**: Medium
  - **Suggested fix**: Add troubleshooting inline: "Fine-grained tokens need BOTH Contents (Read/Write) AND Releases (Read/Write)"

- **[MEDIUM] No troubleshooting for common setup mistakes**: Troubleshooting doc focuses on runtime errors but doesn't cover setup mistakes like forgetting to add token to shell profile (so it works in current terminal but not after reboot).
  - **Severity**: Medium
  - **Suggested fix**: Add "Setup checklist" section: verify token persists across sessions, test with `echo $GITHUB_TOKEN`, etc.

- **[LOW] Migration workflow is prominent but not explained**: The README shows migration commands early but doesn't explain the use case (moving from a monolithic books repo to organized shelves). A first-time user might think migration is required.
  - **Severity**: Low
  - **Suggested fix**: Add context: "Already have PDFs committed in a GitHub repo? Use migration to reorganize them:" before showing scan/batch commands

- **[LOW] API rate limits mentioned but not quantified**: The README says "you're unlikely to hit rate limits" but doesn't give numbers (5000/hour authenticated) or explain what happens if you do hit them.
  - **Severity**: Low
  - **Suggested fix**: Add note: "Authenticated API allows 5,000 requests/hour. Typical usage (browsing, adding books) uses ~1-10 requests per operation."

## Quick Wins

1. **Add interactive config initialization**: Detect missing config on first run and prompt: "No config found. Create one now? (Y/n)". Collect GitHub username and token, create `~/.config/shelfctl/config.yml` automatically. This eliminates the biggest barrier to getting started.

2. **Improve error messages with next steps**: Change "no GitHub token found" to "No GitHub token found. Set GITHUB_TOKEN environment variable. See: https://github.com/blackwell-systems/shelfctl#authentication". Apply this pattern to all first-run errors.

3. **Add 30-second quickstart to README**: Create a single code block at the top showing the absolute minimum to add one book: token setup → init → shelve. Currently users have to read through Prerequisites, Install, Authentication, Quick start (which shows migration first).

4. **Show which shelf failed in validation errors**: Change `shelfctl shelves` error output to name the problematic shelf and the specific issue (repo not found, catalog missing, etc.). This turns a dead-end error into an actionable debugging step.

5. **Add help text to empty TUI states**: When browse shows zero books, display centered help text: "No books in this library yet. Press 'q' and select 'Add Book' to get started." Similarly for empty cache, empty search results, etc.
