package app

import (
	"fmt"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cfg      *config.Config
	gh       *ghclient.Client
	cacheMgr *cache.Manager

	flagNoColor       bool
	flagNoInteractive bool
	flagConfig        string
)

var rootCmd = &cobra.Command{
	Use:   "shelfctl",
	Short: "Manage a personal document library using GitHub repos and releases",
	Long: `shelfctl manages PDF/EPUB libraries stored in GitHub Release assets.

Shelf repos hold metadata (catalog.yml). Release assets hold the files.
No self-hosted infrastructure required.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("error:"), err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&flagNoInteractive, "no-interactive", false, "Disable interactive TUI mode")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "Config file path (default: ~/.config/shelfctl/config.yml)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		util.InitColor(flagNoColor)

		// Allow init to run without an existing config.
		if cmd.Name() == "init" && cmd.Parent().Name() == "shelfctl" {
			var err error
			cfg, err = config.Load()
			if err != nil {
				cfg = &config.Config{}
			}
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if cfg.GitHub.Token == "" {
			return fmt.Errorf("no GitHub token found — set %s or SHELFCTL_GITHUB_TOKEN",
				cfg.GitHub.TokenEnv)
		}

		gh = ghclient.New(cfg.GitHub.Token, cfg.GitHub.APIBase)
		cacheMgr = cache.New(cfg.Defaults.CacheDir)
		return nil
	}

	// Register sub-commands.
	rootCmd.AddCommand(
		newInitCmd(),
		newShelvesCmd(),
		newBrowseCmd(),
		newInfoCmd(),
		newOpenCmd(),
		newShelveCmd(),
		newMoveCmd(),
		newSplitCmd(),
		newMigrateCmd(),
		newImportCmd(),
	)
}

// ok prints a green success line.
func ok(format string, a ...interface{}) {
	fmt.Println(color.GreenString("✓"), fmt.Sprintf(format, a...))
}

// warn prints a yellow warning line.
func warn(format string, a ...interface{}) {
	fmt.Fprintln(os.Stderr, color.YellowString("!"), fmt.Sprintf(format, a...))
}

// fail prints a red error and exits 1.
func fail(format string, a ...interface{}) {
	fmt.Fprintln(os.Stderr, color.RedString("✗"), fmt.Sprintf(format, a...))
	os.Exit(1)
}

// header prints a cyan section heading.
func header(format string, a ...interface{}) {
	fmt.Println(color.CyanString(fmt.Sprintf(format, a...)))
}
