package cmd

import (
	"errors"
	"io"
	"os"
	"strings"

	agentrt "github.com/jakeraft/clier/internal/adapter/runtime"
	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/run"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage runs",
	}
	cmd.AddCommand(newRunStartCmd())
	cmd.AddCommand(newRunStopCmd())
	cmd.AddCommand(newRunListCmd())
	cmd.AddCommand(newRunTellCmd())
	cmd.AddCommand(newRunNoteCmd())
	cmd.AddCommand(newRunNotesCmd())
	cmd.AddCommand(newRunMessagesCmd())
	cmd.AddCommand(newRunAttachCmd())
	return cmd
}

func newRunStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "start <team-id>",
		Short:       "Start a run",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store := newStore()

			t, err := store.GetTeam(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			runtimes := map[string]run.AgentRuntime{
				"claude": &agentrt.ClaudeRuntime{},
			}
			svc := run.New(store, term, ws, cfg.Paths.Workspaces(), runtimes)

			r, err := svc.Start(cmd.Context(), t)
			if err != nil {
				return err
			}
			return printJSON(r)
		},
	}
	return cmd
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "stop <id>",
		Short:       "Stop a run",
		Annotations: map[string]string{mutates: "true"},
		Args:        cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store := newStore()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			runtimes := map[string]run.AgentRuntime{
				"claude": &agentrt.ClaudeRuntime{},
			}
			svc := run.New(store, term, ws, cfg.Paths.Workspaces(), runtimes)

			if err := svc.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			return printJSON(map[string]string{"stopped": args[0]})
		},
	}
}

func newRunListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()

			runs, err := client.ListRuns()
			if err != nil {
				return err
			}
			return printJSON(runs)
		},
	}
}

func newRunTellCmd() *cobra.Command {
	var runFlag, toMemberID string

	cmd := &cobra.Command{
		Use:   "tell [content]",
		Short: "Tell a teammate",
		Long: `Tell a teammate. Content can be provided as an argument or via stdin.

Examples:
  clier run tell --to <id> "simple message"
  echo "message with special chars" | clier run tell --to <id>
  clier run tell --to <id> <<'EOF'
  message with ` + "`backticks`" + ` and --flags
  EOF`,
		Args:        cobra.MaximumNArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, fromMemberID, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store := newStore()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			runtimes := map[string]run.AgentRuntime{
				"claude": &agentrt.ClaudeRuntime{},
			}
			svc := run.New(store, term, ws, cfg.Paths.Workspaces(), runtimes)

			if err := svc.Send(cmd.Context(), runID, fromMemberID, toMemberID, content); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to CLIER_RUN_ID)")
	cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newRunNoteCmd() *cobra.Command {
	var runFlag string

	cmd := &cobra.Command{
		Use:         "note [content]",
		Short:       "Post a progress note",
		Args:        cobra.MaximumNArgs(1),
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, memberID, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store := newStore()

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			runtimes := map[string]run.AgentRuntime{
				"claude": &agentrt.ClaudeRuntime{},
			}
			svc := run.New(store, term, ws, cfg.Paths.Workspaces(), runtimes)

			if err := svc.Note(cmd.Context(), runID, memberID, content); err != nil {
				return err
			}
			return printJSON(map[string]string{
				"status": "posted",
				"member": memberID,
				"run":    runID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to CLIER_RUN_ID)")
	return cmd
}

func newRunNotesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notes <run-id>",
		Short: "List run notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()

			notes, err := client.ListNotes(args[0])
			if err != nil {
				return err
			}
			return printJSON(notes)
		},
	}
}

func newRunMessagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "messages <run-id>",
		Short: "List run messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()

			msgs, err := client.ListMessages(args[0])
			if err != nil {
				return err
			}
			return printJSON(msgs)
		},
	}
}

func newRunAttachCmd() *cobra.Command {
	var memberFlag string

	cmd := &cobra.Command{
		Use:   "attach <run-id>",
		Short: "Attach to a running run's terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := newStore()

			term := terminal.NewTmuxTerminal(store)

			var memberID *string
			if memberFlag != "" {
				memberID = &memberFlag
			}
			return term.Attach(args[0], memberID)
		},
	}
	cmd.Flags().StringVar(&memberFlag, "member", "", "Attach to a specific member's window")
	return cmd
}

// readContent returns content from args[0] or stdin when no argument is given.
func readContent(args []string) (string, error) {
	if len(args) > 0 && args[0] != "-" {
		return args[0], nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(b))
	if content == "" {
		return "", errors.New("no content provided (pass as argument or pipe via stdin)")
	}
	return content, nil
}

// resolveRunContext resolves run ID and member ID from env vars set by clier.
// CLIER_RUN_ID identifies the run, CLIER_MEMBER_ID identifies the sender.
func resolveRunContext(runFlag string) (runID, memberID string, err error) {
	runID = runFlag
	if runID == "" {
		runID = os.Getenv("CLIER_RUN_ID")
	}
	if runID == "" {
		return "", "", errors.New("--run flag or CLIER_RUN_ID must be set")
	}
	memberID = os.Getenv("CLIER_MEMBER_ID")
	return runID, memberID, nil
}
