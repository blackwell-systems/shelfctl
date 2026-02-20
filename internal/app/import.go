package app

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
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
			srcOwner, srcRepo, err := parseOwnerRepo(fromFlag)
			if err != nil {
				return err
			}

			// Setup import configuration
			importCtx, err := setupImportContext(shelfName, releaseTag, srcOwner, srcRepo)
			if err != nil {
				return err
			}

			// Perform the import
			imported, skipped, err := performImport(importCtx, maxN, dryRun)
			if err != nil {
				return err
			}

			// Save results
			return saveImportResults(importCtx, imported, skipped, dryRun, noPush)
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

func parseOwnerRepo(fromFlag string) (string, string, error) {
	if _, err := fmt.Sscanf(fromFlag, "%s", &fromFlag); err != nil {
		return "", "", err
	}
	parts := splitOwnerRepo(fromFlag)
	if parts == nil {
		return "", "", fmt.Errorf("--from must be owner/repo")
	}
	return parts[0], parts[1], nil
}

type importContext struct {
	shelf        *config.ShelfConfig
	srcOwner     string
	srcRepo      string
	dstOwner     string
	releaseTag   string
	catalogPath  string
	srcBooks     []catalog.Book
	dstBooks     []catalog.Book
	existingSHAs map[string]bool
	dstRel       *ghclient.Release
}

func setupImportContext(shelfName, releaseTag, srcOwner, srcRepo string) (*importContext, error) {
	shelf := cfg.ShelfByName(shelfName)
	if shelf == nil {
		return nil, fmt.Errorf("shelf %q not found in config", shelfName)
	}
	dstOwner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	if releaseTag == "" {
		releaseTag = shelf.EffectiveRelease(cfg.Defaults.Release)
	}

	// Load source catalog.
	srcCatalogData, _, err := gh.GetFileContent(srcOwner, srcRepo, "catalog.yml", "")
	if err != nil {
		return nil, fmt.Errorf("reading source catalog: %w", err)
	}
	srcBooks, err := catalog.Parse(srcCatalogData)
	if err != nil {
		return nil, err
	}

	// Load destination catalog.
	catalogPath := shelf.EffectiveCatalogPath()
	dstData, _, _ := gh.GetFileContent(dstOwner, shelf.Repo, catalogPath, "")
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
		return nil, err
	}

	return &importContext{
		shelf:        shelf,
		srcOwner:     srcOwner,
		srcRepo:      srcRepo,
		dstOwner:     dstOwner,
		releaseTag:   releaseTag,
		catalogPath:  catalogPath,
		srcBooks:     srcBooks,
		dstBooks:     dstBooks,
		existingSHAs: existingSHAs,
		dstRel:       dstRel,
	}, nil
}

func performImport(ctx *importContext, maxN int, dryRun bool) (int, int, error) {
	imported := 0
	skipped := 0

	for i := range ctx.srcBooks {
		if maxN > 0 && imported >= maxN {
			fmt.Printf("Limit of %d reached.\n", maxN)
			break
		}

		b := &ctx.srcBooks[i]

		if ctx.existingSHAs[b.Checksum.SHA256] {
			fmt.Printf("  skip (duplicate sha256): %s\n", b.ID)
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("  would import: %s — %s\n", b.ID, b.Title)
			imported++
			continue
		}

		newBook, err := importSingleBook(ctx, b)
		if err != nil {
			warn("%v", err)
			skipped++
			continue
		}

		ctx.dstBooks = catalog.Append(ctx.dstBooks, *newBook)
		ctx.existingSHAs[newBook.Checksum.SHA256] = true
		imported++
		ok("Imported: %s", b.ID)
	}

	return imported, skipped, nil
}

func importSingleBook(ctx *importContext, b *catalog.Book) (*catalog.Book, error) {
	// Find the source release asset.
	srcRel, err := gh.GetReleaseByTag(b.Source.Owner, b.Source.Repo, b.Source.Release)
	if err != nil {
		return nil, fmt.Errorf("skipping %s: release %s not found: %v", b.ID, b.Source.Release, err)
	}
	srcAsset, err := gh.FindAsset(b.Source.Owner, b.Source.Repo, srcRel.ID, b.Source.Asset)
	if err != nil || srcAsset == nil {
		return nil, fmt.Errorf("skipping %s: asset not found", b.ID)
	}

	fmt.Printf("  importing %s — %s …\n", b.ID, b.Title)

	// Download and upload the asset
	hr, err := downloadAndUploadAsset(ctx, b, srcAsset.ID)
	if err != nil {
		return nil, err
	}

	// Build new entry for destination.
	newBook := *b
	newBook.Source = catalog.Source{
		Type:    "github_release",
		Owner:   ctx.dstOwner,
		Repo:    ctx.shelf.Repo,
		Release: ctx.releaseTag,
		Asset:   b.Source.Asset,
	}
	newBook.Checksum.SHA256 = hr.SHA256()
	newBook.SizeBytes = hr.Size()
	newBook.Meta.AddedAt = time.Now().UTC().Format(time.RFC3339)
	newBook.Meta.MigratedFrom = fmt.Sprintf("%s/%s", ctx.srcOwner, ctx.srcRepo)

	return &newBook, nil
}

func downloadAndUploadAsset(ctx *importContext, b *catalog.Book, assetID int64) (*ingest.Reader, error) {
	rc, err := gh.DownloadAsset(b.Source.Owner, b.Source.Repo, assetID)
	if err != nil {
		return nil, fmt.Errorf("download failed for %s: %v", b.ID, err)
	}

	// Buffer to temp file.
	tmp, err := os.CreateTemp("", "shelfctl-import-*")
	if err != nil {
		_ = rc.Close()
		return nil, err
	}
	tmpPath := tmp.Name()

	hr := ingest.NewReader(rc)
	if _, err := io.Copy(tmp, hr); err != nil {
		_ = tmp.Close()
		_ = rc.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("buffer failed for %s: %v", b.ID, err)
	}
	_ = tmp.Close()
	_ = rc.Close()

	if err := uploadTempFile(ctx, b, tmpPath); err != nil {
		return nil, err
	}

	return hr, nil
}

func uploadTempFile(ctx *importContext, b *catalog.Book, tmpPath string) error {
	fi, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("stat failed for %s: %v", b.ID, err)
	}

	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("open failed for %s: %v", b.ID, err)
	}

	_, err = gh.UploadAsset(ctx.dstOwner, ctx.shelf.Repo, ctx.dstRel.ID, b.Source.Asset,
		uploadFile, fi.Size(), "application/octet-stream")
	_ = uploadFile.Close()
	_ = os.Remove(tmpPath)

	if err != nil {
		return fmt.Errorf("upload failed for %s: %v", b.ID, err)
	}

	return nil
}

func saveImportResults(ctx *importContext, imported, skipped int, dryRun, noPush bool) error {
	if dryRun {
		fmt.Printf("\n(dry run) would import=%d skipped=%d\n", imported, skipped)
		return nil
	}

	if imported == 0 {
		fmt.Printf("Nothing new to import (skipped=%d).\n", skipped)
		return nil
	}

	newData, err := catalog.Marshal(ctx.dstBooks)
	if err != nil {
		return err
	}

	if !noPush {
		msg := fmt.Sprintf("import: %d books from %s/%s", imported, ctx.srcOwner, ctx.srcRepo)
		if err := gh.CommitFile(ctx.dstOwner, ctx.shelf.Repo, ctx.catalogPath, newData, msg); err != nil {
			return err
		}
		ok("Catalog committed (%d imported, %d skipped)", imported, skipped)
	} else {
		ok("Done (not pushed): imported=%d skipped=%d", imported, skipped)
	}
	return nil
}
