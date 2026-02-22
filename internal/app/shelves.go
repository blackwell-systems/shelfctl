package app

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type shelfStatus struct {
	name       string
	repo       string
	owner      string
	bookCount  int
	release    string
	repoOK     bool
	catalogOK  bool
	releaseOK  bool
	errorMsg   string
	needsFix   bool
}

func newShelvesCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:   "shelves",
		Short: "Validate all configured shelves",
		Long:  "Checks that each shelf repo exists, has a catalog.yml, and has the required release.",
		RunE: func(_ *cobra.Command, _ []string) error {
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

			// Render as table
			renderShelfTable(statuses)

			if anyFailed {
				fmt.Println()
				return fmt.Errorf("one or more shelves have issues (run with --fix to repair)")
			}
			fmt.Println()
			ok("All shelves healthy")
			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically repair missing catalog.yml or release")
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

func renderShelfTable(statuses []shelfStatus) {
	// Define styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	borderColor := lipgloss.Color("#383838")
	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor)

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
		// Remove ANSI codes for length calculation
		statusLen := lipgloss.Width(statusStr)
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
	fmt.Println(borderStyle.Render("┌" + strings.Repeat("─", totalWidth-2) + "┐"))

	// Title
	title := " Configured Shelves "
	padding := (totalWidth - len(title) - 2) / 2
	fmt.Println(borderStyle.Render("│") +
		strings.Repeat(" ", padding) +
		lipgloss.NewStyle().Bold(true).Render(title) +
		strings.Repeat(" ", totalWidth-len(title)-padding-2) +
		borderStyle.Render("│"))

	// Header separator
	fmt.Println(borderStyle.Render("├" + strings.Repeat("─", maxNameLen) +
		"┬" + strings.Repeat("─", maxRepoLen) +
		"┬" + strings.Repeat("─", maxBooksLen) +
		"┬" + strings.Repeat("─", maxStatusLen) + "┤"))

	// Headers
	fmt.Print(borderStyle.Render("│"))
	fmt.Print(headerStyle.Width(maxNameLen - 2).Render("Shelf"))
	fmt.Print(borderStyle.Render("│"))
	fmt.Print(headerStyle.Width(maxRepoLen - 2).Render("Repository"))
	fmt.Print(borderStyle.Render("│"))
	fmt.Print(headerStyle.Width(maxBooksLen - 2).Render("Books"))
	fmt.Print(borderStyle.Render("│"))
	fmt.Print(headerStyle.Width(maxStatusLen - 2).Render("Status"))
	fmt.Println(borderStyle.Render("│"))

	// Data separator
	fmt.Println(borderStyle.Render("├" + strings.Repeat("─", maxNameLen) +
		"┼" + strings.Repeat("─", maxRepoLen) +
		"┼" + strings.Repeat("─", maxBooksLen) +
		"┼" + strings.Repeat("─", maxStatusLen) + "┤"))

	// Data rows
	for i, s := range statuses {
		fmt.Print(borderStyle.Render("│"))
		fmt.Print(cellStyle.Width(maxNameLen - 2).Render(s.name))
		fmt.Print(borderStyle.Render("│"))
		fmt.Print(cellStyle.Width(maxRepoLen - 2).Render(s.repo))
		fmt.Print(borderStyle.Render("│"))
		fmt.Print(cellStyle.Width(maxBooksLen - 2).Render(formatBookCount(s.bookCount)))
		fmt.Print(borderStyle.Render("│"))
		statusStr := formatStatus(s)
		fmt.Print(cellStyle.Width(maxStatusLen - 2).Render(statusStr))
		fmt.Println(borderStyle.Render("│"))

		// Add separator between rows if not last
		if i < len(statuses)-1 {
			fmt.Println(borderStyle.Render("├" + strings.Repeat("─", maxNameLen) +
				"┼" + strings.Repeat("─", maxRepoLen) +
				"┼" + strings.Repeat("─", maxBooksLen) +
				"┼" + strings.Repeat("─", maxStatusLen) + "┤"))
		}
	}

	// Bottom border
	fmt.Println(borderStyle.Render("└" + strings.Repeat("─", maxNameLen) +
		"┴" + strings.Repeat("─", maxRepoLen) +
		"┴" + strings.Repeat("─", maxBooksLen) +
		"┴" + strings.Repeat("─", maxStatusLen) + "┘"))
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
