package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type shelfStatus struct {
	name      string
	repo      string
	owner     string
	bookCount int
	release   string
	repoOK    bool
	catalogOK bool
	releaseOK bool
	errorMsg  string
	needsFix  bool
}

func newShelvesCmd() *cobra.Command {
	var fix bool
	var tableMode bool

	cmd := &cobra.Command{
		Use:   "shelves",
		Short: "Validate all configured shelves",
		Long:  "Checks that each shelf repo exists, has a catalog.yml, and has the required release.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(cfg.Shelves) == 0 {
				warn("No shelves configured. Run: shelfctl init --repo shelf-<topic> --name <topic>")
				return nil
			}

			// Collect all shelf statuses
			statuses := make([]shelfStatus, 0, len(cfg.Shelves))
			anyFailed := false

			for _, shelf := range cfg.Shelves {
				status := collectShelfStatus(shelf, fix)
				statuses = append(statuses, status)
				if !status.repoOK || !status.catalogOK || !status.releaseOK {
					anyFailed = true
				}
			}

			// Render based on mode
			if tableMode {
				// Table mode: show formatted table
				renderShelfTable(statuses)

				if anyFailed {
					fmt.Println()
					return fmt.Errorf("one or more shelves have issues (run with --fix to repair)")
				}
				fmt.Println()
				ok("All shelves healthy")
			} else {
				// Default: simple list for scriptability
				renderShelfList(statuses)

				if anyFailed {
					return fmt.Errorf("one or more shelves have issues")
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically repair missing catalog.yml or release")
	cmd.Flags().BoolVar(&tableMode, "table", false, "Display as formatted table (default: simple list)")
	return cmd
}

func collectShelfStatus(shelf config.ShelfConfig, fix bool) shelfStatus {
	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	release := shelf.EffectiveRelease(cfg.Defaults.Release)
	catalogPath := shelf.EffectiveCatalogPath()

	status := shelfStatus{
		name:    shelf.Name,
		repo:    shelf.Repo,
		owner:   owner,
		release: release,
	}

	// 1. Check repo exists
	exists, err := gh.RepoExists(owner, shelf.Repo)
	if err != nil {
		status.errorMsg = fmt.Sprintf("repo error: %v", err)
		return status
	}
	if !exists {
		status.errorMsg = "repo not found"
		return status
	}
	status.repoOK = true

	// 2. Check catalog.yml exists
	catalogData, _, catalogErr := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if catalogErr != nil {
		if fix {
			if err := gh.CommitFile(owner, shelf.Repo, catalogPath, []byte("[]\n"), "init: add catalog.yml"); err != nil {
				status.errorMsg = fmt.Sprintf("catalog fix failed: %v", err)
				return status
			}
			status.catalogOK = true
			status.bookCount = 0
		} else {
			status.needsFix = true
			status.errorMsg = "catalog.yml missing"
			return status
		}
	} else {
		status.catalogOK = true
		// Count books
		if books, err := catalog.Parse(catalogData); err == nil {
			status.bookCount = len(books)
		}
	}

	// 3. Check release exists
	_, err = gh.GetReleaseByTag(owner, shelf.Repo, release)
	if err != nil {
		if fix {
			_, err = gh.CreateRelease(owner, shelf.Repo, release, release)
			if err != nil {
				status.errorMsg = fmt.Sprintf("release fix failed: %v", err)
				return status
			}
			status.releaseOK = true
		} else {
			status.needsFix = true
			status.errorMsg = fmt.Sprintf("release '%s' missing", release)
			return status
		}
	} else {
		status.releaseOK = true
	}

	return status
}

func renderShelfList(statuses []shelfStatus) {
	for _, s := range statuses {
		fmt.Printf("%s\t%s\t%d\n", s.name, s.repo, s.bookCount)
	}
}

func renderShelfTable(statuses []shelfStatus) {
	gray := color.New(color.FgHiBlack).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	headerBg := color.New(color.FgCyan, color.Bold).SprintFunc()

	// Calculate column widths
	maxNameLen := len("Shelf")
	maxRepoLen := len("Repository")
	maxBooksLen := len("Books")
	maxStatusLen := len("Status")

	for _, s := range statuses {
		if len(s.name) > maxNameLen {
			maxNameLen = len(s.name)
		}
		if len(s.repo) > maxRepoLen {
			maxRepoLen = len(s.repo)
		}
		booksStr := formatBookCount(s.bookCount)
		if len(booksStr) > maxBooksLen {
			maxBooksLen = len(booksStr)
		}
		statusStr := formatStatus(s)
		// Remove ANSI codes for length calculation (count runes not bytes for Unicode)
		statusLen := utf8.RuneCountInString(stripAnsi(statusStr))
		if statusLen > maxStatusLen {
			maxStatusLen = statusLen
		}
	}

	// Add padding
	maxNameLen += 2
	maxRepoLen += 2
	maxBooksLen += 2
	maxStatusLen += 2

	totalWidth := maxNameLen + maxRepoLen + maxBooksLen + maxStatusLen + 5 // +5 for borders

	// Top border
	fmt.Println(gray("┌" + strings.Repeat("─", totalWidth-2) + "┐"))

	// Title
	title := " Configured Shelves "
	padding := (totalWidth - len(title) - 2) / 2
	fmt.Println(gray("│") +
		strings.Repeat(" ", padding) +
		bold(title) +
		strings.Repeat(" ", totalWidth-len(title)-padding-2) +
		gray("│"))

	// Header separator
	fmt.Println(gray("├" + strings.Repeat("─", maxNameLen) +
		"┬" + strings.Repeat("─", maxRepoLen) +
		"┬" + strings.Repeat("─", maxBooksLen) +
		"┬" + strings.Repeat("─", maxStatusLen) + "┤"))

	// Headers
	fmt.Printf("%s %s %s %s %s %s %s %s %s\n",
		gray("│"),
		headerBg(padRight("Shelf", maxNameLen-2)),
		gray("│"),
		headerBg(padRight("Repository", maxRepoLen-2)),
		gray("│"),
		headerBg(padRight("Books", maxBooksLen-2)),
		gray("│"),
		headerBg(padRight("Status", maxStatusLen-2)),
		gray("│"))

	// Data separator
	fmt.Println(gray("├" + strings.Repeat("─", maxNameLen) +
		"┼" + strings.Repeat("─", maxRepoLen) +
		"┼" + strings.Repeat("─", maxBooksLen) +
		"┼" + strings.Repeat("─", maxStatusLen) + "┤"))

	// Data rows
	for i, s := range statuses {
		statusStr := formatStatus(s)
		fmt.Printf("%s %s %s %s %s %s %s %s %s\n",
			gray("│"),
			padRight(s.name, maxNameLen-2),
			gray("│"),
			padRight(s.repo, maxRepoLen-2),
			gray("│"),
			padRight(formatBookCount(s.bookCount), maxBooksLen-2),
			gray("│"),
			padRightColored(statusStr, maxStatusLen-2),
			gray("│"))

		// Add separator between rows if not last
		if i < len(statuses)-1 {
			fmt.Println(gray("├" + strings.Repeat("─", maxNameLen) +
				"┼" + strings.Repeat("─", maxRepoLen) +
				"┼" + strings.Repeat("─", maxBooksLen) +
				"┼" + strings.Repeat("─", maxStatusLen) + "┤"))
		}
	}

	// Bottom border
	fmt.Println(gray("└" + strings.Repeat("─", maxNameLen) +
		"┴" + strings.Repeat("─", maxRepoLen) +
		"┴" + strings.Repeat("─", maxBooksLen) +
		"┴" + strings.Repeat("─", maxStatusLen) + "┘"))
}

// padRight pads a string to the specified width with spaces
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padRightColored pads a colored string accounting for ANSI codes
func padRightColored(s string, width int) string {
	plainLen := utf8.RuneCountInString(stripAnsi(s))
	if plainLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-plainLen)
}

// stripAnsi removes ANSI color codes from a string for length calculation
func stripAnsi(s string) string {
	// Simple ANSI escape sequence stripper
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

func formatBookCount(count int) string {
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

func formatStatus(s shelfStatus) string {
	if !s.repoOK {
		return color.RedString("✗ ") + s.errorMsg
	}
	if !s.catalogOK {
		if s.needsFix {
			return color.YellowString("⚠ ") + s.errorMsg
		}
		return color.RedString("✗ ") + s.errorMsg
	}
	if !s.releaseOK {
		if s.needsFix {
			return color.YellowString("⚠ ") + s.errorMsg
		}
		return color.RedString("✗ ") + s.errorMsg
	}
	return color.GreenString("✓ Healthy")
}
