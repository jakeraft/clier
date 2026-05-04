package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "clier",
	Short:         "Harness multi-agent teams with a native CLI",
	Long:          rootLong,
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

const rootLong = `clier is a thin tmux harness for AI coding agent teams.

It asks clier-server for a run manifest, clones each agent's repo into
a per-run scratch dir, drops the agent's rendered protocol markdown,
and launches one tmux window per agent. The vendor-specific launch
flags are composed server-side, so adding a new agent vendor never
requires a new CLI release.

Get started:
  clier tutorial                            Walk through your first run
  clier auth login                          Log in via GitHub device flow
  clier team list                           Browse the catalog
  clier run start <namespace/name>          Launch a team in tmux
  clier run attach <run-id>                 Watch and intervene in real time
  clier run tell --run <id> --to <agent>    Message an agent
  clier run stop <run-id>                   Tear the run down
  clier open dashboard                      Open the web UI

Output is JSON on stdout for every successful command. Errors print a
single line on stderr and exit non-zero.`

// SetVersion configures the --version string. Called from main.go so the
// build pipeline can stamp it.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute is the CLI entry point. Errors are formatted on stderr and exit
// codes are non-zero; success is silent (commands print their own JSON).
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
