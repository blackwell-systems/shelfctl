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
9. **GitHub token permissions** - Clarified that fine-grained tokens need Contents (Read/Write); Releases fall under Contents
10. **Empty shelf guidance** - Browsing a shelf with zero books now displays helpful text explaining how to add books
11. **Comprehensive testing infrastructure** - 100+ tests, CI/CD automation, mock server for isolated testing

## Solved in v0.4.7

12. **API rate limits quantified** - README now states "Authenticated API allows 5,000 requests/hour"
13. **Migration workflow context** - README now explains use case ("Already have PDFs committed in a repo?") before showing scan/batch commands
14. **Duplicate book ID documented** - `--force` flag documented in commands.md; behavior is explicit
15. **"Release" terminology** - README Core Concepts now explicitly states "NOT a version release" for the library release tag
16. **Token persistence** - Troubleshooting doc now includes shell profile instructions with `echo '...' >> ~/.zshrc`

## Summary

Bootstrapping friction remains the biggest barrier. New users face a multi-step setup (config file, token, owner, shelf init) with no guided wizard. Error messages are generally actionable but some edge cases remain.

---

## Remaining Items

### First-time Setup

- **[CRITICAL] GitHub owner chicken-and-egg**: Running `shelfctl init --repo shelf-test --name test --create-repo` fails with `--owner is required (or set github.owner in config)` but there's no config file yet. `init.go` returns the error without prompting.
  - **Suggested fix**: Auto-create config on first run — no new command needed. Both `shelfctl` (TUI) and `shelfctl init` should detect missing config, prompt for GitHub owner and token env var name inline, write `~/.config/shelfctl/config.yml`, then continue. Two questions, no wizard:
    ```
    No config found. Let's set up shelfctl.

    GitHub username or org: blackwell-systems
    Token env var name [SHELFCTL_GITHUB_TOKEN]:

    Config created at ~/.config/shelfctl/config.yml
    ```

- **[MEDIUM] No prerequisites check**: No check for Git installation or optional deps like poppler.
  - **Suggested fix**: `shelfctl doctor` command — on the roadmap

### TUI

- **[HIGH] TUI setup wizard not implemented**: `hub_runner.go` detects missing config/token and falls through to a text-based setup message. The interactive wizard path (comment: "For now, fall back to legacy runHub() for setup") is not completed.
  - **Suggested fix**: Same auto-create flow as above — detect missing config in `runUnifiedTUI()`, prompt inline, write config, then launch the TUI normally. No separate wizard needed.

- **[MEDIUM] TUI error behavior undocumented**: No docs covering what happens when downloads fail, GitHub API times out, or connectivity drops while in the TUI.
  - **Suggested fix**: Add a section to docs/guides/hub.md

### Error Messages

- **[HIGH] Network failures not actionable**: GitHub API 401/403/500 errors surface as raw HTTP status codes. No friendly messages for connectivity issues or rate limiting.
  - **Suggested fix**: Intercept common error codes and return user-readable messages

- **[HIGH] Invalid shelf config shows generic error**: `shelfctl shelves` fails with "one or more shelves have issues" (`shelves.go:60,69`) without naming the shelf or the specific issue.
  - **Suggested fix**: Name the shelf and issue: "Shelf 'programming' failed: repository not found"

- **[MEDIUM] Token scope errors cryptic**: 403 on private repo operations surfaces as a raw API error. No hint that `repo` scope is needed vs `public_repo`.
  - **Suggested fix**: Detect 403 on private repo operations and suggest checking token scope

### CLI

- **[LOW] `--private` default not surfaced**: `init --private` defaults to true but help text doesn't state this explicitly.
  - **Suggested fix**: Update help text to: `--private    Make repo private (default: true, use --private=false for public)`

- **[LOW] catalog.yml direct editing undocumented**: Advanced users may want to know they can edit catalog.yml directly. Not mentioned anywhere.
  - **Suggested fix**: Add a note in docs/reference/architecture.md under "For advanced users"
