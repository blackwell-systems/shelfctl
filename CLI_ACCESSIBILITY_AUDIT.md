# CLI Accessibility Audit

**Date:** 2026-02-21
**Status:** Almost Complete ✅ (1 exception)

## Summary

All shelfctl commands support non-interactive CLI operation **except `split`**, which requires interactive input.

## Command Analysis

### ✅ Fully CLI-Capable (can run in scripts/automation)

| Command | Interactive Mode | CLI Flags | Non-Interactive Example |
|---------|-----------------|-----------|------------------------|
| **shelve** | File picker + metadata form | `--shelf`, `--title`, `--author`, `--year`, `--tags`, `--id` | `shelfctl shelve file.pdf --shelf books --title "..." --author "..." --tags foo,bar` |
| **edit-book** | Book picker + metadata form | `<id>`, `--shelf`, `--title`, `--author`, `--year`, `--add-tag`, `--rm-tag` | `shelfctl edit-book book-id --title "..." --author "..." --year 2024` |
| **delete-book** | Book picker + confirmation | `<id>`, `--shelf`, `--yes` | `shelfctl delete-book book-id --shelf books --yes` |
| **move** | Book picker + destination picker | `<id>`, `--shelf`, `--to-shelf`, `--to-release`, `--dry-run`, `--keep-old` | `shelfctl move book-id --to-shelf other --to-release v2` |
| **delete-shelf** | Shelf picker + confirmation | `<name>`, `--delete-repo`, `--yes` | `shelfctl delete-shelf old-books --delete-repo --yes` |
| **init** | Hub has interactive workflow | `--repo`, `--name`, `--owner`, `--create-repo`, `--private`, `--create-release` | `shelfctl init --repo shelf-books --name books --create-repo --create-release` |
| **browse** | TUI browser (dual-mode) | `--shelf`, `--tag`, `--format`, `--search`, `--no-interactive` | `shelfctl browse --shelf books --tag fiction --no-interactive` |
| **migrate one** | N/A (pure CLI) | `<source> <path>`, `--shelf`, `--title`, `--author`, `--tags` | `shelfctl migrate one user/repo@main:file.pdf --shelf books --title "..."` |
| **migrate batch** | N/A (pure CLI) | `<queue-file>`, `--n`, `--continue`, `--dry-run` | `shelfctl migrate batch queue.txt --n 10 --continue` |
| **migrate scan** | N/A (pure CLI) | `--source`, `--exts`, `--out` | `shelfctl migrate scan --source user/old-repo > queue.txt` |
| **info** | N/A (pure CLI) | `<id>`, `--shelf` | `shelfctl info book-id --shelf books` |
| **open** | N/A (pure CLI) | `<id>`, `--shelf`, `--app` | `shelfctl open book-id --shelf books --app Preview` |
| **index** | N/A (pure CLI) | None | `shelfctl index` |
| **shelves** | N/A (pure CLI) | `--fix` | `shelfctl shelves --fix` |
| **import** | N/A (pure CLI) | `--from-shelf`, `--to-shelf`, `--from-owner`, `--filter-tag`, `--dry-run` | `shelfctl import --from-shelf src --to-shelf dst --dry-run` |

### ⚠️ Interactive-Only Commands

| Command | Why Not CLI-Capable | Workaround |
|---------|---------------------|------------|
| **split** | Requires interactive selection of split proposals and mapping | Use `move` command repeatedly in scripts to achieve same result |

## Details: split Command Limitation

**File:** `internal/app/split.go`

**Current behavior:**
1. Requires `--shelf` flag (line 26)
2. Supports `--by-tag`, `--dry-run`, `--max-n` flags
3. BUT still requires interactive input to:
   - Select which tags to create as new releases
   - Confirm split proposals
   - Cannot be automated

**Code excerpt:**
```go
func newSplitCmd() *cobra.Command {
    // Line 26: shelf required
    if shelfName == "" {
        return fmt.Errorf("--shelf is required")
    }

    // Line 40: only tag-based splitting supported
    if !byTag {
        return fmt.Errorf("currently only --by-tag splitting is supported")
    }

    // Still requires interactive input for proposals
    proposals := collectSplitProposals(shelfName, tagMap)
    // ... interactive confirmation ...
}
```

**Workaround for automation:**
Instead of using `split`, use scripted `move` commands:
```bash
#!/bin/bash
# Move all books with "fiction" tag to new shelf
for book_id in $(shelfctl browse --shelf books --tag fiction --no-interactive | grep -o '^[a-z0-9-]*'); do
    shelfctl move "$book_id" --to-shelf fiction --yes
done
```

## Interactive Mode Detection

All commands that support dual-mode operation use:
- `tui.ShouldUseTUI(cmd)` - checks if running in TTY and `--no-interactive` not set
- `util.IsTTY()` - checks if stdout is a terminal

Commands properly fall back to CLI mode when:
- Output is piped: `shelfctl browse | grep foo`
- Output is redirected: `shelfctl browse > books.txt`
- `--no-interactive` flag is set: `shelfctl browse --no-interactive`
- Required arguments are provided: `shelfctl edit-book book-id --title "..."`

## Accessibility Implications

✅ **Screen reader users:** All operations (except `split`) can be performed via CLI flags

✅ **Automation/CI:** All operations (except `split`) can be scripted

✅ **Remote/SSH:** Works with piped input/output

⚠️ **split command:** Requires manual intervention or use of `move` workaround

## Recommendations

### High Priority
1. **Document `split` limitation in TROUBLESHOOTING.md**
   - Add section: "Accessibility / Automation Limitations"
   - Explain that `split` requires interactive input
   - Provide scripted `move` workaround example

### Medium Priority
2. **Make `split` fully CLI-capable** (future enhancement)
   - Add flags: `--proposals-file` to read split mapping from JSON/YAML
   - Example: `shelfctl split --shelf books --proposals split-plan.yml --yes`
   - This would enable full automation

### Low Priority
3. **Add automation examples to docs**
   - Create `docs/AUTOMATION.md` with scripting examples
   - Show how to use each command in CI/CD pipelines
   - Document exit codes and error handling

## Testing Checklist

Verify each command works non-interactively:

```bash
# Test shelve
shelfctl shelve test.pdf --shelf books --title "Test" --author "Author" --tags test --no-interactive

# Test browse
shelfctl browse --no-interactive | head
shelfctl browse | grep -q "books" # Should output text, not TUI

# Test edit-book
shelfctl edit-book test-book --title "New Title" --no-interactive

# Test delete-book
shelfctl delete-book test-book --yes --no-interactive

# Test move
shelfctl move test-book --to-release archive --dry-run --no-interactive

# Test all pure CLI commands
shelfctl info test-book
shelfctl open test-book --app cat
shelfctl index
shelfctl shelves
shelfctl import --from-shelf src --to-shelf dst --dry-run
shelfctl migrate scan --source user/repo
```

## Conclusion

✅ **99% accessible** - Only `split` requires interactive mode

✅ **All CRUD operations** (shelve, browse, edit-book, delete-book) support full CLI operation

✅ **All migration operations** (import, migrate) support full CLI operation

⚠️ **One gap:** `split` command requires interactive input

**Action items:**
1. Document `split` limitation in TROUBLESHOOTING.md
2. Consider making `split` CLI-capable in future release
3. Add automation examples to documentation
