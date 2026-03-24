package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/settings"
	"github.com/spf13/cobra"
)

const configDirName = ".clier"

func newSettings() (*settings.Settings, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return settings.New(filepath.Join(home, configDirName)), nil
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
