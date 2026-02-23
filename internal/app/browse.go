package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// browserDownloader implements tui.Downloader for background downloads
type browserDownloader struct {
	gh    *github.Client
	cache *cache.Manager
}

func (d *browserDownloader) Download(owner, repo, bookID, release, asset, sha256 string) (bool, error) {
	return d.DownloadWithProgress(owner, repo, bookID, release, asset, sha256, nil) == nil, nil
}

func (d *browserDownloader) DownloadWithProgress(owner, repo, bookID, release, asset, sha256 string, progressCh chan<- float64) error {
	// Get release
	rel, err := d.gh.GetReleaseByTag(owner, repo, release)
	if err != nil {
		return fmt.Errorf("release %q: %w", release, err)
	}

	// Find asset
	assetObj, err := d.gh.FindAsset(owner, repo, rel.ID, asset)
	if err != nil {
		return fmt.Errorf("finding asset: %w", err)
	}
	if assetObj == nil {
		return fmt.Errorf("asset %q not found", asset)
	}

	// Download
	rc, err := d.gh.DownloadAsset(owner, repo, assetObj.ID)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Wrap with progress tracking if channel provided
	var reader io.Reader = rc
	if progressCh != nil {
		reader = &progressReader{
			reader:     rc,
			total:      assetObj.Size,
			progressCh: progressCh,
		}
	}

	// Store in cache
	_, err = d.cache.Store(owner, repo, bookID, asset, reader, sha256)
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}

	return nil
}

func (d *browserDownloader) Uncache(owner, repo, bookID, asset string) error {
	return d.cache.Remove(owner, repo, bookID, asset)
}

func (d *browserDownloader) Sync(owner, repo, bookID, release, asset, catalogPath, catalogSHA256 string) (bool, error) {
	// Check if cached and modified
	if !d.cache.Exists(owner, repo, bookID, asset) {
		return false, fmt.Errorf("not cached")
	}

	if !d.cache.HasBeenModified(owner, repo, bookID, asset, catalogSHA256) {
		return false, nil // No changes
	}

	// Get cached file path, hash, and size
	cachedPath := d.cache.Path(owner, repo, bookID, asset)
	cachedSHA, cachedSize, err := computeFileHash(cachedPath)
	if err != nil {
		return false, fmt.Errorf("computing hash: %w", err)
	}

	// Get release
	rel, err := d.gh.GetReleaseByTag(owner, repo, release)
	if err != nil {
		return false, fmt.Errorf("release %q: %w", release, err)
	}

	// Find and delete old asset
	oldAsset, err := d.gh.FindAsset(owner, repo, rel.ID, asset)
	if err != nil {
		return false, fmt.Errorf("finding asset: %w", err)
	}
	if oldAsset != nil {
		if err := d.gh.DeleteAsset(owner, repo, oldAsset.ID); err != nil {
			return false, fmt.Errorf("deleting old asset: %w", err)
		}
	}

	// Upload modified file
	f, err := os.Open(cachedPath)
	if err != nil {
		return false, fmt.Errorf("opening cached file: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = d.gh.UploadAsset(owner, repo, rel.ID, asset, f, cachedSize, "application/octet-stream")
	if err != nil {
		return false, fmt.Errorf("uploading: %w", err)
	}

	// Update catalog with new SHA256
	mgr := catalog.NewManager(d.gh, owner, repo, catalogPath)
	books, err := mgr.Load()
	if err != nil {
		return false, fmt.Errorf("loading catalog: %w", err)
	}

	// Find and update the book
	bookToUpdate := catalog.ByID(books, bookID)
	if bookToUpdate != nil {
		bookToUpdate.Checksum.SHA256 = cachedSHA
		bookToUpdate.SizeBytes = cachedSize

		commitMsg := fmt.Sprintf("sync: update %s with local changes", bookID)
		if err := mgr.Save(books, commitMsg); err != nil {
			return false, fmt.Errorf("saving catalog: %w", err)
		}
	}

	return true, nil
}

func (d *browserDownloader) HasBeenModified(owner, repo, bookID, asset, catalogSHA256 string) bool {
	return d.cache.HasBeenModified(owner, repo, bookID, asset, catalogSHA256)
}

// progressReader wraps io.Reader to send progress updates
type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	progressCh chan<- float64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	if pr.progressCh != nil && pr.total > 0 {
		pct := float64(pr.read) / float64(pr.total)
		select {
		case pr.progressCh <- pct:
		default:
		}
	}

	return n, err
}

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
					releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

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
							Book:        b,
							ShelfName:   shelf.Name,
							Cached:      cached,
							HasCover:    hasCover,
							CoverPath:   coverPath,
							Owner:       owner,
							Repo:        shelf.Repo,
							Release:     releaseTag,
							CatalogPath: catalogPath,
						})
					}
				}

				if len(allItems) == 0 {
					warn("No books found.")
					return nil
				}

				// Create downloader for background downloads
				dl := &browserDownloader{
					gh:    gh,
					cache: cacheMgr,
				}

				result, err := tui.RunListBrowser(allItems, dl)
				if err != nil {
					return err
				}

				// Handle browser actions
				// Downloads are handled in background by TUI, other actions exit TUI
				if result.Action != tui.ActionNone {
					if result.BookItem != nil {
						return handleBrowserAction(cmd, result)
					}
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
