package unified

import (
	"fmt"
	"os"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	tea "github.com/charmbracelet/bubbletea"
)

func (m ImportShelfModel) scanAsync() tea.Cmd {
	gh := m.gh
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	destRelTag := m.destRelTag
	destCatPath := m.destCatPath

	return func() tea.Msg {
		// Load source catalog
		srcData, _, err := gh.GetFileContent(srcOwner, srcRepo, "catalog.yml", "")
		if err != nil {
			return importShelfScanCompleteMsg{err: fmt.Errorf("reading source catalog: %w", err)}
		}
		srcBooks, err := catalog.Parse(srcData)
		if err != nil {
			return importShelfScanCompleteMsg{err: fmt.Errorf("parsing source catalog: %w", err)}
		}

		// Load destination catalog
		dstData, _, _ := gh.GetFileContent(destOwner, destRepo, destCatPath, "")
		dstBooks, _ := catalog.Parse(dstData)

		// Build SHA256 dedup index
		existingSHAs := map[string]bool{}
		for _, b := range dstBooks {
			if b.Checksum.SHA256 != "" {
				existingSHAs[b.Checksum.SHA256] = true
			}
		}

		// Ensure release
		rel, err := gh.EnsureRelease(destOwner, destRepo, destRelTag)
		if err != nil {
			return importShelfScanCompleteMsg{err: fmt.Errorf("ensuring release: %w", err)}
		}

		return importShelfScanCompleteMsg{
			srcBooks:     srcBooks,
			dstBooks:     dstBooks,
			existingSHAs: existingSHAs,
			release:      rel,
		}
	}
}

func (m ImportShelfModel) processAsync(ch chan importShelfProgressMsg) tea.Cmd {
	gh := m.gh
	toImport := m.toImport
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	release := m.release
	destRelTag := m.destRelTag

	go func() {
		defer close(ch)
		total := len(toImport)

		for i, b := range toImport {
			ch <- importShelfProgressMsg{kind: "status", bookID: b.ID, current: i + 1, total: total}

			// Find source release asset
			srcRel, err := gh.GetReleaseByTag(b.Source.Owner, b.Source.Repo, b.Source.Release)
			if err != nil {
				ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, current: i + 1, total: total,
					err: fmt.Errorf("source release not found for %s: %w", b.ID, err)}
				continue
			}

			srcAsset, err := gh.FindAsset(b.Source.Owner, b.Source.Repo, srcRel.ID, b.Source.Asset)
			if err != nil || srcAsset == nil {
				ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, current: i + 1, total: total,
					err: fmt.Errorf("source asset not found for %s", b.ID)}
				continue
			}

			// Download and buffer to temp
			tmpPath, size, err := downloadAndBufferAsset(gh, b.Source.Owner, b.Source.Repo, srcAsset.ID)
			if err != nil {
				ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, current: i + 1, total: total,
					err: fmt.Errorf("download failed for %s: %w", b.ID, err)}
				continue
			}

			// Upload to destination release
			uploadFile, err := os.Open(tmpPath)
			if err != nil {
				_ = os.Remove(tmpPath)
				ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, current: i + 1, total: total,
					err: fmt.Errorf("open temp failed for %s: %w", b.ID, err)}
				continue
			}

			_, err = gh.UploadAsset(destOwner, destRepo, release.ID, b.Source.Asset,
				uploadFile, size, "application/octet-stream")
			_ = uploadFile.Close()
			_ = os.Remove(tmpPath)

			if err != nil {
				ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, current: i + 1, total: total,
					err: fmt.Errorf("upload failed for %s: %w", b.ID, err)}
				continue
			}

			// Build new book entry for destination
			newBook := b
			newBook.Source = catalog.Source{
				Type:    "github_release",
				Owner:   destOwner,
				Repo:    destRepo,
				Release: destRelTag,
				Asset:   b.Source.Asset,
			}
			newBook.Meta.AddedAt = time.Now().UTC().Format(time.RFC3339)
			newBook.Meta.MigratedFrom = fmt.Sprintf("%s/%s", srcOwner, srcRepo)

			ch <- importShelfProgressMsg{kind: "done", bookID: b.ID, book: &newBook, current: i + 1, total: total}
		}
	}()

	return nil // caller uses tea.Batch with waitFor
}

func (m ImportShelfModel) commitAsync() tea.Cmd {
	gh := m.gh
	importedBooks := m.importedBooks
	dstBooks := m.dstBooks
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	destCatPath := m.destCatPath
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo

	return func() tea.Msg {
		allBooks := dstBooks
		for _, b := range importedBooks {
			allBooks = catalog.Append(allBooks, b)
		}

		data, err := catalog.Marshal(allBooks)
		if err != nil {
			return importShelfCommitCompleteMsg{err: err}
		}

		msg := fmt.Sprintf("import: %d books from %s/%s", len(importedBooks), srcOwner, srcRepo)
		if err := gh.CommitFile(destOwner, destRepo, destCatPath, data, msg); err != nil {
			return importShelfCommitCompleteMsg{err: err}
		}

		// Update README
		readmeData, _, err := gh.GetFileContent(destOwner, destRepo, "README.md", "")
		if err == nil {
			orig := string(readmeData)
			content := operations.UpdateShelfREADMEStats(orig, len(allBooks))
			for _, b := range importedBooks {
				content = operations.AppendToShelfREADME(content, b)
			}
			if content != orig {
				_ = gh.CommitFile(destOwner, destRepo, "README.md", []byte(content),
					fmt.Sprintf("Update README: import %d books", len(importedBooks)))
			}
		}

		return importShelfCommitCompleteMsg{}
	}
}

func waitForImportShelfProgress(ch <-chan importShelfProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return importShelfProgressMsg{kind: "done", current: 0, total: 0}
		}
		return msg
	}
}
