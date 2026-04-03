package cmd

import (
	"fmt"
	"os"

	"github.com/jakeraft/clier/internal/adapter/dashboard"
	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/adapter/settings"
	"github.com/jakeraft/clier/ui"
	"github.com/spf13/cobra"
)

// mutates is the annotation key that marks commands which modify data.
// PersistentPostRunE checks this to decide whether to regenerate the dashboard.
const mutates = "mutates"

func newSettings() (*settings.Settings, error) {
	return settings.New()
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
			fmt.Fprintf(os.Stderr, "warning: dashboard not updated: %v\n", err)
			return nil
		}
		store, err := newStore(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: dashboard not updated: %v\n", err)
			return nil
		}
		defer store.Close()
		if _, err := dashboard.Generate(cmd.Context(), store, cfg.Paths.Dashboard(), ui.DistFS, ui.DistRoot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: dashboard not updated: %v\n", err)
		}
		return nil
	},
}

func Execute() {
	if os.Getenv("CLIER_MEMBER_ID") != "" {
		filterAgentCommands()
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// filterAgentCommands removes all commands except "session" when running as an agent.
func filterAgentCommands() {
	allowed := map[string]bool{"session": true}
	var keep []*cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if allowed[cmd.Name()] {
			keep = append(keep, cmd)
		}
	}
	rootCmd.ResetCommands()
	for _, cmd := range keep {
		rootCmd.AddCommand(cmd)
	}
}
