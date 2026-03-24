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
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return db.NewStore(filepath.Join(dir, "clier.db"))
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
