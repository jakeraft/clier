package cmd

import (
	"fmt"
	"os"

	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/adapter/settings"
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
	Long: `Orchestrate AI coding agent teams in isolated workspaces.

Building blocks (profile, prompt, env, repo) define agent capabilities.
Combine them into a member, assemble members into a team with
leader-worker relations, then start a task to launch the agents.
Monitor progress through messages and updates, or open the dashboard.

New to clier? Run "clier tutorial" for a step-by-step guide.`,
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
		if _, err := generateDashboard(cmd.Context(), store, cfg.Paths.Dashboard()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: dashboard not updated: %v\n", err)
		}
		return nil
	},
}

func Execute() {
	if os.Getenv("CLIER_MEMBER_ID") != "" {
		filterAgentCommands()
	} else {
		filterUserCommands()
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// filterUserCommands removes agent-only subcommands from "task" in user context.
func filterUserCommands() {
	// Coupled to: newTaskUpdateCmd
	hidden := map[string]bool{"update": true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "task" {
			var keep []*cobra.Command
			for _, sub := range cmd.Commands() {
				if !hidden[sub.Name()] {
					keep = append(keep, sub)
				}
			}
			cmd.ResetCommands()
			for _, sub := range keep {
				cmd.AddCommand(sub)
			}
		}
	}
}

// filterAgentCommands removes all commands except "task" when running as an agent,
// and within "task" keeps only agent-facing subcommands (tell, update).
func filterAgentCommands() {
	allowed := map[string]bool{"task": true}
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

	// Coupled to: newTaskTellCmd, newTaskUpdateCmd
	agentSubs := map[string]bool{"tell": true, "update": true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "task" {
			var subs []*cobra.Command
			for _, sub := range cmd.Commands() {
				if agentSubs[sub.Name()] {
					subs = append(subs, sub)
				}
			}
			cmd.ResetCommands()
			for _, sub := range subs {
				cmd.AddCommand(sub)
			}
		}
	}
}
