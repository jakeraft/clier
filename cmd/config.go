package cmd

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
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
	cmd.AddCommand(newConfigGetCmd())
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
			return present.Success(cmd.OutOrStdout(), view.ConfigOf(cfg))
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

func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a config value",
		RunE:  subcommandRequired,
	}
	cmd.AddCommand(newConfigGetServerURLCmd())
	cmd.AddCommand(newConfigGetDashboardURLCmd())
	cmd.AddCommand(newConfigGetCredentialsPathCmd())
	cmd.AddCommand(newConfigGetWorkspaceDirCmd())
	return cmd
}

func newConfigGetServerURLCmd() *cobra.Command {
	return newConfigGetValueCmd("server-url", "server_url", func(cfg *config.File) string {
		return cfg.ServerURL
	})
}

func newConfigGetDashboardURLCmd() *cobra.Command {
	return newConfigGetValueCmd("dashboard-url", "dashboard_url", func(cfg *config.File) string {
		return cfg.DashboardURL
	})
}

func newConfigGetCredentialsPathCmd() *cobra.Command {
	return newConfigGetValueCmd("credentials-path", "credentials_path", func(cfg *config.File) string {
		return cfg.CredentialsPath
	})
}

func newConfigGetWorkspaceDirCmd() *cobra.Command {
	return newConfigGetValueCmd("workspace-dir", "workspace_dir", func(cfg *config.File) string {
		return cfg.WorkspaceDir
	})
}

func newConfigGetValueCmd(use, key string, getter func(*config.File) string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "Get the " + key + " value",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.ConfigValueOf(key, getter(cfg)))
		},
	}
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
			path, err := configPath()
			if err != nil {
				return err
			}
			if err := config.Save(path, cfg); err != nil {
				return err
			}

			resolved, err := config.Resolve(cfg)
			if err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.ConfigOf(resolved))
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
			configFilePath, err := configPath()
			if err != nil {
				return err
			}
			if err := config.Save(configFilePath, cfg); err != nil {
				return err
			}

			resolved, err := config.Resolve(cfg)
			if err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.ConfigOf(resolved))
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
			path, err := configPath()
			if err != nil {
				return err
			}
			if err := config.Save(path, cfg); err != nil {
				return err
			}

			resolved, err := config.Resolve(cfg)
			if err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.ConfigOf(resolved))
		},
	}
}
