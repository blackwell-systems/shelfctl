package app

import (
	"fmt"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/cache"
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
	flagCreateShelf   bool
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
		// If no subcommand provided and in TUI mode, launch unified TUI
		if tui.ShouldUseTUI(cmd) {
			return runUnifiedTUI()
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
	rootCmd.PersistentFlags().BoolVar(&flagCreateShelf, "create-shelf", false, "Auto-create shelf if it doesn't exist")

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
		newDeleteShelfCmd(),
		newDeleteBookCmd(),
		newEditBookCmd(),
		newBrowseCmd(),
		newInfoCmd(),
		newOpenCmd(),
		newShelveCmd(),
		newMoveCmd(),
		newSplitCmd(),
		newMigrateCmd(),
		newImportCmd(),
		newIndexCmd(),
		newVerifyCmd(),
		newSyncCmd(),
		newCacheCmd(),
		newStatusCmd(),
		newSearchCmd(),
		newTagsCmd(),
		newCompletionCmd(),
	)

	// Set up colored help template after commands are added
	setupColoredHelp()
}

// setupColoredHelp customizes Cobra's help template with colors
func setupColoredHelp() {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	helpTemplate := `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

	usageTemplate := cyan("Usage:") + `{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

` + cyan("Aliases:") + `
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

` + cyan("Examples:") + `
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

` + cyan("Available Commands:") + `{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  ` + green("{{rpad .Name .NamePadding}}") + ` {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  ` + green("{{rpad .Name .NamePadding}}") + ` {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  ` + green("{{rpad .Name .NamePadding}}") + ` {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + cyan("Flags:") + `
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + cyan("Global Flags:") + `
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use ` + yellow("{{.CommandPath}} [command] --help") + ` for more information about a command.{{end}}
`

	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)

	// Apply to all subcommands recursively
	applyTemplateRecursive(rootCmd, helpTemplate, usageTemplate)
}

// applyTemplateRecursive applies help templates to a command and all its subcommands
func applyTemplateRecursive(cmd *cobra.Command, helpTpl, usageTpl string) {
	for _, subCmd := range cmd.Commands() {
		subCmd.SetHelpTemplate(helpTpl)
		subCmd.SetUsageTemplate(usageTpl)
		applyTemplateRecursive(subCmd, helpTpl, usageTpl)
	}
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
