# New User Friction Analysis

## Solved in v0.4.0

The following friction items were resolved in the v0.4.0 release:

1. **Root command help improvements** - Added "Getting Started" section with 4 critical first commands
2. **Init command documentation** - Documents key concepts (shelf, release, asset, catalog.yml) and clarifies required/optional flags
3. **Shelve command help** - Explains interactive vs non-interactive workflows, documents --shelf flag requirement
4. **Browse command help** - Documents all TUI key bindings and empty shelf behavior
5. **Open command help** - Explains cache behavior (location, auto-download, sync workflow)
6. **Config loading error messages** - Now include file path and hint to check permissions
7. **Missing config error** - Suggests running `shelfctl init --help`
8. **README restructuring** - 30-second quickstart at top, Core Concepts section, TUI Quick Reference callout
9. **GitHub token permissions** - Clarified that fine-grained tokens need BOTH Contents and Releases permissions
10. **Empty shelf guidance** - Browsing a shelf with zero books now displays helpful text explaining how to add books
11. **Comprehensive testing infrastructure** - 100+ tests, CI/CD automation, mock server for isolated testing

## Summary
shelfctl has strong documentation and clear help text, but the first-run experience still has some friction points. New users face a multi-step bootstrapping process (config file creation, token setup, owner configuration, shelf initialization) with no guided wizard to help them through it. Error messages generally provide actionable next steps, though some edge cases remain. The terminology (shelf/repo/release/catalog) is now better documented with the Core Concepts section.

## First-time Setup

- **[CRITICAL] GitHub owner must be set before init works**: Running `shelfctl init --repo shelf-test --name test --create-repo` fails with "error: --owner is required (or set github.owner in config)" but there's no config file yet. This is a chicken-and-egg problem.
  - **Severity**: High
  - **Suggested fix**: Allow `init` to prompt for owner on first run and create the config file automatically, or make `--owner` mandatory in the help text for first-time users

- **[MEDIUM] Missing prerequisites check**: The tool doesn't check if Git is installed (required for repo operations) or warn about optional dependencies like poppler (for PDF cover extraction).
  - **Severity**: Medium
  - **Suggested fix**: Add a `shelfctl doctor` command that checks prerequisites and reports status

## CLI Discoverability

- **[LOW] Default flag values sometimes confusing**: `init --private` defaults to true, which is good for security, but the help text shows it as a flag without explaining that private is the default.
  - **Severity**: Low
  - **Suggested fix**: Change help text to: "--private    Make the repo private (default: true, use --private=false for public)"

## TUI Navigation

- **[HIGH] TUI requires existing config and shelves**: Running `shelfctl` (TUI mode) without a config or without any shelves configured shows help text instead of launching an interactive setup wizard. The code has logic to detect this (hub_runner.go:24-28) but falls back to a legacy `runHub()` that isn't fully implemented.
  - **Severity**: High
  - **Suggested fix**: Complete the interactive setup wizard in `runHub()` to guide users through config creation, token setup, and first shelf creation

- **[MEDIUM] Error handling in TUI unclear**: Code shows TUI exits on action failures, but there's no indication in docs about what happens if a download fails, GitHub API times out, or the user has no internet connection.
  - **Severity**: Medium
  - **Suggested fix**: Document TUI error handling behavior in docs/guides/hub.md

## Terminology and Concepts

- **[MEDIUM] "Release" vs "release tag" confusion**: The tool defaults to a release tag called "library" but this isn't explained anywhere in first-run docs. A user might think "release" means a versioned release (v1.0) rather than a storage container.
  - **Severity**: Medium
  - **Suggested fix**: Use "storage release" or "asset release" in docs to distinguish from version releases, or explain in Prerequisites: "shelfctl uses GitHub release tags as storage containers (default: 'library')"

- **[LOW] catalog.yml exposure is inconsistent**: Sometimes the docs say "only metadata is versioned" (true), but it's not clear that users can directly edit catalog.yml or that it's just a YAML file in their repo. Advanced users might want to know this; new users don't need to.
  - **Severity**: Low
  - **Suggested fix**: Add a "For advanced users" section in Architecture doc explaining direct catalog.yml manipulation

## Common Failure Modes

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

- **[MEDIUM] No troubleshooting for common setup mistakes**: Troubleshooting doc focuses on runtime errors but doesn't cover setup mistakes like forgetting to add token to shell profile (so it works in current terminal but not after reboot).
  - **Severity**: Medium
  - **Suggested fix**: Add "Setup checklist" section: verify token persists across sessions, test with `echo $GITHUB_TOKEN`, etc.

- **[LOW] Migration workflow is prominent but not explained**: The README shows migration commands early but doesn't explain the use case (moving from a monolithic books repo to organized shelves). A first-time user might think migration is required.
  - **Severity**: Low
  - **Suggested fix**: Add context: "Already have PDFs committed in a GitHub repo? Use migration to reorganize them:" before showing scan/batch commands

- **[LOW] API rate limits mentioned but not quantified**: The README says "you're unlikely to hit rate limits" but doesn't give numbers (5000/hour authenticated) or explain what happens if you do hit them.
  - **Severity**: Low
  - **Suggested fix**: Add note: "Authenticated API allows 5,000 requests/hour. Typical usage (browsing, adding books) uses ~1-10 requests per operation."

## Remaining Quick Wins

1. **Add interactive config initialization**: Detect missing config on first run and prompt: "No config found. Create one now? (Y/n)". Collect GitHub username and token, create `~/.config/shelfctl/config.yml` automatically. This eliminates the biggest barrier to getting started.

2. **Show which shelf failed in validation errors**: Change `shelfctl shelves` error output to name the problematic shelf and the specific issue (repo not found, catalog missing, etc.). This turns a dead-end error into an actionable debugging step.
