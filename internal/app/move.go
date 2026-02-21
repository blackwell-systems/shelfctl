package app

import (
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/spf13/cobra"
)

type moveParams struct {
	shelfName   string
	toRelease   string
	toShelfName string
	dryRun      bool
	keepOld     bool
}

type moveDestination struct {
	owner   string
	repo    string
	release string
	shelf   *config.ShelfConfig
}

func newMoveCmd() *cobra.Command {
	params := &moveParams{}

	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a book to a different release or shelf",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(args[0], params)
		},
	}

	cmd.Flags().StringVar(&params.shelfName, "shelf", "", "Source shelf (if ID is ambiguous)")
	cmd.Flags().StringVar(&params.toRelease, "to-release", "", "Destination release tag (same repo)")
	cmd.Flags().StringVar(&params.toShelfName, "to-shelf", "", "Destination shelf name (different repo)")
	cmd.Flags().BoolVar(&params.dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVar(&params.keepOld, "keep-old", false, "Do not delete the old asset after copy")
	return cmd
}

func runMove(id string, params *moveParams) error {
	if params.toRelease == "" && params.toShelfName == "" {
		return fmt.Errorf("one of --to-release or --to-shelf is required")
	}

	// Find source book
	b, srcShelf, err := findBook(id, params.shelfName)
	if err != nil {
		return err
	}
	srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)

	// Determine destination
	dst, err := determineDestination(srcShelf, srcOwner, params)
	if err != nil {
		return err
	}

	fmt.Printf("Moving %s: %s/%s@%s → %s/%s@%s\n",
		id, srcOwner, srcShelf.Repo, b.Source.Release,
		dst.owner, dst.repo, dst.release)

	if params.dryRun {
		fmt.Println("(dry run — no changes made)")
		return nil
	}

	// Transfer asset
	if err := transferAsset(b, srcShelf, srcOwner, dst); err != nil {
		return err
	}

	// Delete old asset unless keeping
	if !params.keepOld {
		deleteOldAsset(srcOwner, srcShelf.Repo, b, srcShelf)
	}

	// Update catalogs
	if params.toShelfName != "" {
		return updateCatalogsForCrossShelfMove(id, b, srcShelf, srcOwner, dst)
	}
	return updateCatalogForSameShelfMove(id, b, srcShelf, srcOwner, dst)
}

func determineDestination(srcShelf *config.ShelfConfig, srcOwner string, params *moveParams) (*moveDestination, error) {
	dst := &moveDestination{
		owner:   srcOwner,
		repo:    srcShelf.Repo,
		release: params.toRelease,
	}

	if params.toShelfName != "" {
		dstShelf := cfg.ShelfByName(params.toShelfName)
		if dstShelf == nil {
			return nil, fmt.Errorf("destination shelf %q not found in config", params.toShelfName)
		}
		dst.owner = dstShelf.EffectiveOwner(cfg.GitHub.Owner)
		dst.repo = dstShelf.Repo
		dst.shelf = dstShelf
		if params.toRelease == "" {
			dst.release = dstShelf.EffectiveRelease(cfg.Defaults.Release)
		}
	} else if dst.release == "" {
		return nil, fmt.Errorf("--to-release is required when not using --to-shelf")
	}

	return dst, nil
}

func transferAsset(b *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	// Ensure destination release exists
	dstRel, err := gh.EnsureRelease(dst.owner, dst.repo, dst.release)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// Get source asset
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

	// Download and buffer
	tmpPath, size, err := downloadAndBuffer(srcOwner, srcShelf.Repo, srcAsset.ID)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	// Upload to destination
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(dst.owner, dst.repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading to destination: %w", err)
	}

	ok("Uploaded to %s/%s@%s", dst.owner, dst.repo, dst.release)
	return nil
}

func downloadAndBuffer(owner, repo string, assetID int64) (string, int64, error) {
	rc, err := gh.DownloadAsset(owner, repo, assetID)
	if err != nil {
		return "", 0, fmt.Errorf("downloading source: %w", err)
	}

	tmp, err := os.CreateTemp("", "shelfctl-move-*")
	if err != nil {
		_ = rc.Close()
		return "", 0, err
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, rc); err != nil {
		_ = tmp.Close()
		_ = rc.Close()
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	_ = tmp.Close()
	_ = rc.Close()

	fi, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	return tmpPath, fi.Size(), nil
}

func deleteOldAsset(srcOwner, srcRepo string, b *catalog.Book, srcShelf *config.ShelfConfig) {
	srcRel, err := gh.GetReleaseByTag(srcOwner, srcRepo, b.Source.Release)
	if err != nil {
		warn("Could not get source release: %v", err)
		return
	}

	srcAsset, err := gh.FindAsset(srcOwner, srcRepo, srcRel.ID, b.Source.Asset)
	if err != nil {
		warn("Could not find source asset: %v", err)
		return
	}

	if srcAsset == nil {
		warn("Source asset not found")
		return
	}

	if err := gh.DeleteAsset(srcOwner, srcRepo, srcAsset.ID); err != nil {
		warn("Could not delete old asset: %v", err)
	} else {
		ok("Deleted old asset from %s@%s", srcRepo, b.Source.Release)
	}
}

func updateCatalogsForCrossShelfMove(id string, b *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	// Load and update source catalog
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return err
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return err
	}

	// Remove from source
	books, _ = catalog.Remove(books, id)
	srcData, err := catalog.Marshal(books)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, srcData,
		fmt.Sprintf("move: remove %s (moved to %s)", id, dst.shelf.Name)); err != nil {
		return err
	}

	// Update book metadata for destination
	b.Source.Release = dst.release
	b.Source.Owner = dst.owner
	b.Source.Repo = dst.repo

	// Add to destination catalog
	dstCatalogPath := dst.shelf.EffectiveCatalogPath()
	dstData, _, _ := gh.GetFileContent(dst.owner, dst.repo, dstCatalogPath, "")
	dstBooks, _ := catalog.Parse(dstData)
	dstBooks = catalog.Append(dstBooks, *b)
	dstCatalogData, err := catalog.Marshal(dstBooks)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(dst.owner, dst.repo, dstCatalogPath, dstCatalogData,
		fmt.Sprintf("move: add %s (from %s)", id, srcShelf.Name)); err != nil {
		return err
	}

	ok("Catalog updated")

	// Update README files for both shelves
	updateREADMEAfterRemove(srcOwner, srcShelf.Repo, books, b.ID)
	updateREADMEAfterAdd(dst.owner, dst.repo, dstBooks, *b)

	return nil
}

func updateCatalogForSameShelfMove(id string, b *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return err
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return err
	}

	// Update release field for the book
	for i := range books {
		if books[i].ID == id {
			books[i].Source.Release = dst.release
			break
		}
	}

	newData, err := catalog.Marshal(books)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, newData,
		fmt.Sprintf("move: %s → release/%s", id, dst.release)); err != nil {
		return err
	}

	ok("Catalog updated")
	return nil
}

// updateREADMEAfterRemove updates a shelf README after removing a book
func updateREADMEAfterRemove(owner, repo string, remainingBooks []catalog.Book, removedBookID string) {
	readmeData, _, err := gh.GetFileContent(owner, repo, "README.md", "")
	if err != nil {
		return // README doesn't exist or can't be read
	}

	readmeContent := string(readmeData)
	readmeContent = updateShelfREADMEStats(readmeContent, len(remainingBooks))
	readmeContent = removeFromShelfREADME(readmeContent, removedBookID)

	readmeMsg := fmt.Sprintf("Update README: remove %s", removedBookID)
	if err := gh.CommitFile(owner, repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
		warn("Could not update source README.md: %v", err)
	} else {
		ok("Source README.md updated")
	}
}

// updateREADMEAfterAdd updates a shelf README after adding a book
func updateREADMEAfterAdd(owner, repo string, books []catalog.Book, book catalog.Book) {
	readmeData, _, err := gh.GetFileContent(owner, repo, "README.md", "")
	if err != nil {
		return // README doesn't exist or can't be read
	}

	readmeContent := string(readmeData)
	readmeContent = updateShelfREADMEStats(readmeContent, len(books))
	readmeContent = appendToShelfREADME(readmeContent, book)

	readmeMsg := fmt.Sprintf("Update README: add %s", book.ID)
	if err := gh.CommitFile(owner, repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
		warn("Could not update destination README.md: %v", err)
	} else {
		ok("Destination README.md updated")
	}
}
