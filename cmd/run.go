package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/terminal"
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
	cmd.AddCommand(newRunListCmd())
	cmd.AddCommand(newRunViewCmd())
	cmd.AddCommand(newRunStopCmd())
	cmd.AddCommand(newRunAttachCmd())
	cmd.AddCommand(newRunTellCmd())
	cmd.AddCommand(newRunNoteCmd())
	return cmd
}

func newRunListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = requireLogin()
			client := newAPIClient()
			runs, err := client.ListRuns()
			if err != nil {
				return err
			}
			return printJSON(runs)
		},
	}
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <id>",
		Short: "View a run (includes notes and messages)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = requireLogin()
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run id %q: %w", args[0], err)
			}
			client := newAPIClient()
			resp, err := client.GetRun(runID)
			if err != nil {
				return err
			}
			return printJSON(resp)
		},
	}
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run id %q: %w", args[0], err)
			}

			store := newStore()
			term := terminal.NewTmuxTerminal(newRefStore())
			svc := run.New(store, term)

			if err := svc.Stop(cmd.Context(), runID); err != nil {
				return err
			}
			return printJSON(map[string]int64{"stopped": runID})
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
			term := terminal.NewTmuxTerminal(newRefStore())

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

func newRunTellCmd() *cobra.Command {
	var runFlag string
	var toMemberIDRaw int64

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
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, fromMemberID, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			toMemberID := &toMemberIDRaw

			store := newStore()
			term := terminal.NewTmuxTerminal(newRefStore())
			svc := run.New(store, term)

			if err := svc.Send(cmd.Context(), runID, fromMemberID, toMemberID, content); err != nil {
				return err
			}
			return printJSON(map[string]any{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to CLIER_RUN_ID)")
	cmd.Flags().Int64Var(&toMemberIDRaw, "to", 0, "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newRunNoteCmd() *cobra.Command {
	var runFlag string

	cmd := &cobra.Command{
		Use:   "note [content]",
		Short: "Post a progress note",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, memberID, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			store := newStore()
			term := terminal.NewTmuxTerminal(newRefStore())
			svc := run.New(store, term)

			if err := svc.Note(cmd.Context(), runID, memberID, content); err != nil {
				return err
			}
			var memberVal any
			if memberID != nil {
				memberVal = *memberID
			}
			return printJSON(map[string]any{
				"status": "posted",
				"member": memberVal,
				"run":    runID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to CLIER_RUN_ID)")
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
func resolveRunContext(runFlag string) (runID int64, memberID *int64, err error) {
	rawRunID := runFlag
	if rawRunID == "" {
		rawRunID = os.Getenv("CLIER_RUN_ID")
	}
	if rawRunID == "" {
		return 0, nil, errors.New("--run flag or CLIER_RUN_ID must be set")
	}
	runID, err = strconv.ParseInt(rawRunID, 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("run id is not a valid int64: %w", err)
	}
	if raw := os.Getenv("CLIER_MEMBER_ID"); raw != "" {
		v, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			return 0, nil, fmt.Errorf("CLIER_MEMBER_ID is not a valid int64: %w", parseErr)
		}
		memberID = &v
	}
	return runID, memberID, nil
}
