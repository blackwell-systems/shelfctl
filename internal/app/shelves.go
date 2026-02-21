package app

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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

			anyFailed := false
			for _, shelf := range cfg.Shelves {
				owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
				release := shelf.EffectiveRelease(cfg.Defaults.Release)
				catalogPath := shelf.EffectiveCatalogPath()

				header("Shelf: %s  (%s/%s)", shelf.Name, owner, shelf.Repo)

				// 1. Repo exists?
				exists, err := gh.RepoExists(owner, shelf.Repo)
				if err != nil {
					fmt.Printf("  %-12s %s\n", "repo:", color.RedString("error — %v", err))
					anyFailed = true
					continue
				}
				if !exists {
					fmt.Printf("  %-12s %s\n", "repo:", color.RedString("NOT FOUND"))
					anyFailed = true
					continue
				}
				fmt.Printf("  %-12s %s\n", "repo:", color.GreenString("ok"))

				// 2. catalog.yml exists?
				catalogData, _, catalogErr := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
				bookCount := 0
				if catalogErr != nil {
					fmt.Printf("  %-12s %s", "catalog.yml:", color.YellowString("missing"))
					if fix {
						if err := gh.CommitFile(owner, shelf.Repo, catalogPath, []byte("[]\n"), "init: add catalog.yml"); err != nil {
							fmt.Printf(" — fix failed: %v\n", err)
							anyFailed = true
						} else {
							fmt.Printf(" — %s\n", color.GreenString("created"))
						}
					} else {
						fmt.Printf(" (use --fix to create)\n")
						anyFailed = true
					}
				} else {
					// Count books in catalog
					if books, err := catalog.Parse(catalogData); err == nil {
						bookCount = len(books)
					}
					if bookCount > 0 {
						fmt.Printf("  %-12s %s (%d books)\n", "catalog.yml:", color.GreenString("ok"), bookCount)
					} else {
						fmt.Printf("  %-12s %s (empty)\n", "catalog.yml:", color.GreenString("ok"))
					}
				}

				// 3. Release exists?
				rel, err := gh.GetReleaseByTag(owner, shelf.Repo, release)
				if err != nil {
					fmt.Printf("  %-12s %s", fmt.Sprintf("release(%s):", release), color.YellowString("missing"))
					if fix {
						rel, err = gh.CreateRelease(owner, shelf.Repo, release, release)
						if err != nil {
							fmt.Printf(" — fix failed: %v\n", err)
							anyFailed = true
						} else {
							fmt.Printf(" — %s (%s)\n", color.GreenString("created"), rel.HTMLURL)
						}
					} else {
						fmt.Printf(" (use --fix to create)\n")
						anyFailed = true
					}
				} else {
					fmt.Printf("  %-12s %s  id=%d\n", fmt.Sprintf("release(%s):", release), color.GreenString("ok"), rel.ID)
				}
			}

			if anyFailed {
				return fmt.Errorf("one or more shelves have issues (run with --fix to repair)")
			}
			ok("All shelves healthy")
			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically repair missing catalog.yml or release")
	return cmd
}
