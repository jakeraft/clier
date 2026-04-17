package cmd

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jakeraft/clier/internal/config"
	"github.com/jakeraft/clier/internal/domain"
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
	cmd.AddCommand(newConfigSetWorkspaceDirCmd())
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

func newConfigSetWorkspaceDirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace-dir <absolute-path>",
		Short: "Set the workspace directory (must be absolute)",
		Long: `Set the workspace directory where clier owns the per-team
working copies and the central .runs/ store.

The path must be absolute so that all clier commands work
identically regardless of the user's current directory. Relative
paths are rejected.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if !filepath.IsAbs(path) {
				return &domain.Fault{
					Kind:    domain.KindWorkspaceDirNotAbsolute,
					Subject: map[string]string{"path": path},
				}
			}

			cfg, err := loadRawConfig()
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if cfg == nil {
				cfg = &config.File{}
			}

			cfg.WorkspaceDir = path
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
