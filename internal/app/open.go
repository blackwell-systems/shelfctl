package app

import (
	"fmt"
	"os/exec"
	"runtime"

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
				// Reuse get logic.
				getCmd := newGetCmd()
				getCmd.SetArgs([]string{id, "--shelf", shelf.Name})
				if err := getCmd.Execute(); err != nil {
					return fmt.Errorf("downloading: %w", err)
				}
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
