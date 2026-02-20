package app

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
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
				fmt.Printf("Not cached — downloading %s …\n", id)

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

				fmt.Printf("Downloading %s  (%s) …\n",
					color.WhiteString(b.ID),
					color.CyanString(humanBytes(asset.Size)))

				rc, err := gh.DownloadAsset(owner, shelf.Repo, asset.ID)
				if err != nil {
					return fmt.Errorf("download: %w", err)
				}
				defer func() { _ = rc.Close() }()

				path, err := cacheMgr.Store(owner, shelf.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
				if err != nil {
					return fmt.Errorf("cache: %w", err)
				}
				ok("Cached: %s", path)
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
