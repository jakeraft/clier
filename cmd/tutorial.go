package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tutorial",
		Short: "Walk through your first multi-agent run",
		Args:  cobra.NoArgs,
		Long: `clier tutorial — the shortest path to a working run.
Five minutes, end to end.

This page is a flow only. Per-command flags, error shapes, and the
full command surface live in 'clier --help' and 'clier <command>
--help' — those pages are the source of truth, this tutorial does
not restate them.

================================================================
The flow
================================================================

(optional) clier auth login
   Browsing the catalog and starting a run work without a session.
   Log in only when you want to author teams in your own namespace
   or star teams.

1. clier team list
   Browse the catalog. The demo team is jakeraft/hello-clier.

2. clier run start jakeraft/hello-clier
   Note the run_id printed on stdout — every following step takes
   that id.

3. clier run tell --run <run-id> --to jakeraft.hello-clier <<'EOF'
   Greet your peer and tell me what you learned.
   EOF

   *Give the agent its first task.* Without 'tell' the agent just
   sits there — clier never injects a task on its own. Every run
   starts idle and waits for you.

4. clier run capture <run-id>
   Read what landed in the agent's pane after the tell — JSON, no
   tmux attach required. Pair this with 'tell' to drive the agent
   without an interactive terminal.

5. clier run attach <run-id>
   When you do want to watch live, attach. Detach with Ctrl-b d.

6. clier run stop <run-id>
   Tear it down. tmux session, clones, run.json — all gone.

================================================================
Going further
================================================================

Author your own team:    clier team create --help
Whole command surface:   clier --help

Errors print on stderr starting with 'error: ' and exit non-zero;
success commands print one JSON object on stdout. Every command's
--help documents its own flags and the violations it can surface.`,
	}
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return c.Help()
	}
	return cmd
}
