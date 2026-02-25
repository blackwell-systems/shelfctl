package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	var flagOpen bool

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Generate HTML index for browsing your library",
		Long: `Generate an index.html file in your cache directory that displays all books
in a visual grid layout with covers. Cached books are clickable; uncached books
appear greyed out with a hint to download them. Open the index in any web browser
to browse your library without running shelfctl.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(cfg.Shelves) == 0 {
				warn("No shelves configured.")
				return nil
			}

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

				for _, b := range books {
					isCached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)
					var filePath string
					if isCached {
						filePath = cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
					}
					coverPath := cacheMgr.GetCoverPath(shelf.Repo, b.ID)

					indexBooks = append(indexBooks, cache.IndexBook{
						Book:      b,
						ShelfName: shelf.Name,
						Repo:      shelf.Repo,
						FilePath:  filePath,
						CoverPath: coverPath,
						HasCover:  coverPath != "",
						IsCached:  isCached,
					})
				}
			}

			if len(indexBooks) == 0 {
				warn("No books found in any shelf.")
				return nil
			}

			// Generate index
			if err := cacheMgr.GenerateHTMLIndex(indexBooks); err != nil {
				return fmt.Errorf("generating index: %w", err)
			}

			cachedCount := 0
			for _, b := range indexBooks {
				if b.IsCached {
					cachedCount++
				}
			}

			indexPath := filepath.Join(cfg.Defaults.CacheDir, "index.html")
			if cachedCount == len(indexBooks) {
				ok("Generated HTML index with %d books", len(indexBooks))
			} else {
				ok("Generated HTML index with %d books (%d cached)", len(indexBooks), cachedCount)
			}

			if flagOpen {
				var openCmd *exec.Cmd
				switch runtime.GOOS {
				case "darwin":
					openCmd = exec.Command("open", indexPath)
				case "windows":
					openCmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", indexPath)
				default:
					openCmd = exec.Command("xdg-open", indexPath)
				}
				if err := openCmd.Start(); err != nil {
					warn("Could not open browser: %v", err)
					fmt.Printf("\nOpen in browser:\n  file://%s\n", indexPath)
				}
			} else {
				fmt.Printf("\nOpen in browser:\n  file://%s\n", indexPath)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flagOpen, "open", false, "Open the generated index in the default browser")

	return cmd
}
