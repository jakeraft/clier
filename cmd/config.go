package cmd

import (
	"errors"
	"io/fs"

	"github.com/jakeraft/clier/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newConfigCmd())
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage clier settings",
		GroupID: rootGroupSettings,
		RunE:    subcommandRequired,
	}
	cmd.AddCommand(newConfigViewCmd())
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show current settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return printJSON(cfg)
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a config value",
		RunE:  subcommandRequired,
	}
	cmd.AddCommand(newConfigSetServerURLCmd())
	cmd.AddCommand(newConfigSetDashboardURLCmd())
	return cmd
}

func newConfigSetServerURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server-url <url>",
		Short: "Set the server URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadRawConfig()
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if cfg == nil {
				cfg = &config.File{}
			}

			cfg.ServerURL = args[0]
			if err := config.Save(configPath(), cfg); err != nil {
				return err
			}

			resolved, err := config.Resolve(cfg)
			if err != nil {
				return err
			}

			return printJSON(resolved)
		},
	}
}

func newConfigSetDashboardURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard-url <url>",
		Short: "Set the dashboard URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadRawConfig()
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if cfg == nil {
				cfg = &config.File{}
			}

			cfg.DashboardURL = args[0]
			if err := config.Save(configPath(), cfg); err != nil {
				return err
			}

			resolved, err := config.Resolve(cfg)
			if err != nil {
				return err
			}

			return printJSON(resolved)
		},
	}
}
