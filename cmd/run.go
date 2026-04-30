package cmd

import (
	"github.com/jakeraft/clier/internal/runplan"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start, message, and stop tmux runs",
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
		},
	}
	cmd.AddCommand(
		newRunStartCmd(),
		newRunTellCmd(),
		newRunAttachCmd(),
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
		Args:  cobra.ExactArgs(1),
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
			return emit(cmd.OutOrStdout(), startResponse(plan))
		},
	}
}

func newRunTellCmd() *cobra.Command {
	var runFlag, fromFlag, toFlag string
	cmd := &cobra.Command{
		Use:   "tell --run <id> --to <agent-id> [--from <id>] [content]",
		Short: "Send a message to an agent",
		Long: `Send a message to an agent in a running tmux session.

The protocol markdown the server emits at run start already embeds the
fully-qualified ` + "`clier run tell --run <run-id> --to <peer>`" + ` invocation, so an
agent inside the run can copy/paste that line verbatim.`,
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
		Args:  cobra.ExactArgs(1),
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

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Kill the tmux session and free clones",
		Args:  cobra.ExactArgs(1),
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
				items = append(items, summary(p))
			}
			return emit(cmd.OutOrStdout(), map[string]any{"runs": items})
		},
	}
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <run-id>",
		Short: "Show full run state",
		Args:  cobra.ExactArgs(1),
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

func startResponse(plan *runplan.Plan) map[string]any {
	agents := make([]map[string]any, 0, len(plan.Agents))
	for _, a := range plan.Agents {
		agents = append(agents, map[string]any{
			"id":     a.ID,
			"window": a.Window,
			"cwd":    a.AbsCwd,
		})
	}
	return map[string]any{
		"run_id":       plan.RunID,
		"session_name": plan.SessionName,
		"run_dir":      plan.RunDir,
		"namespace":    plan.Namespace,
		"team_name":    plan.TeamName,
		"agents":       agents,
	}
}

func summary(p *runplan.Plan) map[string]any {
	return map[string]any{
		"run_id":       p.RunID,
		"session_name": p.SessionName,
		"namespace":    p.Namespace,
		"team_name":    p.TeamName,
		"status":       p.Status,
		"started_at":   p.StartedAt,
		"agent_count":  len(p.Agents),
	}
}
