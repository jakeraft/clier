package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/adapter/dashboard"
	"github.com/jakeraft/clier/internal/adapter/settings"
	"github.com/jakeraft/clier/web"
	"github.com/spf13/cobra"
)

const configDirName = ".clier"

// mutates is the annotation key that marks commands which modify data.
// PersistentPostRunE checks this to decide whether to regenerate the dashboard.
const mutates = "mutates"

func dataDir() (string, error) {
	if dir := os.Getenv("CLIER_DATA_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, configDirName), nil
}

func newSettings() (*settings.Settings, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	return settings.New(dir), nil
}

func newStore(cfg *settings.Settings) (*db.Store, error) {
	return db.NewStore(cfg.Paths.DB())
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated workspaces",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Annotations[mutates] == "" {
			return nil
		}
		cfg, err := newSettings()
		if err != nil {
			return nil
		}
		store, err := newStore(cfg)
		if err != nil {
			return nil
		}
		defer store.Close()
		_, _ = dashboard.Generate(cmd.Context(), store, cfg.Paths.Base(), web.DistFS, web.DistRoot)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
