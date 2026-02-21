package app

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
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
						hasCover := cacheMgr.HasCover(shelf.Repo, b.ID)
						coverPath := cacheMgr.CoverPath(shelf.Repo, b.ID)
						allItems = append(allItems, tui.BookItem{
							Book:      b,
							ShelfName: shelf.Name,
							Cached:    cached,
							HasCover:  hasCover,
							CoverPath: coverPath,
							Owner:     owner,
							Repo:      shelf.Repo,
						})
					}
				}

				if len(allItems) == 0 {
					warn("No books found.")
					return nil
				}

				result, err := tui.RunListBrowser(allItems)
				if err != nil {
					return err
				}

				// Handle browser actions
				if result.Action != tui.ActionNone && result.BookItem != nil {
					return handleBrowserAction(cmd, result)
				}

				return nil
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

// handleBrowserAction executes the action requested from the book browser
func handleBrowserAction(cmd *cobra.Command, result *tui.BrowserResult) error {
	item := result.BookItem
	b := &item.Book

	switch result.Action {
	case tui.ActionShowDetails:
		// Show book details
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
		printField("shelf", item.ShelfName)
		printField("release", b.Source.Release)
		printField("asset", b.Source.Asset)
		if b.Meta.AddedAt != "" {
			printField("added_at", b.Meta.AddedAt)
		}
		if b.Meta.MigratedFrom != "" {
			printField("migrated_from", b.Meta.MigratedFrom)
		}

		cacheStatus := color.RedString("not cached")
		if item.Cached {
			path := cacheMgr.Path(item.Owner, item.Repo, b.ID, b.Source.Asset)
			cacheStatus = color.GreenString("cached") + "  " + path
		}
		printField("cache", cacheStatus)
		return nil

	case tui.ActionDownload:
		// Download to cache only (don't open)
		if item.Cached {
			fmt.Println(color.GreenString("✓") + " Already cached")
			path := cacheMgr.Path(item.Owner, item.Repo, b.ID, b.Source.Asset)
			fmt.Println("  " + path)
			return nil
		}

		header("Downloading %s", b.ID)

		// Get release and asset
		rel, err := gh.GetReleaseByTag(item.Owner, item.Repo, b.Source.Release)
		if err != nil {
			return fmt.Errorf("release %q: %w", b.Source.Release, err)
		}
		asset, err := gh.FindAsset(item.Owner, item.Repo, rel.ID, b.Source.Asset)
		if err != nil {
			return fmt.Errorf("finding asset: %w", err)
		}
		if asset == nil {
			return fmt.Errorf("asset %q not found in release %q", b.Source.Asset, b.Source.Release)
		}

		rc, err := gh.DownloadAsset(item.Owner, item.Repo, asset.ID)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		defer func() { _ = rc.Close() }()

		// Use progress bar in TTY mode
		var path string
		if util.IsTTY() && tui.ShouldUseTUI(cmd) {
			progressCh := make(chan int64, 10)
			errCh := make(chan error, 1)

			// Start download in goroutine
			go func() {
				pr := tui.NewProgressReader(rc, asset.Size, progressCh)
				p, err := cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, pr, b.Checksum.SHA256)
				close(progressCh)
				errCh <- err
				if err == nil {
					// Store path for later
					path = p
				}
			}()

			// Show progress UI
			label := fmt.Sprintf("Downloading %s (%s)", b.ID, humanBytes(asset.Size))
			_ = tui.ShowProgress(label, asset.Size, progressCh)

			// Get result
			if err := <-errCh; err != nil {
				return fmt.Errorf("cache: %w", err)
			}
		} else {
			// Non-interactive mode: just print and download
			fmt.Printf("Downloading %s (%s) …\n", b.ID, humanBytes(asset.Size))
			path, err = cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
			if err != nil {
				return fmt.Errorf("cache: %w", err)
			}
		}
		ok("Cached: %s", path)
		return nil

	case tui.ActionOpen:
		// Download if needed, then open
		if !item.Cached {
			// Get release and asset
			rel, err := gh.GetReleaseByTag(item.Owner, item.Repo, b.Source.Release)
			if err != nil {
				return fmt.Errorf("release %q: %w", b.Source.Release, err)
			}
			asset, err := gh.FindAsset(item.Owner, item.Repo, rel.ID, b.Source.Asset)
			if err != nil {
				return fmt.Errorf("finding asset: %w", err)
			}
			if asset == nil {
				return fmt.Errorf("asset %q not found in release %q", b.Source.Asset, b.Source.Release)
			}

			rc, err := gh.DownloadAsset(item.Owner, item.Repo, asset.ID)
			if err != nil {
				return fmt.Errorf("download: %w", err)
			}
			defer func() { _ = rc.Close() }()

			// Use progress bar in TTY mode
			if util.IsTTY() && tui.ShouldUseTUI(cmd) {
				progressCh := make(chan int64, 10)
				errCh := make(chan error, 1)

				// Start download in goroutine
				go func() {
					pr := tui.NewProgressReader(rc, asset.Size, progressCh)
					_, err := cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, pr, b.Checksum.SHA256)
					close(progressCh)
					errCh <- err
				}()

				// Show progress UI
				label := fmt.Sprintf("Downloading %s (%s)", b.ID, humanBytes(asset.Size))
				_ = tui.ShowProgress(label, asset.Size, progressCh)

				// Get result
				if err := <-errCh; err != nil {
					return fmt.Errorf("cache: %w", err)
				}
			} else {
				// Non-interactive mode: just print and download
				fmt.Printf("Downloading %s (%s) …\n", b.ID, humanBytes(asset.Size))
				_, err = cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
				if err != nil {
					return fmt.Errorf("cache: %w", err)
				}
			}
			ok("Cached")
		}

		// Open the file
		path := cacheMgr.Path(item.Owner, item.Repo, b.ID, b.Source.Asset)
		return openFile(path, "")

	default:
		return nil
	}
}
