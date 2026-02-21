package app

import (
	"fmt"
	"path/filepath"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Generate local HTML index for browsing cached books",
		Long: `Generate an index.html file in your cache directory that displays all cached books
in a visual grid layout with covers. Open the index in any web browser to browse
your library without running shelfctl.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(cfg.Shelves) == 0 {
				warn("No shelves configured.")
				return nil
			}

			// Collect all cached books across shelves
			var indexBooks []cache.IndexBook

			for i := range cfg.Shelves {
				shelf := &cfg.Shelves[i]
				owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
				catalogPath := shelf.EffectiveCatalogPath()

				// Load catalog
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

				// Check which books are cached
				for _, b := range books {
					if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
						continue // Skip uncached books
					}

					filePath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
					coverPath := cacheMgr.GetCoverPath(shelf.Repo, b.ID)

					indexBooks = append(indexBooks, cache.IndexBook{
						Book:      b,
						ShelfName: shelf.Name,
						Repo:      shelf.Repo,
						FilePath:  filePath,
						CoverPath: coverPath,
						HasCover:  coverPath != "",
					})
				}
			}

			if len(indexBooks) == 0 {
				warn("No cached books found. Download some books first:")
				fmt.Println("  shelfctl open <book-id>")
				return nil
			}

			// Generate index
			if err := cacheMgr.GenerateHTMLIndex(indexBooks); err != nil {
				return fmt.Errorf("generating index: %w", err)
			}

			indexPath := filepath.Join(cfg.Defaults.CacheDir, "index.html")
			ok("Generated HTML index with %d books", len(indexBooks))
			fmt.Printf("\nOpen in browser:\n  file://%s\n", indexPath)

			return nil
		},
	}

	return cmd
}
