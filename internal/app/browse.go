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

						// Download catalog cover if specified and not already cached
						if b.Cover != "" && !cacheMgr.HasCatalogCover(shelf.Repo, b.ID) {
							if coverData, _, err := gh.GetFileContent(owner, shelf.Repo, b.Cover, ""); err == nil {
								_ = cacheMgr.StoreCatalogCover(shelf.Repo, b.ID, strings.NewReader(string(coverData)))
							}
						}

						// Get best available cover (catalog > extracted > none)
						coverPath := cacheMgr.GetCoverPath(shelf.Repo, b.ID)
						hasCover := coverPath != ""

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
				if result.Action != tui.ActionNone && (result.BookItem != nil || len(result.BookItems) > 0) {
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
	switch result.Action {
	case tui.ActionDownload:
		// Check for multi-select download
		if len(result.BookItems) > 0 {
			return handleMultiDownload(cmd, result.BookItems)
		}
		// Fall through to single book handling

	}

	// Single book actions require BookItem to be set
	if result.BookItem == nil {
		return fmt.Errorf("no book selected")
	}

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
			progressCh := make(chan int64, 50)
			errCh := make(chan error, 1)

			// Show connecting message
			fmt.Printf("Connecting to GitHub...\n")

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
			if err := tui.ShowProgress(label, asset.Size, progressCh); err != nil {
				return err // User cancelled
			}

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

			// Show poppler hint if needed (one-time, PDF only)
			showPopplerHintIfNeeded(b.Source.Asset)
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
				progressCh := make(chan int64, 50)
				errCh := make(chan error, 1)

				// Show connecting message
				fmt.Printf("Connecting to GitHub...\n")

				// Start download in goroutine
				go func() {
					pr := tui.NewProgressReader(rc, asset.Size, progressCh)
					_, err := cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, pr, b.Checksum.SHA256)
					close(progressCh)
					errCh <- err
				}()

				// Show progress UI
				label := fmt.Sprintf("Downloading %s (%s)", b.ID, humanBytes(asset.Size))
				if err := tui.ShowProgress(label, asset.Size, progressCh); err != nil {
					return err // User cancelled
				}

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

			// Show poppler hint if needed (one-time, PDF only)
			showPopplerHintIfNeeded(b.Source.Asset)
		}

		// Open the file
		path := cacheMgr.Path(item.Owner, item.Repo, b.ID, b.Source.Asset)
		return openFile(path, "")

	case tui.ActionEdit:
		// Edit book metadata
		shelf := cfg.ShelfByName(item.ShelfName)
		if shelf == nil {
			return fmt.Errorf("shelf %q not found", item.ShelfName)
		}
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()

		// Show edit form
		defaults := tui.EditFormDefaults{
			BookID: b.ID,
			Title:  b.Title,
			Author: b.Author,
			Year:   b.Year,
			Tags:   b.Tags,
		}

		formData, err := tui.RunEditForm(defaults)
		if err != nil {
			return err
		}

		// Parse tags
		tags := []string{}
		if formData.Tags != "" {
			for _, t := range strings.Split(formData.Tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		// Build updated book
		updatedBook := *b
		updatedBook.Title = formData.Title
		updatedBook.Author = formData.Author
		updatedBook.Year = formData.Year
		updatedBook.Tags = tags

		// Load catalog
		data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
		if err != nil {
			return fmt.Errorf("loading catalog: %w", err)
		}
		books, err := catalog.Parse(data)
		if err != nil {
			return fmt.Errorf("parsing catalog: %w", err)
		}

		// Update book in catalog
		books = catalog.Append(books, updatedBook)

		// Commit catalog
		updatedData, err := catalog.Marshal(books)
		if err != nil {
			return fmt.Errorf("marshaling catalog: %w", err)
		}
		commitMsg := fmt.Sprintf("edit: update %s metadata", b.ID)
		if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
			return fmt.Errorf("committing catalog: %w", err)
		}

		// Update README with new metadata
		readmeData, _, readmeErr := gh.GetFileContent(owner, shelf.Repo, "README.md", "")
		if readmeErr == nil {
			originalContent := string(readmeData)
			readmeContent := updateShelfREADMEStats(originalContent, len(books))
			readmeContent = appendToShelfREADME(readmeContent, updatedBook)

			// Only commit if content actually changed
			if readmeContent != originalContent {
				readmeMsg := fmt.Sprintf("Update README: edit %s", b.ID)
				if err := gh.CommitFile(owner, shelf.Repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
					warn("Could not update README.md: %v", err)
				} else {
					ok("README.md updated")
				}
			}
		}

		ok("Book successfully updated: %s", b.ID)
		return nil

	default:
		return nil
	}
}

// handleMultiDownload downloads multiple books to cache
func handleMultiDownload(cmd *cobra.Command, items []tui.BookItem) error {
	fmt.Printf("Downloading %d books...\n\n", len(items))

	successCount := 0
	failCount := 0

	for i, item := range items {
		b := &item.Book
		fmt.Printf("[%d/%d] %s\n", i+1, len(items), b.ID)

		// Skip if already cached
		if item.Cached {
			fmt.Println(color.GreenString("  ✓ Already cached"))
			successCount++
			continue
		}

		// Get release and asset
		rel, err := gh.GetReleaseByTag(item.Owner, item.Repo, b.Source.Release)
		if err != nil {
			warn("  Failed to get release: %v", err)
			failCount++
			continue
		}

		asset, err := gh.FindAsset(item.Owner, item.Repo, rel.ID, b.Source.Asset)
		if err != nil || asset == nil {
			warn("  Failed to find asset: %v", err)
			failCount++
			continue
		}

		rc, err := gh.DownloadAsset(item.Owner, item.Repo, asset.ID)
		if err != nil {
			warn("  Failed to download: %v", err)
			failCount++
			continue
		}

		// Store in cache
		path, err := cacheMgr.Store(item.Owner, item.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
		_ = rc.Close()
		if err != nil {
			warn("  Failed to cache: %v", err)
			failCount++
			continue
		}

		fmt.Println(color.GreenString("  ✓ Cached: ") + path)
		successCount++
	}

	// Summary
	fmt.Println()
	if successCount > 0 {
		ok("Successfully downloaded %d books", successCount)
	}
	if failCount > 0 {
		warn("%d books failed to download", failCount)
	}

	return nil
}
