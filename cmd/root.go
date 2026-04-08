package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/adapter/settings"
	"github.com/spf13/cobra"
)

// mutates is the annotation key that marks commands which modify data.
// PersistentPostRunE checks this to decide whether to regenerate the dashboard.
const mutates = "mutates"

func newSettings() (*settings.Settings, error) {
	return settings.New()
}

func newAPIClient() *api.Client {
	serverURL := os.Getenv("CLIER_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	token := os.Getenv("CLIER_TOKEN")
	return api.NewClient(serverURL, token)
}

func resolveOwner() string {
	owner := os.Getenv("CLIER_OWNER")
	if owner == "" {
		owner = "default"
	}
	return owner
}

func newStore() *api.Store {
	return api.NewStore(newAPIClient(), resolveOwner())
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated workspaces",
	Long: `Orchestrate AI coding agent teams in isolated workspaces.

Building blocks (prompt, settings, repo) define agent capabilities.
Combine them into a member, assemble members into a team with
leader-worker relations, then start a run to launch the agents.
Monitor progress through messages and notes, or open the dashboard.

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
		client := newAPIClient()
		if _, err := generateDashboard(cmd.Context(), client, resolveOwner(), cfg.Paths.Dashboard()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: dashboard not updated: %v\n", err)
		}
		return nil
	},
}

func Execute() {
	if os.Getenv("CLIER_AGENT") == "true" {
		filterAgentCommands()
	} else {
		filterUserCommands()
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// filterUserCommands removes agent-only subcommands from "run" in user context.
func filterUserCommands() {
	// Coupled to: newRunNoteCmd
	hidden := map[string]bool{"note": true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
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

// parseOwnerName splits "owner/name" into owner and name.
// If no slash is present, resolveOwner() is used as owner.
func parseOwnerName(s string) (owner, name string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return resolveOwner(), s
}

// filterAgentCommands removes all commands except "run" when running as an agent,
// and within "run" keeps only agent-facing subcommands (tell, note).
func filterAgentCommands() {
	allowed := map[string]bool{"run": true}
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

	// Coupled to: newRunTellCmd, newRunNoteCmd
	agentSubs := map[string]bool{"tell": true, "note": true}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
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
