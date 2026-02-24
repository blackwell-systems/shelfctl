package unified

import (
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/readme"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Phase: Processing (async) ---

func (m MoveBookModel) moveAsync() tea.Cmd {
	toMove := m.toMove
	moveType := m.moveType
	destShelfName := m.destShelfName
	destRelease := m.destRelease
	gh := m.gh
	cfg := m.cfg
	cacheMgr := m.cacheMgr

	return func() tea.Msg {
		successCount := 0
		failCount := 0

		for _, item := range toMove {
			var err error
			if moveType == moveToShelf {
				err = moveSingleBookToShelf(item, destShelfName, gh, cfg, cacheMgr)
			} else {
				err = moveSingleBookToRelease(item, destRelease, gh, cfg)
			}

			if err != nil {
				failCount++
			} else {
				successCount++
			}
		}

		return moveCompleteMsg{
			successCount: successCount,
			failCount:    failCount,
		}
	}
}

// moveSingleBookToShelf moves a book to a different shelf (cross-repo)
func moveSingleBookToShelf(item tui.BookItem, destShelfName string, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) error {
	srcShelf := cfg.ShelfByName(item.ShelfName)
	if srcShelf == nil {
		return fmt.Errorf("source shelf %q not found", item.ShelfName)
	}
	dstShelf := cfg.ShelfByName(destShelfName)
	if dstShelf == nil {
		return fmt.Errorf("destination shelf %q not found", destShelfName)
	}

	srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)
	dstOwner := dstShelf.EffectiveOwner(cfg.GitHub.Owner)
	dstRelease := dstShelf.EffectiveRelease(cfg.Defaults.Release)

	b := &item.Book

	// Check if already at destination
	if srcOwner == dstOwner && srcShelf.Repo == dstShelf.Repo {
		return fmt.Errorf("book is already in shelf %q", destShelfName)
	}

	// 1. Ensure destination release
	dstRel, err := gh.EnsureRelease(dstOwner, dstShelf.Repo, dstRelease)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// 2. Get source asset
	srcRel, err := gh.GetReleaseByTag(srcOwner, srcShelf.Repo, b.Source.Release)
	if err != nil {
		return fmt.Errorf("getting source release: %w", err)
	}
	srcAsset, err := gh.FindAsset(srcOwner, srcShelf.Repo, srcRel.ID, b.Source.Asset)
	if err != nil {
		return fmt.Errorf("finding source asset: %w", err)
	}
	if srcAsset == nil {
		return fmt.Errorf("source asset %q not found", b.Source.Asset)
	}

	// 3. Download and buffer
	tmpPath, size, err := downloadAndBufferAsset(gh, srcOwner, srcShelf.Repo, srcAsset.ID)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// 4. Upload to destination
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(dstOwner, dstShelf.Repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading to destination: %w", err)
	}

	// 5. Delete old asset
	if err := gh.DeleteAsset(srcOwner, srcShelf.Repo, srcAsset.ID); err != nil {
		// Warn but continue — asset was already copied
		_ = err
	}

	// 6. Update source catalog (remove book)
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	srcData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return fmt.Errorf("loading source catalog: %w", err)
	}
	srcBooks, err := catalog.Parse(srcData)
	if err != nil {
		return fmt.Errorf("parsing source catalog: %w", err)
	}
	srcBooks, _ = catalog.Remove(srcBooks, b.ID)
	srcMarshal, err := catalog.Marshal(srcBooks)
	if err != nil {
		return fmt.Errorf("marshaling source catalog: %w", err)
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, srcMarshal,
		fmt.Sprintf("move: remove %s (moved to %s)", b.ID, destShelfName)); err != nil {
		return fmt.Errorf("committing source catalog: %w", err)
	}

	// 7. Update destination catalog (add book)
	dstCatalogPath := dstShelf.EffectiveCatalogPath()
	dstData, _, _ := gh.GetFileContent(dstOwner, dstShelf.Repo, dstCatalogPath, "")
	dstBooks, _ := catalog.Parse(dstData)

	// Update book metadata for destination
	movedBook := *b
	movedBook.Source.Release = dstRelease
	movedBook.Source.Owner = dstOwner
	movedBook.Source.Repo = dstShelf.Repo

	dstBooks = catalog.Append(dstBooks, movedBook)
	dstMarshal, err := catalog.Marshal(dstBooks)
	if err != nil {
		return fmt.Errorf("marshaling destination catalog: %w", err)
	}
	if err := gh.CommitFile(dstOwner, dstShelf.Repo, dstCatalogPath, dstMarshal,
		fmt.Sprintf("move: add %s (from %s)", b.ID, item.ShelfName)); err != nil {
		return fmt.Errorf("committing destination catalog: %w", err)
	}

	// 8. Clear local cache (path changes after move)
	if cacheMgr.Exists(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset) {
		_ = cacheMgr.Remove(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset)
	}

	// 9. Update README files
	// Source: remove book
	srcReadmeData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, "README.md", "")
	if err == nil {
		originalContent := string(srcReadmeData)
		readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(srcBooks))
		readmeContent = operations.RemoveFromShelfREADME(readmeContent, b.ID)
		if readmeContent != originalContent {
			_ = gh.CommitFile(srcOwner, srcShelf.Repo, "README.md", []byte(readmeContent),
				fmt.Sprintf("Update README: remove %s", b.ID))
		}
	}

	// Destination: add book
	dstReadmeMgr := readme.NewUpdater(gh, dstOwner, dstShelf.Repo)
	_ = dstReadmeMgr.UpdateWithStats(len(dstBooks), []catalog.Book{movedBook})

	// 10. Handle catalog cover if it exists
	if b.Cover != "" {
		coverData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, b.Cover, "")
		if err == nil {
			_ = gh.CommitFile(dstOwner, dstShelf.Repo, b.Cover, coverData,
				fmt.Sprintf("move: copy cover for %s", b.ID))
		}
	}

	return nil
}

// moveSingleBookToRelease moves a book to a different release within the same shelf
func moveSingleBookToRelease(item tui.BookItem, destRelease string, gh *github.Client, cfg *config.Config) error {
	shelf := cfg.ShelfByName(item.ShelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", item.ShelfName)
	}

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	b := &item.Book

	// Check if already at destination
	if b.Source.Release == destRelease {
		return fmt.Errorf("book is already at release %q", destRelease)
	}

	// 1. Ensure destination release
	dstRel, err := gh.EnsureRelease(owner, shelf.Repo, destRelease)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// 2. Get source asset
	srcRel, err := gh.GetReleaseByTag(owner, shelf.Repo, b.Source.Release)
	if err != nil {
		return fmt.Errorf("getting source release: %w", err)
	}
	srcAsset, err := gh.FindAsset(owner, shelf.Repo, srcRel.ID, b.Source.Asset)
	if err != nil {
		return fmt.Errorf("finding source asset: %w", err)
	}
	if srcAsset == nil {
		return fmt.Errorf("source asset %q not found", b.Source.Asset)
	}

	// 3. Download and buffer
	tmpPath, size, err := downloadAndBufferAsset(gh, owner, shelf.Repo, srcAsset.ID)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// 4. Upload to destination release
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(owner, shelf.Repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading: %w", err)
	}

	// 5. Delete old asset
	if err := gh.DeleteAsset(owner, shelf.Repo, srcAsset.ID); err != nil {
		_ = err // Warn but continue
	}

	// 6. Update catalog (change release field)
	catalogPath := shelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing catalog: %w", err)
	}

	for i := range books {
		if books[i].ID == b.ID {
			books[i].Source.Release = destRelease
			break
		}
	}

	newData, err := catalog.Marshal(books)
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}
	if err := gh.CommitFile(owner, shelf.Repo, catalogPath, newData,
		fmt.Sprintf("move: %s → release/%s", b.ID, destRelease)); err != nil {
		return fmt.Errorf("committing catalog: %w", err)
	}

	return nil
}

// downloadAndBufferAsset downloads a release asset to a temp file
func downloadAndBufferAsset(gh *github.Client, owner, repo string, assetID int64) (string, int64, error) {
	rc, err := gh.DownloadAsset(owner, repo, assetID)
	if err != nil {
		return "", 0, fmt.Errorf("downloading: %w", err)
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
