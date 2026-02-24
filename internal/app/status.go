package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type shelfSyncStatus struct {
	Name       string `json:"name"`
	Repo       string `json:"repo"`
	Owner      string `json:"owner"`
	Books      int    `json:"books"`
	Cached     int    `json:"cached"`
	Modified   int    `json:"modified"`
	CacheBytes int64  `json:"cache_bytes"`

	// per-book detail (verbose only, not in JSON summary)
	bookDetails []bookStatus
}

type bookStatus struct {
	ID       string
	Title    string
	Asset    string
	Cached   bool
	Modified bool
}

type statusOutput struct {
	Shelves         []shelfSyncStatus `json:"shelves"`
	TotalBooks      int               `json:"total_books"`
	TotalCached     int               `json:"total_cached"`
	TotalModified   int               `json:"total_modified"`
	TotalCacheBytes int64             `json:"total_cache_bytes"`
}

func newStatusCmd() *cobra.Command {
	var (
		shelfName string
		verbose   bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show library sync status and statistics",
		Long: `Show an overview of your library: book counts, cache status, and modified files.

By default, shows a per-shelf summary. Use --verbose to see per-book status lines.

Examples:
  shelfctl status                   Summary of all shelves
  shelfctl status --verbose         Per-book status lines
  shelfctl status --shelf books     Status for one shelf
  shelfctl status --json            Machine-readable JSON output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(cfg.Shelves) == 0 {
				warn("No shelves configured. Run: shelfctl init --repo shelf-<topic> --name <topic>")
				return nil
			}

			shelves := cfg.Shelves
			if shelfName != "" {
				s := cfg.ShelfByName(shelfName)
				if s == nil {
					return fmt.Errorf("shelf %q not found in config", shelfName)
				}
				shelves = []config.ShelfConfig{*s}
			}

			result := collectStatus(shelves)

			if jsonOut {
				return printStatusJSON(result)
			}
			printStatusText(result, verbose)
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Show status for a specific shelf")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show per-book status lines")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func collectStatus(shelves []config.ShelfConfig) statusOutput {
	var result statusOutput

	for i := range shelves {
		shelf := &shelves[i]
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()

		ss := shelfSyncStatus{
			Name:  shelf.Name,
			Repo:  shelf.Repo,
			Owner: owner,
		}

		mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
		books, err := mgr.Load()
		if err != nil {
			warn("Could not load catalog for shelf %q: %v", shelf.Name, err)
			result.Shelves = append(result.Shelves, ss)
			continue
		}

		ss.Books = len(books)

		for j := range books {
			b := &books[j]
			bs := bookStatus{
				ID:    b.ID,
				Title: b.Title,
				Asset: b.Source.Asset,
			}

			if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				bs.Cached = true
				ss.Cached++

				path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
				if info, err := os.Stat(path); err == nil {
					ss.CacheBytes += info.Size()
				}

				if cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
					bs.Modified = true
					ss.Modified++
				}
			}

			ss.bookDetails = append(ss.bookDetails, bs)
		}

		result.Shelves = append(result.Shelves, ss)
		result.TotalBooks += ss.Books
		result.TotalCached += ss.Cached
		result.TotalModified += ss.Modified
		result.TotalCacheBytes += ss.CacheBytes
	}

	return result
}

func printStatusText(result statusOutput, verbose bool) {
	for i, ss := range result.Shelves {
		if i > 0 {
			fmt.Println()
		}

		header("Shelf: %s (%s/%s)", ss.Name, ss.Owner, ss.Repo)

		if verbose {
			for _, bs := range ss.bookDetails {
				switch {
				case bs.Modified:
					fmt.Printf("  %s modified  %s\n", color.YellowString("✗"), bs.ID)
				case bs.Cached:
					fmt.Printf("  %s cached    %s\n", color.GreenString("✓"), bs.ID)
				default:
					fmt.Printf("  %s remote    %s\n", color.HiBlackString("·"), bs.ID)
				}
			}
		} else {
			summary := fmt.Sprintf("  %d books, %d cached", ss.Books, ss.Cached)
			if ss.Modified > 0 {
				summary += fmt.Sprintf(", %d modified", ss.Modified)
			}
			fmt.Println(summary)
		}
	}

	if len(result.Shelves) > 1 || verbose {
		fmt.Println()
		total := fmt.Sprintf("Total: %d books, %d cached (%s)", result.TotalBooks, result.TotalCached, humanBytes(result.TotalCacheBytes))
		if result.TotalModified > 0 {
			total += fmt.Sprintf(", %d modified", result.TotalModified)
		}
		fmt.Println(total)
	}

	if result.TotalModified > 0 {
		fmt.Printf("\n%s Run 'shelfctl sync' to upload local changes\n", color.CyanString("hint:"))
	}
}

func printStatusJSON(result statusOutput) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
