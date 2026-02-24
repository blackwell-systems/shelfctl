package app

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell autocompletion scripts",
		Long: `Generate autocompletion scripts for your shell.

Examples:
  # Bash (add to ~/.bashrc)
  source <(shelfctl completion bash)

  # Zsh (add to ~/.zshrc)
  source <(shelfctl completion zsh)

  # Fish
  shelfctl completion fish > ~/.config/fish/completions/shelfctl.fish

  # PowerShell
  shelfctl completion powershell | Out-String | Invoke-Expression`,
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}

	return cmd
}
