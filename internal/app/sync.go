package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var (
		shelfName string
		all       bool
	)

	cmd := &cobra.Command{
		Use:   "sync [book-id...]",
		Short: "Sync locally modified books back to GitHub",
		Long: `Detect books with local modifications (annotations, highlights) and re-upload to GitHub.

The cached file's SHA256 is compared with the catalog. If different, the Release asset
is replaced and the catalog is updated. This keeps GitHub in sync with your working copies
without creating multiple versions.

Examples:
  shelfctl sync book-id           # Sync specific book
  shelfctl sync book-1 book-2     # Sync multiple books
  shelfctl sync --all             # Sync all modified books
  shelfctl sync --all --shelf prog # Sync all modified in specific shelf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !all && len(args) == 0 {
				return fmt.Errorf("provide book IDs or use --all")
			}
			return runSync(cmd, args, shelfName, all)
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Limit to specific shelf")
	cmd.Flags().BoolVar(&all, "all", false, "Sync all modified cached books")

	return cmd
}

func runSync(cmd *cobra.Command, bookIDs []string, shelfName string, all bool) error {
	// Collect shelves to process
	shelves := cfg.Shelves
	if shelfName != "" {
		shelf := cfg.ShelfByName(shelfName)
		if shelf == nil {
			return fmt.Errorf("shelf %q not found", shelfName)
		}
		shelves = []config.ShelfConfig{*shelf}
	}

	// First pass: count how many books need syncing
	booksToSync := []struct {
		shelf  *config.ShelfConfig
		book   *catalog.Book
		path   string
		size   int64
		newSHA string
	}{}

	for i := range shelves {
		shelf := &shelves[i]
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()

		mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
		books, err := mgr.Load()
		if err != nil {
			warn("Could not load catalog for shelf %s: %v", shelf.Name, err)
			continue
		}

		for j := range books {
			b := &books[j]

			// Skip if not in our target list (when not --all)
			if !all && !contains(bookIDs, b.ID) {
				continue
			}

			// Skip if not cached
			if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				if !all {
					warn("Book %s not cached locally", b.ID)
				}
				continue
			}

			// Check if modified
			cachedPath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			cachedSHA, cachedSize, err := computeFileHash(cachedPath)
			if err != nil {
				warn("Could not read cached file for %s: %v", b.ID, err)
				continue
			}

			if cachedSHA != b.Checksum.SHA256 {
				booksToSync = append(booksToSync, struct {
					shelf  *config.ShelfConfig
					book   *catalog.Book
					path   string
					size   int64
					newSHA string
				}{
					shelf:  shelf,
					book:   b,
					path:   cachedPath,
					size:   cachedSize,
					newSHA: cachedSHA,
				})
			} else if !all {
				// Only print for explicitly requested books
				if tui.ShouldUseTUI(cmd) {
					fmt.Printf("✓ %s: no changes\n", b.ID)
				}
			}
		}
	}

	// If nothing to sync, exit early
	if len(booksToSync) == 0 {
		if all {
			fmt.Println("No modified books found in cache")
		}
		return nil
	}

	// Second pass: sync books with progress indicators
	// Cache catalogs and managers per shelf to avoid redundant API calls
	type shelfState struct {
		mgr   *catalog.Manager
		books []catalog.Book
	}
	shelfCache := make(map[string]*shelfState)

	totalSynced := 0
	totalErrors := 0

	for idx, item := range booksToSync {
		shelf := item.shelf
		b := item.book
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()
		releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

		// Show progress counter
		progressPrefix := fmt.Sprintf("[%d/%d]", idx+1, len(booksToSync))
		if tui.ShouldUseTUI(cmd) {
			fmt.Printf("%s Syncing %s (%s)\n", progressPrefix, b.ID, b.Title)
		}

		// Load catalog (cached per shelf)
		cacheKey := owner + "/" + shelf.Repo
		state, cached := shelfCache[cacheKey]
		if !cached {
			mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
			books, err := mgr.Load()
			if err != nil {
				warn("Could not load catalog for shelf %s: %v", shelf.Name, err)
				totalErrors++
				continue
			}
			state = &shelfState{mgr: mgr, books: books}
			shelfCache[cacheKey] = state
		}

		// Get release
		rel, err := gh.EnsureRelease(owner, shelf.Repo, releaseTag)
		if err != nil {
			warn("Could not get release for shelf %s: %v", shelf.Name, err)
			totalErrors++
			continue
		}

		// Find and delete old asset
		oldAsset, err := gh.FindAsset(owner, shelf.Repo, rel.ID, b.Source.Asset)
		if err != nil {
			warn("Could not find asset for %s: %v", b.ID, err)
			totalErrors++
			continue
		}
		if oldAsset != nil {
			if err := gh.DeleteAsset(owner, shelf.Repo, oldAsset.ID); err != nil {
				warn("Could not delete old asset for %s: %v", b.ID, err)
				totalErrors++
				continue
			}
		}

		// Upload modified file with progress bar
		f, err := os.Open(item.path)
		if err != nil {
			warn("Could not open cached file for %s: %v", b.ID, err)
			totalErrors++
			continue
		}

		var uploadErr error
		if tui.ShouldUseTUI(cmd) {
			// Show progress bar during upload
			progressCh := make(chan int64, 100)
			errCh := make(chan error, 1)

			go func() {
				progressCh <- 0
				pr := tui.NewProgressReader(f, item.size, progressCh)
				_, err := gh.UploadAsset(owner, shelf.Repo, rel.ID, b.Source.Asset, pr, item.size, "application/octet-stream")
				close(progressCh)
				errCh <- err
			}()

			label := fmt.Sprintf("%s Uploading %s → %s/%s", progressPrefix, b.Source.Asset, owner, shelf.Repo)
			if err := tui.ShowProgress(label, item.size, progressCh); err != nil {
				_ = f.Close()
				return err // User cancelled
			}

			uploadErr = <-errCh
		} else {
			// Non-interactive: direct upload
			_, uploadErr = gh.UploadAsset(owner, shelf.Repo, rel.ID, b.Source.Asset, f, item.size, "application/octet-stream")
		}

		_ = f.Close()

		if uploadErr != nil {
			warn("Could not upload modified file for %s: %v", b.ID, uploadErr)
			totalErrors++
			continue
		}

		// Update catalog entry
		bookToUpdate := catalog.ByID(state.books, b.ID)
		if bookToUpdate != nil {
			bookToUpdate.Checksum.SHA256 = item.newSHA
			bookToUpdate.SizeBytes = item.size

			commitMsg := fmt.Sprintf("sync: update %s with local changes", b.ID)
			if err := state.mgr.Save(state.books, commitMsg); err != nil {
				warn("Could not save catalog for shelf %s: %v", shelf.Name, err)
				totalErrors++
				continue
			}

			totalSynced++
			if tui.ShouldUseTUI(cmd) {
				ok("%s Synced %s", progressPrefix, b.ID)
			}
		} else {
			warn("Could not find book %s in catalog after upload", b.ID)
			totalErrors++
		}
	}

	// Print summary
	if tui.ShouldUseTUI(cmd) {
		fmt.Println()
		if totalSynced > 0 {
			ok("Successfully synced %d books", totalSynced)
		}
		if totalErrors > 0 {
			warn("%d books failed to sync", totalErrors)
		}
	}

	return nil
}

func computeFileHash(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
