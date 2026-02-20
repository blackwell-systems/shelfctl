package app

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newBrowseCmd() *cobra.Command {
	var (
		shelfName string
		tag       string
		search    string
		format    string
	)

	cmd := &cobra.Command{
		Use:     "browse",
		Aliases: []string{"ls"},
		Short:   "Browse your library (interactive TUI or text output)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var shelves []config.ShelfConfig
			if shelfName != "" {
				s := cfg.ShelfByName(shelfName)
				if s == nil {
					return fmt.Errorf("shelf %q not found in config", shelfName)
				}
				shelves = []config.ShelfConfig{*s}
			} else {
				shelves = cfg.Shelves
			}

			if len(shelves) == 0 {
				warn("No shelves configured.")
				return nil
			}

			f := catalog.Filter{Tag: tag, Search: search, Format: format}

			// Check if we should use TUI mode
			if tui.ShouldUseTUI(cmd) {
				// Collect all book data for TUI
				var allItems []tui.BookItem
				for i := range shelves {
					shelf := &shelves[i]
					owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
					catalogPath := shelf.EffectiveCatalogPath()

					data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
					if err != nil {
						warn("Could not load catalog for shelf %q: %v", shelf.Name, err)
						continue
					}
					books, err := catalog.Parse(data)
					if err != nil {
						warn("Could not parse catalog for shelf %q: %v", shelf.Name, err)
						continue
					}

					matched := f.Apply(books)
					for _, b := range matched {
						cached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)
						allItems = append(allItems, tui.BookItem{
							Book:      b,
							ShelfName: shelf.Name,
							Cached:    cached,
							Owner:     owner,
							Repo:      shelf.Repo,
						})
					}
				}

				if len(allItems) == 0 {
					warn("No books found.")
					return nil
				}

				return tui.RunListBrowser(allItems)
			}

			// CLI mode: use existing text output
			total := 0

			for i := range shelves {
				shelf := &shelves[i]
				owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
				catalogPath := shelf.EffectiveCatalogPath()

				data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
				if err != nil {
					warn("Could not load catalog for shelf %q: %v", shelf.Name, err)
					continue
				}
				books, err := catalog.Parse(data)
				if err != nil {
					warn("Could not parse catalog for shelf %q: %v", shelf.Name, err)
					continue
				}

				matched := f.Apply(books)
				if len(matched) == 0 {
					continue
				}

				header("── %s  (%d books)", shelf.Name, len(matched))
				for _, b := range matched {
					cachedMark := ""
					if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
						cachedMark = color.GreenString(" ✓")
					}
					tagStr := ""
					if len(b.Tags) > 0 {
						tagStr = " " + color.CyanString("["+strings.Join(b.Tags, ",")+"]")
					}
					fmt.Printf("  %-22s  %s%s%s\n",
						color.WhiteString(b.ID),
						b.Title,
						tagStr,
						cachedMark,
					)
				}
				total += len(matched)
			}

			if total == 0 {
				fmt.Println("No books found.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Filter to a specific shelf")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&search, "search", "", "Full-text search (title, author, tags)")
	cmd.Flags().StringVar(&format, "format", "", "Filter by format (pdf, epub, …)")
	return cmd
}
