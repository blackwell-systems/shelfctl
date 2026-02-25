package unified

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Phase: Setup (async) ---

func (m ShelveModel) setupAsync() tea.Cmd {
	owner := m.owner
	repo := m.shelf.Repo
	catalogPath := m.catalogPath
	releaseTag := m.releaseTag
	gh := m.gh

	return func() tea.Msg {
		// Load existing catalog
		catalogMgr := catalog.NewManager(gh, owner, repo, catalogPath)
		existingBooks, err := catalogMgr.Load()
		if err != nil {
			return shelveSetupCompleteMsg{err: fmt.Errorf("loading catalog: %w", err)}
		}

		// Ensure release exists
		rel, err := gh.EnsureRelease(owner, repo, releaseTag)
		if err != nil {
			return shelveSetupCompleteMsg{err: fmt.Errorf("ensuring release: %w", err)}
		}

		return shelveSetupCompleteMsg{
			existingBooks: existingBooks,
			release:       rel,
		}
	}
}

// --- Phase: Ingesting (async) ---

func (m ShelveModel) ingestCurrentFile() tea.Cmd {
	filePath := m.selectedFiles[m.fileIndex]
	token := m.cfg.GitHub.Token
	apiBase := m.cfg.GitHub.APIBase

	return func() tea.Msg {
		// Resolve source
		src, err := ingest.Resolve(filePath, token, apiBase)
		if err != nil {
			return shelveIngestCompleteMsg{err: fmt.Errorf("resolve %s: %w", filePath, err)}
		}

		// Open and stream to temp file
		rc, err := src.Open()
		if err != nil {
			return shelveIngestCompleteMsg{err: fmt.Errorf("open: %w", err)}
		}

		tmp, err := os.CreateTemp("", "shelfctl-add-*")
		if err != nil {
			_ = rc.Close()
			return shelveIngestCompleteMsg{err: err}
		}
		tmpPath := tmp.Name()

		hr := ingest.NewReader(rc)
		if _, err := io.Copy(tmp, hr); err != nil {
			_ = tmp.Close()
			_ = rc.Close()
			_ = os.Remove(tmpPath)
			return shelveIngestCompleteMsg{err: fmt.Errorf("buffering: %w", err)}
		}
		_ = tmp.Close()
		_ = rc.Close()

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src.Name), "."))

		result := shelveIngestCompleteMsg{
			tmpPath: tmpPath,
			sha256:  hr.SHA256(),
			size:    hr.Size(),
			format:  ext,
			srcName: src.Name,
		}

		// Extract PDF metadata if applicable
		if ext == "pdf" {
			if pdfMeta, err := ingest.ExtractPDFMetadata(tmpPath); err == nil {
				result.pdfMetadata = pdfMeta
			}
		}

		return result
	}
}

// --- Phase: Uploading / Processing (async with progress) ---

func (m ShelveModel) processFileAsync(statusCh chan shelveProcessingMsg, bookID, title, author string, tags []string, assetName string) tea.Cmd {
	existingBooks := m.existingBooks
	sha256 := m.ingested.sha256
	tmpPath := m.ingested.tmpPath
	size := m.ingested.size
	format := m.ingested.format
	doCache := m.cacheLocally
	owner := m.owner
	repo := m.shelf.Repo
	releaseID := m.release.ID
	releaseTag := m.releaseTag
	gh := m.gh
	cacheMgr := m.cacheMgr

	go func() {
		defer close(statusCh)

		// 1. Check duplicates
		statusCh <- shelveProcessingMsg{kind: "status", status: "Checking for duplicates..."}
		for _, b := range existingBooks {
			if b.Checksum.SHA256 == sha256 {
				statusCh <- shelveProcessingMsg{
					kind: "done",
					err:  fmt.Errorf("duplicate: file already exists as %q (%s)", b.ID, b.Title),
				}
				return
			}
		}

		// 2. Check asset collision
		statusCh <- shelveProcessingMsg{kind: "status", status: "Checking for conflicts..."}
		existingAsset, err := gh.FindAsset(owner, repo, releaseID, assetName)
		if err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: fmt.Errorf("checking assets: %w", err)}
			return
		}
		if existingAsset != nil {
			statusCh <- shelveProcessingMsg{
				kind: "done",
				err:  fmt.Errorf("asset %q already exists in release %s", assetName, releaseTag),
			}
			return
		}

		// 3. Upload with progress
		statusCh <- shelveProcessingMsg{kind: "status", status: "Uploading..."}

		uploadFile, err := os.Open(tmpPath)
		if err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: err}
			return
		}

		progressCh := make(chan int64, 50)
		uploadErrCh := make(chan error, 1)

		go func() {
			defer uploadFile.Close() //nolint:errcheck
			progressCh <- 0
			pr := tui.NewProgressReader(uploadFile, size, progressCh)
			_, uploadErr := gh.UploadAsset(owner, repo, releaseID, assetName, pr, size, "application/octet-stream")
			close(progressCh)
			uploadErrCh <- uploadErr
		}()

		// Forward progress to status channel
		for bytes := range progressCh {
			statusCh <- shelveProcessingMsg{kind: "progress", bytes: bytes, total: size}
		}

		if err := <-uploadErrCh; err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: fmt.Errorf("upload: %w", err)}
			return
		}

		// 4. Build catalog entry
		book := catalog.Book{
			ID:        bookID,
			Title:     title,
			Author:    author,
			Tags:      tags,
			Format:    format,
			SizeBytes: size,
			Checksum:  catalog.Checksum{SHA256: sha256},
			Source: catalog.Source{
				Type:    "github_release",
				Owner:   owner,
				Repo:    repo,
				Release: releaseTag,
				Asset:   assetName,
			},
			Meta: catalog.Meta{
				AddedAt: time.Now().UTC().Format(time.RFC3339),
			},
		}

		// 5. Cache locally if requested
		if doCache {
			statusCh <- shelveProcessingMsg{kind: "status", status: "Caching locally..."}
			f, err := os.Open(tmpPath)
			if err == nil {
				_, _ = cacheMgr.Store(owner, repo, bookID, assetName, f, sha256)
				_ = f.Close()
			}
		}

		statusCh <- shelveProcessingMsg{kind: "done", book: &book}
	}()

	return waitForProcessingMsg(statusCh)
}

func waitForProcessingMsg(ch <-chan shelveProcessingMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return shelveProcessingMsg{kind: "done", err: fmt.Errorf("processing channel closed unexpectedly")}
		}
		return msg
	}
}

func (m ShelveModel) handleProcessingMsg(msg shelveProcessingMsg) (ShelveModel, tea.Cmd) {
	switch msg.kind {
	case "status":
		m.uploadStatus = msg.status
		return m, waitForProcessingMsg(m.statusCh)

	case "progress":
		m.uploadBytes = msg.bytes
		m.uploadTotal = msg.total
		if msg.total > 0 {
			m.uploadProgress = float64(msg.bytes) / float64(msg.total)
		}
		return m, waitForProcessingMsg(m.statusCh)

	case "done":
		// Clean up temp file
		if m.ingested != nil {
			_ = os.Remove(m.ingested.tmpPath)
			m.ingested = nil
		}

		if msg.err != nil {
			m.failCount++
		} else if msg.book != nil {
			m.newBooks = append(m.newBooks, *msg.book)
			m.existingBooks = catalog.Append(m.existingBooks, *msg.book)
			m.successCount++
		}

		return m.advanceToNextFileOrCommit()
	}

	return m, nil
}

func (m ShelveModel) advanceToNextFileOrCommit() (ShelveModel, tea.Cmd) {
	m.fileIndex++

	if m.fileIndex < len(m.selectedFiles) {
		// More files to process
		m.phase = shelveIngesting
		m.uploadStatus = ""
		m.uploadProgress = 0
		m.uploadBytes = 0
		return m, m.ingestCurrentFile()
	}

	// All files processed
	if m.successCount == 0 {
		// Nothing to commit
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	}

	// Commit
	m.phase = shelveCommitting
	return m, m.commitAsync()
}

// --- Phase: Committing (async) ---

func (m ShelveModel) commitAsync() tea.Cmd {
	existingBooks := m.existingBooks
	newBooks := m.newBooks
	owner := m.owner
	repo := m.shelf.Repo
	catalogPath := m.catalogPath
	gh := m.gh

	return func() tea.Msg {
		// Create commit message
		var msg string
		if len(newBooks) == 1 {
			msg = fmt.Sprintf("add: %s — %s", newBooks[0].ID, newBooks[0].Title)
		} else {
			msg = fmt.Sprintf("add: %d books", len(newBooks))
		}

		// Save catalog
		catalogMgr := catalog.NewManager(gh, owner, repo, catalogPath)
		if err := catalogMgr.Save(existingBooks, msg); err != nil {
			return shelveCommitCompleteMsg{err: err}
		}

		// Update README
		readmeData, _, readmeErr := gh.GetFileContent(owner, repo, "README.md", "")
		if readmeErr == nil {
			originalContent := string(readmeData)
			readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(existingBooks))
			for _, book := range newBooks {
				readmeContent = operations.AppendToShelfREADME(readmeContent, book)
			}
			if readmeContent != originalContent {
				var readmeMsg string
				if len(newBooks) == 1 {
					readmeMsg = fmt.Sprintf("Update README: add %s", newBooks[0].ID)
				} else {
					readmeMsg = fmt.Sprintf("Update README: add %d books", len(newBooks))
				}
				_ = gh.CommitFile(owner, repo, "README.md", []byte(readmeContent), readmeMsg)
			}
		}

		return shelveCommitCompleteMsg{}
	}
}
