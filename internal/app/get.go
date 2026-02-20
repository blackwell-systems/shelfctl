package app

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var (
		shelfName string
		force     bool
		copyTo    string
	)

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Download a book asset to the local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			b, shelf, err := findBook(id, shelfName)
			if err != nil {
				return err
			}
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)

			destPath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)

			if !force && cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
				ok("Already cached: %s", destPath)
				if copyTo != "" {
					if err := util.CopyFile(destPath, copyTo); err != nil {
						return err
					}
					ok("Copied to %s", copyTo)
				}
				return nil
			}

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
			defer rc.Close()

			path, err := cacheMgr.Store(owner, shelf.Repo, b.ID, b.Source.Asset, rc, b.Checksum.SHA256)
			if err != nil {
				return fmt.Errorf("cache: %w", err)
			}
			ok("Cached: %s", path)

			if copyTo != "" {
				if err := util.CopyFile(path, copyTo); err != nil {
					return err
				}
				ok("Copied to %s", copyTo)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Specify shelf if ID is ambiguous")
	cmd.Flags().BoolVar(&force, "force", false, "Re-download even if already cached")
	cmd.Flags().StringVar(&copyTo, "to", "", "Copy to this path after download")
	return cmd
}
