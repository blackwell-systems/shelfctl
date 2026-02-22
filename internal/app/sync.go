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

	totalSynced := 0
	totalSkipped := 0
	totalErrors := 0

	for _, shelf := range shelves {
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()
		releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

		// Load catalog
		mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
		books, err := mgr.Load()
		if err != nil {
			warn("Could not load catalog for shelf %s: %v", shelf.Name, err)
			continue
		}

		// Get release
		rel, err := gh.EnsureRelease(owner, shelf.Repo, releaseTag)
		if err != nil {
			warn("Could not get release for shelf %s: %v", shelf.Name, err)
			continue
		}

		// Process books
		modified := false
		for i := range books {
			b := &books[i]

			// Skip if not in our target list (when not --all)
			if !all && !contains(bookIDs, b.ID) {
				continue
			}

			// Skip if not cached
			if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				if !all {
					// Only warn for explicitly requested books
					warn("Book %s not cached locally", b.ID)
					totalSkipped++
				}
				continue
			}

			// Check if modified
			cachedPath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			cachedSHA, cachedSize, err := computeFileHash(cachedPath)
			if err != nil {
				warn("Could not read cached file for %s: %v", b.ID, err)
				totalErrors++
				continue
			}

			if cachedSHA == b.Checksum.SHA256 {
				// No changes
				if !all {
					// Only print for explicitly requested books
					if tui.ShouldUseTUI(cmd) {
						fmt.Printf("✓ %s: no changes\n", b.ID)
					}
				}
				totalSkipped++
				continue
			}

			// File has been modified - sync it
			if tui.ShouldUseTUI(cmd) {
				fmt.Printf("Syncing %s (%s) …\n", b.ID, b.Title)
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

			// Upload modified file
			f, err := os.Open(cachedPath)
			if err != nil {
				warn("Could not open cached file for %s: %v", b.ID, err)
				totalErrors++
				continue
			}

			_, err = gh.UploadAsset(owner, shelf.Repo, rel.ID, b.Source.Asset, f, cachedSize, "application/octet-stream")
			_ = f.Close()
			if err != nil {
				warn("Could not upload modified file for %s: %v", b.ID, err)
				totalErrors++
				continue
			}

			// Update catalog entry
			b.Checksum.SHA256 = cachedSHA
			b.SizeBytes = cachedSize

			modified = true
			totalSynced++

			if tui.ShouldUseTUI(cmd) {
				ok("Synced %s", b.ID)
			}
		}

		// Commit catalog if any books were synced
		if modified {
			var commitMsg string
			if totalSynced == 1 {
				// Find which book was synced for better message
				for i := range books {
					if contains(bookIDs, books[i].ID) || all {
						commitMsg = fmt.Sprintf("sync: update %s with local changes", books[i].ID)
						break
					}
				}
				if commitMsg == "" {
					commitMsg = "sync: update book with local changes"
				}
			} else {
				commitMsg = fmt.Sprintf("sync: update %d books with local changes", totalSynced)
			}

			if err := mgr.Save(books, commitMsg); err != nil {
				warn("Could not save catalog for shelf %s: %v", shelf.Name, err)
				continue
			}
		}
	}

	// Print summary
	if tui.ShouldUseTUI(cmd) {
		fmt.Println()
		if totalSynced > 0 {
			ok("Synced %d books", totalSynced)
		}
		if totalSkipped > 0 && !all {
			fmt.Printf("%d books had no changes\n", totalSkipped)
		}
		if totalErrors > 0 {
			warn("%d errors", totalErrors)
		}
		if totalSynced == 0 && totalErrors == 0 && totalSkipped == 0 && all {
			fmt.Println("No modified books found in cache")
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
