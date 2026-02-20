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

// showShelfArchitectureHelp displays information about shelf structure and organization
func showShelfArchitectureHelp() {
	fmt.Println()
	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println(color.CyanString("  How Shelves Work"))
	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println()

	fmt.Println(color.YellowString("Structure:"))
	fmt.Println("  Each shelf is a GitHub repository with:")
	fmt.Println("    • catalog.yml (in Git) - Metadata for your books")
	fmt.Println("    • Release assets (not in Git) - The actual PDF/EPUB files")
	fmt.Println()

	fmt.Println(color.YellowString("Organization Strategy:"))
	fmt.Println("  1. " + color.GreenString("Start broad") + " - One shelf is often enough at first")
	fmt.Println("     Example: shelf-books (general collection)")
	fmt.Println()
	fmt.Println("  2. " + color.GreenString("Use tags") + " - Organize books within a shelf using tags")
	fmt.Println("     Example: --tags programming,golang,textbook")
	fmt.Println()
	fmt.Println("  3. " + color.GreenString("Split later") + " - When a shelf grows large, split it")
	fmt.Println("     Use: shelfctl split (interactive wizard)")
	fmt.Println()

	fmt.Println(color.YellowString("When to Create Multiple Shelves:"))
	fmt.Println("  ✓ Different topics with distinct audiences")
	fmt.Println("    Example: shelf-work, shelf-personal, shelf-research")
	fmt.Println()
	fmt.Println("  ✓ When a shelf exceeds ~200-300 books")
	fmt.Println("    GitHub releases work best with moderate asset counts")
	fmt.Println()
	fmt.Println("  ✓ Different access requirements")
	fmt.Println("    Example: shelf-public (public repo), shelf-private (private repo)")
	fmt.Println()

	fmt.Println(color.YellowString("Naming Conventions:"))
	fmt.Println("  • Use " + color.CyanString("shelf-<topic>") + " for the repository name")
	fmt.Println("    Examples: shelf-programming, shelf-fiction, shelf-papers")
	fmt.Println()
	fmt.Println("  • The shelf name (config) is shorter:")
	fmt.Println("    Examples: programming, fiction, papers")
	fmt.Println()

	fmt.Println(color.YellowString("Advanced: Sub-organization with Releases"))
	fmt.Println("  You can use multiple releases within one shelf:")
	fmt.Println()
	fmt.Println("  shelf-programming/")
	fmt.Println("    release: library (default)")
	fmt.Println("    release: textbooks")
	fmt.Println("    release: papers")
	fmt.Println()
	fmt.Println("  Move books between releases with: shelfctl move --to-release")
	fmt.Println()

	fmt.Println(color.YellowString("Splitting Shelves:"))
	fmt.Println("  Don't worry about perfect organization now!")
	fmt.Println()
	fmt.Println("  Later, you can run: " + color.CyanString("shelfctl split"))
	fmt.Println()
	fmt.Println("  This launches a wizard that:")
	fmt.Println("    • Groups books by tags or other criteria")
	fmt.Println("    • Helps you decide which books go where")
	fmt.Println("    • Moves everything automatically")
	fmt.Println()

	fmt.Println(color.YellowString("Recommendation for First Shelf:"))
	fmt.Println("  Start with " + color.GreenString("shelf-books") + " or " + color.GreenString("shelf-library"))
	fmt.Println("  → Simple, general-purpose name")
	fmt.Println("  → Easy to split later by topic")
	fmt.Println("  → Use tags for organization in the meantime")
	fmt.Println()

	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println()
}

// runInteractiveInit guides the user through creating their first shelf
func runInteractiveInit() error {
	fmt.Println(color.CyanString("📚 Let's set up your first shelf!"))
	fmt.Println()
	fmt.Println(color.GreenString("Tip:") + " Type 'help' or '?' at any prompt for more information about shelf structure")
	fmt.Println()

	// Offer architecture info upfront
	fmt.Print("Want to learn about shelf architecture first? (y/n): ")
	var wantInfo string
	fmt.Scanln(&wantInfo)
	if wantInfo == "y" || wantInfo == "Y" || wantInfo == "yes" {
		showShelfArchitectureHelp()
	}
	fmt.Println()

	// Get repo name with help option
	var repoName string
	for {
		fmt.Print("Repository name (e.g., shelf-books): ")
		fmt.Scanln(&repoName)

		if repoName == "help" || repoName == "?" {
			showShelfArchitectureHelp()
			fmt.Println()
			continue
		}

		if repoName == "" {
			repoName = "shelf-books"
			fmt.Printf("  Using default: %s\n", repoName)
		}
		break
	}

	// Get shelf name with help option
	var shelfName string
	for {
		fmt.Print("Shelf name (e.g., books): ")
		fmt.Scanln(&shelfName)

		if shelfName == "help" || shelfName == "?" {
			showShelfArchitectureHelp()
			fmt.Println()
			continue
		}

		if shelfName == "" {
			// Default: strip "shelf-" prefix if present
			shelfName = repoName
			if len(repoName) > 6 && repoName[:6] == "shelf-" {
				shelfName = repoName[6:]
			}
			fmt.Printf("  Using default: %s\n", shelfName)
		}
		break
	}

	// Confirm creation
	fmt.Println()
	fmt.Printf("This will create:\n")
	fmt.Printf("  • GitHub repository: %s/%s\n", cfg.GitHub.Owner, repoName)
	fmt.Printf("  • Release tag: library\n")
	fmt.Printf("  • Shelf name: %s\n", shelfName)
	fmt.Println()
	fmt.Print("Proceed? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		fmt.Println(color.YellowString("Cancelled."))
		fmt.Println()
		fmt.Println("You can run this manually anytime:")
		fmt.Printf("  %s\n", color.CyanString(fmt.Sprintf("shelfctl init --repo %s --name %s --create-repo --create-release", repoName, shelfName)))
		return nil
	}

	// Run init command
	fmt.Println()
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{
		"--repo", repoName,
		"--name", shelfName,
		"--create-repo",
		"--create-release",
	})

	if err := initCmd.Execute(); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓ Shelf created successfully!"))
	fmt.Println()
	fmt.Println("What's next?")
	fmt.Printf("  1. Add your first book:\n")
	fmt.Printf("     %s\n\n", color.CyanString("shelfctl shelve"))
	fmt.Printf("  2. Or run the interactive menu:\n")
	fmt.Printf("     %s\n", color.CyanString("shelfctl"))

	return nil
}

// runHub launches the interactive hub menu and routes to selected action
func runHub() error {
	// Check configuration status
	hasToken := cfg != nil && cfg.GitHub.Token != ""
	hasShelves := cfg != nil && len(cfg.Shelves) > 0

	// If not fully configured, show welcome/setup message
	if !hasToken || !hasShelves {
		fmt.Println(color.YellowString("⚠ Welcome to shelfctl!"))
		fmt.Println()
		fmt.Println("Setup status:")
		fmt.Println()

		// Token status
		if hasToken {
			fmt.Printf("  %s GitHub token configured\n", color.GreenString("✓"))
		} else {
			fmt.Printf("  %s GitHub token not found\n", color.RedString("✗"))
		}

		// Shelves status
		if hasShelves {
			fmt.Printf("  %s %d shelf(s) configured\n", color.GreenString("✓"), len(cfg.Shelves))
		} else {
			fmt.Printf("  %s No shelves configured\n", color.RedString("✗"))
		}

		fmt.Println()

		// Show specific next steps based on what's missing
		if !hasToken {
			fmt.Println("Next step: Set your GitHub token")
			fmt.Printf("  %s\n\n", color.CyanString("export GITHUB_TOKEN=ghp_your_token_here"))
			fmt.Println("Then run 'shelfctl' again.")
		} else if !hasShelves {
			fmt.Println("Next step: Create your first shelf")
			fmt.Println()

			// Offer to guide them through init
			if util.IsTTY() {
				fmt.Print("Would you like to create a shelf now? (y/n): ")
				var response string
				fmt.Scanln(&response)
				if response == "y" || response == "Y" || response == "yes" {
					fmt.Println()
					return runInteractiveInit()
				}
			}

			fmt.Println()
			fmt.Println("Or run manually:")
			fmt.Printf("  %s\n\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
			fmt.Println("Then run 'shelfctl' again to use the interactive menu.")
		}

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
