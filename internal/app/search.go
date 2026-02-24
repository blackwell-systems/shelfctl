package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type searchResult struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Author    string   `json:"author,omitempty"`
	Format    string   `json:"format"`
	Tags      []string `json:"tags,omitempty"`
	Shelf     string   `json:"shelf"`
	Cached    bool     `json:"cached"`
	SizeBytes int64    `json:"size_bytes,omitempty"`
}

func newSearchCmd() *cobra.Command {
	var (
		shelfName string
		tag       string
		format    string
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search books by title, author, or tags",
		Long: `Search across all shelves for books matching a query string.
Searches title, author, and tags (case-insensitive).

Use --tag or --format to narrow results further.

Examples:
  shelfctl search "neural networks"
  shelfctl search golang --tag programming
  shelfctl search --tag fiction --shelf books
  shelfctl search "smith" --format epub --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			if query == "" && tag == "" && format == "" {
				return fmt.Errorf("provide a search query or use --tag/--format to filter")
			}

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
				warn("No shelves configured")
				return nil
			}

			f := catalog.Filter{Tag: tag, Search: query, Format: format}
			var results []searchResult
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

				if !jsonOut {
					header("── %s  (%d matches)", shelf.Name, len(matched))
				}

				for _, b := range matched {
					cached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)

					if jsonOut {
						results = append(results, searchResult{
							ID:        b.ID,
							Title:     b.Title,
							Author:    b.Author,
							Format:    b.Format,
							Tags:      b.Tags,
							Shelf:     shelf.Name,
							Cached:    cached,
							SizeBytes: b.SizeBytes,
						})
					} else {
						cachedMark := ""
						if cached {
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
				}
				total += len(matched)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			if total == 0 {
				fmt.Println("No books found.")
			} else {
				fmt.Printf("\n%d result(s)\n", total)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Search within a specific shelf")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&format, "format", "", "Filter by format (pdf, epub, ...)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}
