package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start, message, and stop tmux runs",
		Args:  cobra.ArbitraryArgs,
		RunE:  helpOrUnknown,
	}
	cmd.AddCommand(
		newRunStartCmd(),
		newRunTellCmd(),
		newRunAttachCmd(),
		newRunCaptureCmd(),
		newRunStopCmd(),
		newRunListCmd(),
		newRunViewCmd(),
	)
	return cmd
}

func newRunStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <namespace/name>",
		Short: "Mint a run, clone the team, and launch in tmux",
		Long: `Mint a fresh run for the given team. Public — works
without a session.

The server walks the subteam graph, the CLI clones each agent's
repo into a per-run scratch dir under ~/.clier/runs/<run_id>/,
drops the rendered protocol markdown, and launches one tmux
window per agent.

The agents start *idle* — they do nothing until you send a task
with 'clier run tell'.`,
		Args: requireOneArg("<namespace/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, name, err := splitTeamID(args[0])
			if err != nil {
				return err
			}
			run, err := newRunner()
			if err != nil {
				return err
			}
			plan, err := run.Start(ns, name)
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), plan.StartView())
		},
	}
}

func newRunTellCmd() *cobra.Command {
	var runFlag, fromFlag, toFlag string
	cmd := &cobra.Command{
		Use:   "tell --run <id> --to <agent-id> [--from <id>] [content]",
		Short: "Send a message to an agent",
		Long: `Send a message to an agent in a running tmux session.

Required flags:
  --run <id>           run-id from 'run start'
  --to  <agent-id>     workspace-flat slug (namespace.name)

Optional flags:
  --from <agent-id>    sender slug — when present the recipient sees
                       a "[Message from <sender>]" prefix; lets agents
                       attribute peer messages.

Content can be passed as a positional arg (single line) or piped
on stdin (multi-line) — pass exactly one of the two. Both at the
same time is rejected.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}
			run, err := newRunner()
			if err != nil {
				return err
			}
			var fromPtr *string
			if fromFlag != "" {
				fromPtr = &fromFlag
			}
			if err := run.Tell(runFlag, fromPtr, toFlag, content); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"run_id": runFlag,
				"to":     toFlag,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID")
	cmd.Flags().StringVar(&toFlag, "to", "", "Recipient agent ID")
	cmd.Flags().StringVar(&fromFlag, "from", "", "Sender agent ID (optional)")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newRunAttachCmd() *cobra.Command {
	var agentFlag string
	cmd := &cobra.Command{
		Use:   "attach <run-id>",
		Short: "Attach to the tmux session",
		Long: `Hand control of the terminal to tmux. Detach with Ctrl-b d.

Optional flag:
  --agent <id>   select that agent's window before attaching

Requires an interactive TTY.`,
		Args: requireOneArg("<run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := newRunner()
			if err != nil {
				return err
			}
			var agentPtr *string
			if agentFlag != "" {
				agentPtr = &agentFlag
			}
			return run.Attach(args[0], agentPtr)
		},
	}
	cmd.Flags().StringVar(&agentFlag, "agent", "", "Select a specific agent's window before attaching")
	return cmd
}

func newRunCaptureCmd() *cobra.Command {
	var agentFlag string
	var linesFlag int
	cmd := &cobra.Command{
		Use:   "capture <run-id>",
		Short: "Snapshot the current tmux pane(s) for an agent",
		Long: `Read the live tmux pane buffer without attaching. Useful
when 'run tell' is followed by an automated check, or whenever a
TTY is unavailable (CI, scripted QA, headless agents).

Optional flags:
  --agent <id>   capture only this agent's pane (default: every agent)
  --lines <n>    include this many trailing scrollback lines`,
		Args: requireOneArg("<run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			// out-of-range bounds must be loud and named — `team list
			// --page-size -1` already rejects with the same shape;
			// silently accepting `--lines -1` made the surface
			// inconsistent (qa-checklist Trust 3).
			if linesFlag < 0 {
				return errors.New("invalid argument: --lines must be >= 0")
			}
			run, err := newRunner()
			if err != nil {
				return err
			}
			var agentPtr *string
			if agentFlag != "" {
				agentPtr = &agentFlag
			}
			items, err := run.Capture(args[0], agentPtr, linesFlag)
			if err != nil {
				return err
			}
			out := make([]map[string]any, len(items))
			for i, it := range items {
				out[i] = map[string]any{
					"agent_id":    it.AgentID,
					"window":      it.Window,
					"captured_at": it.CapturedAt,
					"content":     it.Content,
				}
			}
			return emit(cmd.OutOrStdout(), map[string]any{"data": out})
		},
	}
	cmd.Flags().StringVar(&agentFlag, "agent", "", "Capture only this agent's pane (defaults to every agent)")
	cmd.Flags().IntVar(&linesFlag, "lines", 0, "Include this many trailing scrollback lines (0 = visible area only)")
	return cmd
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Kill the tmux session and free clones",
		Long: `Tear down a run. Sends each agent's exit command, kills
the tmux session, and removes the entire ~/.clier/runs/<run-id>/
directory — clones, protocol files, and run.json all gone.

Stop is final and idempotent: running it on an already-stopped run
just clears any leftover dir. Once stopped, 'run list' and
'run view' no longer surface that run.`,
		Args: requireOneArg("<run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := newRunner()
			if err != nil {
				return err
			}
			if err := run.Stop(args[0]); err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), map[string]any{
				"run_id":  args[0],
				"stopped": true,
			})
		},
	}
}

func newRunListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List runs (newest first)",
		Long: `List every live run on this machine, newest first.

Local-only — reads ~/.clier/runs/, never calls the server. Stopped
runs do not appear (their dirs are removed).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			run, err := newRunner()
			if err != nil {
				return err
			}
			plans, err := run.List()
			if err != nil {
				return err
			}
			items := make([]map[string]any, 0, len(plans))
			for _, p := range plans {
				items = append(items, p.ListView())
			}
			// run list is a CLI-local query with no pagination — every
			// known run is in the response. team list ships {data, meta}
			// because it surfaces a server-paginated cursor; replicating
			// that shape here would attach a meaningless next_cursor to
			// every response. The shapes diverge intentionally — run
			// list is `{data}` only.
			return emit(cmd.OutOrStdout(), map[string]any{"data": items})
		},
	}
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <run-id>",
		Short: "Show full run state",
		Long: `Show the persisted plan for a live run — agents, windows,
recorded messages, started_at.

Local-only — reads ~/.clier/runs/<run-id>/run.json, never calls
the server. For live pane contents use 'run capture' instead.`,
		Args: requireOneArg("<run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := newRunner()
			if err != nil {
				return err
			}
			plan, err := run.View(args[0])
			if err != nil {
				return err
			}
			return emit(cmd.OutOrStdout(), plan)
		},
	}
}
