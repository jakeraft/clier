package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/auth"
	"github.com/spf13/cobra"
)

const defaultServerURL = "http://localhost:8080"

// newAPIClient creates an API client.
// Token is loaded from credentials if available, empty otherwise.
func newAPIClient() *api.Client {
	token := ""
	creds, err := auth.Load(auth.DefaultPath())
	if err == nil {
		token = creds.Token
	}
	return api.NewClient(defaultServerURL, token)
}

// requireLogin loads credentials and returns login.
// Exits with error if not logged in.
func requireLogin() string {
	creds, err := auth.Load(auth.DefaultPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: not logged in. Run 'clier auth login' first.")
		os.Exit(1)
	}
	return creds.Login
}

func newStore() *api.Store {
	return api.NewStore(newAPIClient(), requireLogin())
}

var rootCmd = &cobra.Command{
	Use:   "clier",
	Short: "Orchestrate AI coding agent teams in isolated workspaces",
	Long: `Orchestrate AI coding agent teams in isolated workspaces.

Building blocks (prompt, settings, repo) define agent capabilities.
Combine them into a member, assemble members into a team with
leader-worker relations, then start a run to launch the agents.
Monitor progress through messages and notes.

New to clier? Run "clier tutorial" for a step-by-step guide.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
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
// If no slash is present, requireLogin() is used as owner.
func parseOwnerName(s string) (owner, name string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return requireLogin(), s
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
