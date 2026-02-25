package unified

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/migrate"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	tea "github.com/charmbracelet/bubbletea"
)

func (m ImportRepoModel) scanAsync() tea.Cmd {
	gh := m.gh
	cfg := m.cfg
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	destRelTag := m.destRelTag

	return func() tea.Msg {
		// Scan source repo for book files
		token := os.Getenv(cfg.GitHub.TokenEnv)
		apiBase := cfg.GitHub.APIBase
		files, err := migrate.ScanRepo(token, apiBase, srcOwner, srcRepo, "main", []string{"pdf", "epub", "mobi", "djvu", "azw3", "cbz", "cbr"})
		if err != nil {
			return importRepoScanCompleteMsg{err: fmt.Errorf("scanning source repo: %w", err)}
		}

		// Ensure release
		rel, err := gh.EnsureRelease(destOwner, destRepo, destRelTag)
		if err != nil {
			return importRepoScanCompleteMsg{err: fmt.Errorf("ensuring release: %w", err)}
		}

		return importRepoScanCompleteMsg{
			files:   files,
			release: rel,
		}
	}
}

func (m ImportRepoModel) processAsync(ch chan importRepoProgressMsg) tea.Cmd {
	gh := m.gh
	toImport := m.toImport
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	release := m.release
	destRelTag := m.destRelTag
	destShelfName := m.destShelf.Name

	go func() {
		defer close(ch)
		total := len(toImport)

		// Open ledger for resumability
		ledger, _ := migrate.OpenLedger(migrate.DefaultLedgerPath())

		for i, f := range toImport {
			ch <- importRepoProgressMsg{kind: "status", path: f.Path, current: i + 1, total: total}

			// Check ledger for already-migrated
			if ledger != nil {
				source := fmt.Sprintf("%s/%s:%s", srcOwner, srcRepo, f.Path)
				if already, _ := ledger.Contains(source); already {
					ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
						err: fmt.Errorf("already migrated: %s", f.Path)}
					continue
				}
			}

			// Download file content
			data, _, err := gh.GetFileContent(srcOwner, srcRepo, f.Path, "")
			if err != nil {
				ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
					err: fmt.Errorf("download failed for %s: %w", f.Path, err)}
				continue
			}

			// Compute SHA256
			hash := sha256.Sum256(data)
			shaHex := fmt.Sprintf("%x", hash)

			// Generate metadata
			baseName := fileBaseName(f.Path)
			bookID := slugify(baseName)
			ext := strings.TrimPrefix(filepath.Ext(f.Path), ".")
			assetName := bookID + "." + ext

			// Write to temp file for upload
			tmp, err := os.CreateTemp("", "shelfctl-import-*")
			if err != nil {
				ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
					err: fmt.Errorf("temp file failed for %s: %w", f.Path, err)}
				continue
			}
			tmpPath := tmp.Name()

			if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmpPath)
				ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
					err: fmt.Errorf("write temp failed for %s: %w", f.Path, err)}
				continue
			}
			_ = tmp.Close()

			// Upload to destination release
			uploadFile, err := os.Open(tmpPath)
			if err != nil {
				_ = os.Remove(tmpPath)
				ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
					err: fmt.Errorf("open temp failed for %s: %w", f.Path, err)}
				continue
			}

			_, err = gh.UploadAsset(destOwner, destRepo, release.ID, assetName,
				uploadFile, int64(len(data)), "application/octet-stream")
			_ = uploadFile.Close()
			_ = os.Remove(tmpPath)

			if err != nil {
				ch <- importRepoProgressMsg{kind: "done", path: f.Path, current: i + 1, total: total,
					err: fmt.Errorf("upload failed for %s: %w", f.Path, err)}
				continue
			}

			// Build catalog entry
			newBook := catalog.Book{
				ID:     bookID,
				Title:  baseName,
				Author: "Unknown",
				Format: ext,
				Source: catalog.Source{
					Type:    "github_release",
					Owner:   destOwner,
					Repo:    destRepo,
					Release: destRelTag,
					Asset:   assetName,
				},
				Checksum: catalog.Checksum{
					SHA256: shaHex,
				},
				Meta: catalog.Meta{
					AddedAt:      time.Now().UTC().Format(time.RFC3339),
					MigratedFrom: fmt.Sprintf("%s/%s:%s", srcOwner, srcRepo, f.Path),
				},
			}

			// Record in ledger
			if ledger != nil {
				_ = ledger.Append(migrate.LedgerEntry{
					Source: fmt.Sprintf("%s/%s:%s", srcOwner, srcRepo, f.Path),
					BookID: bookID,
					Shelf:  destShelfName,
				})
			}

			ch <- importRepoProgressMsg{kind: "done", path: f.Path, book: &newBook, current: i + 1, total: total}
		}
	}()

	return nil // caller uses tea.Batch with waitFor
}

func (m ImportRepoModel) commitAsync() tea.Cmd {
	gh := m.gh
	importedBooks := m.importedBooks
	destOwner := m.destOwner
	destRepo := m.destShelf.Repo
	destCatPath := m.destCatPath
	srcOwner := m.srcOwner
	srcRepo := m.srcRepo

	return func() tea.Msg {
		// Load existing catalog
		dstData, _, _ := gh.GetFileContent(destOwner, destRepo, destCatPath, "")
		dstBooks, _ := catalog.Parse(dstData)

		allBooks := dstBooks
		for _, b := range importedBooks {
			allBooks = catalog.Append(allBooks, b)
		}

		data, err := catalog.Marshal(allBooks)
		if err != nil {
			return importRepoCommitCompleteMsg{err: err}
		}

		msg := fmt.Sprintf("migrate: %d files from %s/%s", len(importedBooks), srcOwner, srcRepo)
		if err := gh.CommitFile(destOwner, destRepo, destCatPath, data, msg); err != nil {
			return importRepoCommitCompleteMsg{err: err}
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
					fmt.Sprintf("Update README: migrate %d files", len(importedBooks)))
			}
		}

		return importRepoCommitCompleteMsg{}
	}
}

func waitForImportRepoProgress(ch <-chan importRepoProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return importRepoProgressMsg{kind: "done", current: 0, total: 0}
		}
		return msg
	}
}
