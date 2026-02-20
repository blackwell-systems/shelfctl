package app

import (
	"fmt"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
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
No self-hosted infrastructure required.

Run 'shelfctl' with no arguments to launch the interactive menu.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand provided and in TUI mode, launch hub
		if tui.ShouldUseTUI(cmd) {
			return runHub()
		}
		// Otherwise show help
		return cmd.Help()
	},
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

		// Allow init and root (hub) to run without an existing config.
		if (cmd.Name() == "init" || cmd.Name() == "shelfctl") && cmd.Parent() == nil {
			var err error
			cfg, err = config.Load()
			if err != nil {
				cfg = &config.Config{}
			}
			// For root command (hub), still try to initialize clients if possible
			if cfg != nil && cfg.GitHub.Token != "" {
				gh = ghclient.New(cfg.GitHub.Token, cfg.GitHub.APIBase)
				cacheMgr = cache.New(cfg.Defaults.CacheDir)
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

// runHub launches the interactive hub menu and routes to selected action
func runHub() error {
	// Check if config is set up
	if cfg == nil || len(cfg.Shelves) == 0 {
		fmt.Println(color.YellowString("⚠ Welcome to shelfctl!"))
		fmt.Println()
		fmt.Println("You need to set up your first shelf before using the library.")
		fmt.Println()
		fmt.Println("Quick start:")
		fmt.Printf("  1. Set your GitHub token:\n")
		fmt.Printf("     %s\n\n", color.CyanString("export GITHUB_TOKEN=ghp_your_token_here"))
		fmt.Printf("  2. Create your first shelf:\n")
		fmt.Printf("     %s\n\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
		fmt.Printf("  3. Run shelfctl again to use the interactive menu\n")
		fmt.Println()
		fmt.Println("For more details, see docs/TUTORIAL.md or run 'shelfctl init --help'")
		return nil
	}

	// Gather context for the hub
	ctx := tui.HubContext{
		ShelfCount: len(cfg.Shelves),
	}

	// Count total books across all shelves (best effort)
	for _, shelf := range cfg.Shelves {
		owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
		catalogPath := shelf.EffectiveCatalogPath()
		if data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, ""); err == nil {
			if books, err := catalog.Parse(data); err == nil {
				ctx.BookCount += len(books)
			}
		}
	}

	action, err := tui.RunHub(ctx)
	if err != nil {
		return err
	}

	// Route to the appropriate command based on action
	switch action {
	case "browse":
		return newBrowseCmd().Execute()
	case "shelve":
		return newShelveCmd().Execute()
	case "quit":
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
