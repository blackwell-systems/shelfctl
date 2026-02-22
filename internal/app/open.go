package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var (
		shelfName string
		app       string
	)

	cmd := &cobra.Command{
		Use:   "open <id>",
		Short: "Open a book (downloads to cache first if needed)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			b, shelf, err := findBook(id, shelfName)
			if err != nil {
				return err
			}
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)

			// Ensure cached.
			if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				// Find the release and asset.
				rel, err := gh.GetReleaseByTag(owner, shelf.Repo, b.Source.Release)
				if err != nil {
					return fmt.Errorf("release %q: %w", b.Source.Release, err)
				}
				asset, err := gh.FindAsset(owner, shelf.Repo, rel.ID, b.Source.Asset)
				if err != nil {
					return fmt.Errorf("finding asset: %w", err)
				}
				if asset == nil {
					return fmt.Errorf("asset %q not found in release %q", b.Source.Asset, b.Source.Release)
				}

				rc, err := gh.DownloadAsset(owner, shelf.Repo, asset.ID)
				if err != nil {
					return fmt.Errorf("download: %w", err)
				}
				defer func() { _ = rc.Close() }()

				// Use progress bar in TTY mode
				if util.IsTTY() && tui.ShouldUseTUI(cmd) {
					progressCh := make(chan int64, 50)
					errCh := make(chan error, 1)

					// Show connecting message
					fmt.Printf("Connecting to GitHub...\n")

					// Start download in goroutine
					go func() {
						pr := tui.NewProgressReader(rc, asset.Size, progressCh)
						_, err := cacheMgr.Store(owner, shelf.Repo, b.ID, b.Source.Asset, pr, b.Checksum.SHA256)
						close(progressCh)
						errCh <- err
					}()

					// Show progress UI
					label := fmt.Sprintf("Downloading %s (%s)", b.ID, humanBytes(asset.Size))
					if err := tui.ShowProgress(label, asset.Size, progressCh); err != nil {
						return err // User cancelled
					}

					// Get result
					if err := <-errCh; err != nil {
						return fmt.Errorf("cache: %w", err)
					}
				} else {
					// Non-interactive mode: just print and download
					fmt.Printf("Downloading %s (%s) …\n", b.ID, humanBytes(asset.Size))
					_, err = cacheMgr.Store(owner, shelf.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
					if err != nil {
						return fmt.Errorf("cache: %w", err)
					}
				}
				ok("Cached")

				// Show poppler hint if needed (one-time, PDF only)
				showPopplerHintIfNeeded(b.Source.Asset)
			}

			path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			return openFile(path, app)
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Specify shelf if ID is ambiguous")
	cmd.Flags().StringVar(&app, "app", "", "Application to open the file with")
	return cmd
}

func openFile(path, app string) error {
	var cmdName string
	var args []string

	if app != "" {
		cmdName = app
		args = []string{path}
	} else {
		switch runtime.GOOS {
		case "darwin":
			cmdName = "open"
			args = []string{path}
		case "windows":
			cmdName = "cmd"
			args = []string{"/c", "start", "", path}
		default: // linux, freebsd, etc.
			cmdName = "xdg-open"
			args = []string{path}
		}
	}

	c := exec.Command(cmdName, args...)
	if err := c.Start(); err != nil {
		return fmt.Errorf("opening file with %q: %w", cmdName, err)
	}
	return nil
}

// showPopplerHintIfNeeded shows a one-time hint about installing poppler for PDF cover extraction.
func showPopplerHintIfNeeded(assetFilename string) {
	// Only for PDFs
	if !isPDF(assetFilename) {
		return
	}

	// Skip if poppler is already installed
	if cache.IsPopplerInstalled() {
		return
	}

	// Skip if we've shown the hint before
	configDir := filepath.Dir(config.DefaultPath())
	markerPath := filepath.Join(configDir, ".poppler-hint-shown")
	if _, err := os.Stat(markerPath); err == nil {
		return // Already shown
	}

	// Show hint
	fmt.Println()
	fmt.Println(cache.GetPopplerInstallHint())
	fmt.Println()

	// Mark as shown
	_ = os.WriteFile(markerPath, []byte("1"), 0644)
}

// isPDF checks if the filename indicates a PDF file
func isPDF(filename string) bool {
	return filepath.Ext(filename) == ".pdf"
}
