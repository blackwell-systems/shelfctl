package app

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var (
		shelfName  string
		releaseTag string
		dryRun     bool
		maxN       int
		noPush     bool
	)

	cmd := &cobra.Command{
		Use:   "import --from <owner/repo>",
		Short: "Import books from another shelfctl shelf",
		Long: `Reads the catalog.yml from a source shelf repo and re-uploads each asset
to your local shelf. Useful for absorbing another user's shelf or a second
account's shelf into your own.

Skips duplicates by sha256.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fromFlag, _ := cmd.Flags().GetString("from")
			if fromFlag == "" {
				return fmt.Errorf("--from owner/repo is required")
			}

			// Parse "owner/repo".
			var srcOwner, srcRepo string
			if _, err := fmt.Sscanf(fromFlag, "%s", &fromFlag); err != nil {
				return err
			}
			parts := splitOwnerRepo(fromFlag)
			if parts == nil {
				return fmt.Errorf("--from must be owner/repo")
			}
			srcOwner, srcRepo = parts[0], parts[1]

			shelf := cfg.ShelfByName(shelfName)
			if shelf == nil {
				return fmt.Errorf("shelf %q not found in config", shelfName)
			}
			dstOwner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			if releaseTag == "" {
				releaseTag = shelf.EffectiveRelease(cfg.Defaults.Release)
			}

			// Load source catalog.
			srcCatalogData, _, err := gh.GetFileContent(srcOwner, srcRepo, "catalog.yml", "")
			if err != nil {
				return fmt.Errorf("reading source catalog: %w", err)
			}
			srcBooks, err := catalog.Parse(srcCatalogData)
			if err != nil {
				return err
			}

			// Load destination catalog.
			dstCatalogPath := shelf.EffectiveCatalogPath()
			dstData, _, _ := gh.GetFileContent(dstOwner, shelf.Repo, dstCatalogPath, "")
			dstBooks, _ := catalog.Parse(dstData)

			// Build sha256 index of existing books to detect duplicates.
			existingSHAs := map[string]bool{}
			for _, b := range dstBooks {
				if b.Checksum.SHA256 != "" {
					existingSHAs[b.Checksum.SHA256] = true
				}
			}

			dstRel, err := gh.EnsureRelease(dstOwner, shelf.Repo, releaseTag)
			if err != nil {
				return err
			}

			imported := 0
			skipped := 0
			for i := range srcBooks {
				if maxN > 0 && imported >= maxN {
					fmt.Printf("Limit of %d reached.\n", maxN)
					break
				}

				b := &srcBooks[i]

				if existingSHAs[b.Checksum.SHA256] {
					fmt.Printf("  skip (duplicate sha256): %s\n", b.ID)
					skipped++
					continue
				}

				if dryRun {
					fmt.Printf("  would import: %s — %s\n", b.ID, b.Title)
					imported++
					continue
				}

				// Find the source release asset.
				srcRel, err := gh.GetReleaseByTag(b.Source.Owner, b.Source.Repo, b.Source.Release)
				if err != nil {
					warn("Skipping %s: release %s not found: %v", b.ID, b.Source.Release, err)
					skipped++
					continue
				}
				srcAsset, err := gh.FindAsset(b.Source.Owner, b.Source.Repo, srcRel.ID, b.Source.Asset)
				if err != nil || srcAsset == nil {
					warn("Skipping %s: asset not found", b.ID)
					skipped++
					continue
				}

				fmt.Printf("  importing %s — %s …\n", b.ID, b.Title)

				rc, err := gh.DownloadAsset(b.Source.Owner, b.Source.Repo, srcAsset.ID)
				if err != nil {
					warn("Download failed for %s: %v", b.ID, err)
					skipped++
					continue
				}

				// Buffer to temp file.
				tmp, err := os.CreateTemp("", "shelfctl-import-*")
				if err != nil {
					_ = rc.Close()
					return err
				}
				tmpPath := tmp.Name()

				hr := ingest.NewReader(rc)
				if _, err := io.Copy(tmp, hr); err != nil {
					_ = tmp.Close()
					_ = rc.Close()
					_ = os.Remove(tmpPath)
					warn("Buffer failed for %s: %v", b.ID, err)
					skipped++
					continue
				}
				_ = tmp.Close()
				_ = rc.Close()

				fi, err := os.Stat(tmpPath)
				if err != nil {
					_ = os.Remove(tmpPath)
					warn("Stat failed for %s: %v", b.ID, err)
					skipped++
					continue
				}
				uploadFile, err := os.Open(tmpPath)
				if err != nil {
					_ = os.Remove(tmpPath)
					warn("Open failed for %s: %v", b.ID, err)
					skipped++
					continue
				}
				_, err = gh.UploadAsset(dstOwner, shelf.Repo, dstRel.ID, b.Source.Asset,
					uploadFile, fi.Size(), "application/octet-stream")
				_ = uploadFile.Close()
				_ = os.Remove(tmpPath)
				if err != nil {
					warn("Upload failed for %s: %v", b.ID, err)
					skipped++
					continue
				}

				// Build new entry for destination.
				newBook := *b
				newBook.Source = catalog.Source{
					Type:    "github_release",
					Owner:   dstOwner,
					Repo:    shelf.Repo,
					Release: releaseTag,
					Asset:   b.Source.Asset,
				}
				newBook.Checksum.SHA256 = hr.SHA256()
				newBook.SizeBytes = hr.Size()
				newBook.Meta.AddedAt = time.Now().UTC().Format(time.RFC3339)
				newBook.Meta.MigratedFrom = fmt.Sprintf("%s/%s", srcOwner, srcRepo)

				dstBooks = catalog.Append(dstBooks, newBook)
				existingSHAs[newBook.Checksum.SHA256] = true
				imported++
				ok("Imported: %s", b.ID)
			}

			if dryRun {
				fmt.Printf("\n(dry run) would import=%d skipped=%d\n", imported, skipped)
				return nil
			}

			if imported == 0 {
				fmt.Printf("Nothing new to import (skipped=%d).\n", skipped)
				return nil
			}

			newData, err := catalog.Marshal(dstBooks)
			if err != nil {
				return err
			}
			if !noPush {
				msg := fmt.Sprintf("import: %d books from %s/%s", imported, srcOwner, srcRepo)
				if err := gh.CommitFile(dstOwner, shelf.Repo, dstCatalogPath, newData, msg); err != nil {
					return err
				}
				ok("Catalog committed (%d imported, %d skipped)", imported, skipped)
			} else {
				ok("Done (not pushed): imported=%d skipped=%d", imported, skipped)
			}
			return nil
		},
	}

	cmd.Flags().String("from", "", "Source shelf as owner/repo (required)")
	cmd.Flags().StringVar(&shelfName, "shelf", "", "Destination shelf name (required)")
	cmd.Flags().StringVar(&releaseTag, "release", "", "Destination release (default: shelf's default_release)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be imported without doing it")
	cmd.Flags().IntVarP(&maxN, "n", "n", 0, "Limit per run (0 = unlimited)")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Update catalog locally only")

	_ = cmd.MarkFlagRequired("shelf")
	return cmd
}

func splitOwnerRepo(s string) []string {
	for i, c := range s {
		if c == '/' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
