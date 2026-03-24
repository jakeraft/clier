package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	db "github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/adapter/settings"
	"github.com/spf13/cobra"
)

const configDirName = ".clier"

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

func newStore() (*db.Store, error) {
	_, store, err := newSettingsAndStore()
	return store, err
}

func newSettingsAndStore() (*settings.Settings, *db.Store, error) {
	cfg, err := newSettings()
	if err != nil {
		return nil, nil, err
	}
	if err := cfg.EnsureDirs(); err != nil {
		return nil, nil, err
	}
	store, err := db.NewStore(cfg.DBPath())
	if err != nil {
		return nil, nil, err
	}
	return cfg, store, nil
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated workspaces",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
