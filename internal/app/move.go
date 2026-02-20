package app

import (
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/spf13/cobra"
)

func newMoveCmd() *cobra.Command {
	var (
		shelfName   string
		toRelease   string
		toShelfName string
		dryRun      bool
		keepOld     bool
	)

	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a book to a different release or shelf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			if toRelease == "" && toShelfName == "" {
				return fmt.Errorf("one of --to-release or --to-shelf is required")
			}

			b, srcShelf, err := findBook(id, shelfName)
			if err != nil {
				return err
			}
			srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)

			dstOwner := srcOwner
			dstRepo := srcShelf.Repo
			dstRelease := toRelease

			if toShelfName != "" {
				dstShelf := cfg.ShelfByName(toShelfName)
				if dstShelf == nil {
					return fmt.Errorf("destination shelf %q not found in config", toShelfName)
				}
				dstOwner = dstShelf.EffectiveOwner(cfg.GitHub.Owner)
				dstRepo = dstShelf.Repo
				if toRelease == "" {
					dstRelease = dstShelf.EffectiveRelease(cfg.Defaults.Release)
				}
			} else if dstRelease == "" {
				return fmt.Errorf("--to-release is required when not using --to-shelf")
			}

			fmt.Printf("Moving %s: %s/%s@%s → %s/%s@%s\n",
				id, srcOwner, srcShelf.Repo, b.Source.Release,
				dstOwner, dstRepo, dstRelease)

			if dryRun {
				fmt.Println("(dry run — no changes made)")
				return nil
			}

			// Ensure destination release exists.
			dstRel, err := gh.EnsureRelease(dstOwner, dstRepo, dstRelease)
			if err != nil {
				return fmt.Errorf("ensuring destination release: %w", err)
			}

			// Get source asset.
			srcRel, err := gh.GetReleaseByTag(srcOwner, srcShelf.Repo, b.Source.Release)
			if err != nil {
				return err
			}
			srcAsset, err := gh.FindAsset(srcOwner, srcShelf.Repo, srcRel.ID, b.Source.Asset)
			if err != nil {
				return err
			}
			if srcAsset == nil {
				return fmt.Errorf("source asset %q not found", b.Source.Asset)
			}

			// Stream from old release to new.
			rc, err := gh.DownloadAsset(srcOwner, srcShelf.Repo, srcAsset.ID)
			if err != nil {
				return fmt.Errorf("downloading source: %w", err)
			}

			// Buffer to temp file (needed for Content-Length).
			tmp, err := os.CreateTemp("", "shelfctl-move-*")
			if err != nil {
				_ = rc.Close()
				return err
			}
			tmpPath := tmp.Name()
			defer func() { _ = os.Remove(tmpPath) }()

			if _, err := io.Copy(tmp, rc); err != nil {
				_ = tmp.Close()
				_ = rc.Close()
				return err
			}
			_ = tmp.Close()
			_ = rc.Close()


		fi, err := os.Stat(tmpPath)
		if err != nil {
			return err
		}
		uploadFile, err := os.Open(tmpPath)
		if err != nil {
			return err
		}
		defer func() { _ = uploadFile.Close() }()
			_, err = gh.UploadAsset(dstOwner, dstRepo, dstRel.ID, b.Source.Asset,
				uploadFile, fi.Size(), "application/octet-stream")
			if err != nil {
				return fmt.Errorf("uploading to destination: %w", err)
			}
			ok("Uploaded to %s/%s@%s", dstOwner, dstRepo, dstRelease)

			// Delete old asset (unless --keep-old).
			if !keepOld {
				if err := gh.DeleteAsset(srcOwner, srcShelf.Repo, srcAsset.ID); err != nil {
					warn("Could not delete old asset: %v", err)
				} else {
					ok("Deleted old asset from %s@%s", srcShelf.Repo, b.Source.Release)
				}
			}

			// Update catalog: load src catalog, update entry, push.
			srcCatalogPath := srcShelf.EffectiveCatalogPath()
			data, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
			if err != nil {
				return err
			}
			books, err := catalog.Parse(data)
			if err != nil {
				return err
			}

			if toShelfName != "" {
				// Remove from source catalog.
				books, _ = catalog.Remove(books, id)
				srcData, err := catalog.Marshal(books)
				if err != nil {
					return err
				}
				if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, srcData,
					fmt.Sprintf("move: remove %s (moved to %s)", id, toShelfName)); err != nil {
					return err
				}

				// Add to destination catalog.
				b.Source.Release = dstRelease
				b.Source.Owner = dstOwner
				b.Source.Repo = dstRepo

				dstShelf := cfg.ShelfByName(toShelfName)
				dstCatalogPath := dstShelf.EffectiveCatalogPath()
				dstData, _, _ := gh.GetFileContent(dstOwner, dstRepo, dstCatalogPath, "")
				dstBooks, _ := catalog.Parse(dstData)
				dstBooks = catalog.Append(dstBooks, *b)
				dstCatalogData, err := catalog.Marshal(dstBooks)
				if err != nil {
					return err
				}
				if err := gh.CommitFile(dstOwner, dstRepo, dstCatalogPath, dstCatalogData,
					fmt.Sprintf("move: add %s (from %s)", id, srcShelf.Name)); err != nil {
					return err
				}
			} else {
				// Same repo, update release field.
				for i := range books {
					if books[i].ID == id {
						books[i].Source.Release = dstRelease
						break
					}
				}
				newData, err := catalog.Marshal(books)
				if err != nil {
					return err
				}
				if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, newData,
					fmt.Sprintf("move: %s → release/%s", id, dstRelease)); err != nil {
					return err
				}
			}

			ok("Catalog updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Source shelf (if ID is ambiguous)")
	cmd.Flags().StringVar(&toRelease, "to-release", "", "Destination release tag (same repo)")
	cmd.Flags().StringVar(&toShelfName, "to-shelf", "", "Destination shelf name (different repo)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVar(&keepOld, "keep-old", false, "Do not delete the old asset after copy")
	return cmd
}
