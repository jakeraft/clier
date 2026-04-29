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

It resolves teams against clier-server's RunManifest API, clones the
required mounts into a per-run scratch dir, and launches one tmux window
per agent. Protocol injection (peer list, tell commands, operating rules)
is composed server-side and arrives as agent-type-specific args — the CLI
is vendor-blind.

Get started:
  clier auth login                 Log in via GitHub device flow
  clier run start <ns/name>        Launch a team in tmux
  clier run attach <run-id>        Watch and intervene in real time
  clier run tell --to <agent> ...  Message an agent
  clier run stop <run-id>          Tear the run down

Output is line-delimited JSON on stdout for every successful command.
Errors print a single line on stderr and exit non-zero.`

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
