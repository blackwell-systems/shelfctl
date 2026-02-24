package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newTagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List and manage tags across your library",
		Long:  "List all tags with book counts, or rename tags in bulk.",
	}

	cmd.AddCommand(
		newTagsListCmd(),
		newTagsRenameCmd(),
	)

	// Make `shelfctl tags` with no subcommand default to list
	cmd.RunE = newTagsListCmd().RunE

	return cmd
}

func newTagsListCmd() *cobra.Command {
	var (
		shelfName string
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tags with book counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			shelves := cfg.Shelves
			if shelfName != "" {
				s := cfg.ShelfByName(shelfName)
				if s == nil {
					return fmt.Errorf("shelf %q not found in config", shelfName)
				}
				shelves = []config.ShelfConfig{*s}
			}

			if len(shelves) == 0 {
				warn("No shelves configured")
				return nil
			}

			counts := collectTagCounts(shelves)

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(counts)
			}

			if len(counts) == 0 {
				fmt.Println("No tags found.")
				return nil
			}

			// Sort by count descending, then name ascending
			type tagEntry struct {
				Name  string
				Count int
			}
			var entries []tagEntry
			for name, count := range counts {
				entries = append(entries, tagEntry{name, count})
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].Count != entries[j].Count {
					return entries[i].Count > entries[j].Count
				}
				return entries[i].Name < entries[j].Name
			})

			for _, e := range entries {
				fmt.Printf("  %-24s %s\n",
					color.CyanString(e.Name),
					color.HiBlackString("(%d)", e.Count),
				)
			}
			fmt.Printf("\n%d tags across %d shelves\n", len(entries), len(shelves))
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "List tags for a specific shelf")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func newTagsRenameCmd() *cobra.Command {
	var (
		shelfName string
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a tag across all books",
		Long: `Rename all occurrences of a tag across shelves.

Examples:
  shelfctl tags rename "prog" "programming"
  shelfctl tags rename ml machine-learning --shelf research
  shelfctl tags rename "prog" "programming" --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldTag := args[0]
			newTag := args[1]

			shelves := cfg.Shelves
			if shelfName != "" {
				s := cfg.ShelfByName(shelfName)
				if s == nil {
					return fmt.Errorf("shelf %q not found in config", shelfName)
				}
				shelves = []config.ShelfConfig{*s}
			}

			if len(shelves) == 0 {
				warn("No shelves configured")
				return nil
			}

			totalRenamed := 0

			for i := range shelves {
				shelf := &shelves[i]
				owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
				catalogPath := shelf.EffectiveCatalogPath()

				mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
				books, err := mgr.Load()
				if err != nil {
					warn("Could not load catalog for shelf %q: %v", shelf.Name, err)
					continue
				}

				renamed := 0
				for j := range books {
					for k, t := range books[j].Tags {
						if strings.EqualFold(t, oldTag) {
							if dryRun {
								fmt.Printf("  %s: %q → %q\n", books[j].ID, t, newTag)
							}
							books[j].Tags[k] = newTag
							renamed++
						}
					}
				}

				if renamed == 0 {
					continue
				}

				if !dryRun {
					commitMsg := fmt.Sprintf("tags: rename %q → %q (%d books)", oldTag, newTag, renamed)
					if err := mgr.Save(books, commitMsg); err != nil {
						warn("Could not save catalog for shelf %q: %v", shelf.Name, err)
						continue
					}
					ok("Shelf %s: renamed %d occurrences", shelf.Name, renamed)
				} else {
					fmt.Printf("  Shelf %s: %d books would be updated\n", shelf.Name, renamed)
				}
				totalRenamed += renamed
			}

			if totalRenamed == 0 {
				fmt.Printf("Tag %q not found in any book.\n", oldTag)
			} else if dryRun {
				fmt.Printf("\n%d total occurrences (dry run — no changes made)\n", totalRenamed)
			} else {
				fmt.Printf("\n%d total occurrences renamed\n", totalRenamed)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Rename only within a specific shelf")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be renamed without saving")

	return cmd
}

func collectTagCounts(shelves []config.ShelfConfig) map[string]int {
	counts := make(map[string]int)

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

		for _, b := range books {
			for _, t := range b.Tags {
				counts[strings.ToLower(t)]++
			}
		}
	}

	return counts
}
