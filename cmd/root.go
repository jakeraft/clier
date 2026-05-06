package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jakeraft/clier/internal/config"
)

// The binary name is always `clier` — dev vs release is a value of the
// `channel` build variable (see version --json), not a separate binary
// identity. A machine carries one `clier` at a time (brew for release,
// `make install-local` for dev); the user picks which install path is
// active and the binary is interchangeable from the agent's POV.
//
// DisableSuggestions turns off cobra's default "Did you mean ..." typo
// surface — the CLI does not coach users; `clier --help` is the single
// surface for command discovery, and an unknown command exits with the
// raw rejection only.
var rootCmd = &cobra.Command{
	Use:                "clier",
	Short:              "Harness multi-agent teams with a native CLI",
	Long:               rootLong,
	SilenceUsage:       true,
	SilenceErrors:      true,
	DisableSuggestions: true,
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

Output is JSON on stdout for every successful command. Errors print on
stderr starting with "error: " and exit non-zero. Server errors are a
single summary line ("<status> <title>: <detail>"); client-side
validation (missing arguments, unknown commands) follows the same
single-line shape. Use 'clier <command> --help' to discover every
command and flag.`

// buildInfo holds the ldflags-stamped identity of this binary. Surfaced
// via `--version` (cobra default) and `version` (machine-readable JSON
// for any automation that wants to confirm channel / version / server
// URL — the CLI does not assume a specific consumer).
var buildInfo = struct {
	Version string
	Channel string
	Commit  string
}{Version: "dev", Channel: "release"}

// SetBuildInfo is called from main.go so the build pipeline can stamp
// version / channel / commit. Channel marks which install path produced
// the binary ("release" for brew / source / `go install`, "local" for
// `make install-local`); the value surfaces through `clier version`
// for any consumer that needs to confirm what binary it is talking to.
func SetBuildInfo(version, channel, commit string) {
	buildInfo.Version = version
	buildInfo.Channel = channel
	buildInfo.Commit = commit
	rootCmd.Version = version
}

func init() {
	rootCmd.AddCommand(newVersionCmd())

	// Disable cobra's auto-generated `help` subcommand. Discovery is the
	// `--help` flag's job — every command honors it. A second surface
	// (`clier help auth`) would print the same output via two paths and,
	// for unknown commands, print to stdout + exit 0 — breaking the
	// CLI's "errors on stderr, prefix `error:`, non-zero exit" contract.
	//
	// Cobra's help template lists any command literally named "help" in
	// the visible command list even when Hidden is set, so the override
	// uses an unreachable Use value to stay out of `clier --help` while
	// still capturing `clier help <anything>` and emitting the unknown-
	// command error.
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "_disabled-help",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("unknown command %q for %q", "help", cmd.Root().Name())
		},
	})
}

// newVersionCmd emits the build identity as JSON — same success-on-stdout
// JSON contract every other command follows, so a consumer parses one
// shape across the whole CLI surface.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the build identity (binary, channel, server URL)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Default()
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"version":       buildInfo.Version,
				"channel":       buildInfo.Channel,
				"commit":        buildInfo.Commit,
				"server_url":    cfg.ServerURL,
				"dashboard_url": cfg.DashboardURL,
			})
		},
	}
}

// Execute is the CLI entry point. Errors are formatted on stderr and exit
// codes are non-zero; success is silent (commands print their own JSON).
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
