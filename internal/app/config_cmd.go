package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and modify configuration",
	}
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value and save to disk.

Supported keys:
  sync.auto_sync        true|false   Enable background auto-sync of modified books
  sync.debounce_minutes <int>        Minutes to wait before syncing a modified file

Examples:
  shelfctl config set sync.auto_sync true
  shelfctl config set sync.debounce_minutes 10`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			switch strings.ToLower(key) {
			case "sync.auto_sync":
				v, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("sync.auto_sync: expected true or false, got %q", value)
				}
				cfg.Sync.AutoSync = v
			case "sync.debounce_minutes":
				v, err := strconv.Atoi(value)
				if err != nil || v < 0 {
					return fmt.Errorf("sync.debounce_minutes: expected non-negative integer, got %q", value)
				}
				cfg.Sync.DebounceMinutes = v
			default:
				return fmt.Errorf("unknown config key %q\n\nRun 'shelfctl config set --help' for supported keys", key)
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			ok("Set %s = %s", key, value)
			return nil
		},
	}
}
