package app

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var shelfName string

	cmd := &cobra.Command{
		Use:   "info <id>",
		Short: "Show metadata and cache status for a book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			b, shelf, err := findBook(id, shelfName)
			if err != nil {
				return err
			}

			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			cached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)

			header("Book: %s", b.ID)
			printField("title", b.Title)
			if b.Author != "" {
				printField("author", b.Author)
			}
			if b.Year != 0 {
				printField("year", fmt.Sprintf("%d", b.Year))
			}
			printField("format", b.Format)
			if len(b.Tags) > 0 {
				printField("tags", strings.Join(b.Tags, ", "))
			}
			if b.SizeBytes > 0 {
				printField("size", humanBytes(b.SizeBytes))
			}
			if b.Checksum.SHA256 != "" {
				printField("sha256", b.Checksum.SHA256)
			}
			printField("shelf", shelf.Name)
			printField("release", b.Source.Release)
			printField("asset", b.Source.Asset)
			if b.Meta.AddedAt != "" {
				printField("added_at", b.Meta.AddedAt)
			}
			if b.Meta.MigratedFrom != "" {
				printField("migrated_from", b.Meta.MigratedFrom)
			}

			cacheStatus := color.RedString("not cached")
			if cached {
				path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
				cacheStatus = color.GreenString("cached") + "  " + path
			}
			printField("cache", cacheStatus)
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Specify shelf if ID collides across shelves")
	return cmd
}

func printField(label, value string) {
	fmt.Printf("  %-14s %s\n", color.CyanString(label+":"), value)
}

// findBook searches all configured shelves (or a specific one) for a book by ID.
func findBook(id, shelfName string) (*catalog.Book, *config.ShelfConfig, error) {
	var shelves []config.ShelfConfig
	if shelfName != "" {
		s := cfg.ShelfByName(shelfName)
		if s == nil {
			return nil, nil, fmt.Errorf("shelf %q not found in config", shelfName)
		}
		shelves = []config.ShelfConfig{*s}
	} else {
		shelves = cfg.Shelves
	}

	var firstBook *catalog.Book
	var firstShelf *config.ShelfConfig
	var foundShelves []string

	for i := range shelves {
		shelf := &shelves[i]
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		data, _, err := gh.GetFileContent(owner, shelf.Repo, shelf.EffectiveCatalogPath(), "")
		if err != nil {
			continue
		}
		books, err := catalog.Parse(data)
		if err != nil {
			continue
		}
		if b := catalog.ByID(books, id); b != nil {
			foundShelves = append(foundShelves, shelf.Name)
			if firstBook == nil {
				firstBook = b
				firstShelf = shelf
			}
		}
	}

	if firstBook == nil {
		return nil, nil, fmt.Errorf("book %q not found in any shelf", id)
	}

	// Warn if book ID exists in multiple shelves (and user didn't specify --shelf)
	if len(foundShelves) > 1 && shelfName == "" {
		warn("Book ID %q found in multiple shelves:", id)
		for _, s := range foundShelves {
			if s == firstShelf.Name {
				fmt.Printf("  - %s (using this one)\n", color.YellowString(s))
			} else {
				fmt.Printf("  - %s\n", s)
			}
		}
		fmt.Printf("Use --shelf to specify which one to use.\n\n")
	}

	return firstBook, firstShelf, nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
