package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Observe and control running agents",
		GroupID: rootGroupRuntime,
		Long:    `Start, stop, and interact with agents running in tmux.`,
	}
	cmd.AddCommand(newRunStartCmd())
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
		Short: "List active runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeDir, err := resolveRuntimeDir()
			if err != nil {
				return err
			}
			if runtimeDir == "" {
				return printJSON([]*apprun.State{})
			}

			entries, err := os.ReadDir(runtimeDir)
			if err != nil {
				return fmt.Errorf("read runtime dir: %w", err)
			}

			runs := make([]*apprun.RunPlan, 0)
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".state.json") || entry.Name() == appworkspace.ManifestFile {
					continue
				}
				plan, err := apprun.LoadPlanFromPath(filepath.Join(runtimeDir, entry.Name()))
				if err != nil {
					return err
				}
				runs = append(runs, plan)
			}
			slices.SortFunc(runs, func(a, b *apprun.RunPlan) int {
				return b.StartedAt.Compare(a.StartedAt)
			})
			return printJSON(runs)
		},
	}
}

func newRunStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the current local clone in tmux",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			base, err := resolveCurrentDir()
			if err != nil {
				return err
			}
			fs := newFileMaterializer()
			copyRoot, manifest, err := appworkspace.FindManifestAbove(fs, base)
			if err != nil {
				if os.IsNotExist(err) {
					return errNotInWorkingCopy()
				}
				return err
			}
			if err := validateWorkingCopy(copyRoot, manifest); err != nil {
				return err
			}

			runID, err := newRunID()
			if err != nil {
				return err
			}

			switch manifest.Kind {
			case string(api.KindMember):
				memberProjection, err := appworkspace.LoadMemberProjection(fs, appworkspace.MemberProjectionPath(copyRoot))
				if err != nil {
					return err
				}
				member := manifest.Runtime.Member
				runName := apprun.SessionName(member.Name, runID)
				envVars := buildMemberEnv(runID, member.ID, nil, member.Name)
				fullCommand := buildFullCommand(envVars, memberProjection.Command, copyRoot)
				terminalPlans := []apprun.MemberTerminal{{
					TeamMemberID: member.ID,
					Name:         member.Name,
					AgentType:    member.AgentType,
					Window:       0,
					Memberspace:  copyRoot,
					Cwd:          copyRoot,
					Command:      fullCommand,
				}}
				runner := apprun.NewRunner(newTerminal())
				plan, err := runner.Run(copyRoot, runID, runName, terminalPlans)
				if err != nil {
					return err
				}
				return printJSON(map[string]any{"run_id": runID, "session": plan.Session})
			case string(api.KindTeam):
				team := manifest.Runtime.Team
				runName := apprun.SessionName(team.Name, runID)
				var terminalPlans []apprun.MemberTerminal
				for i, member := range team.Members {
					memberProjection, err := appworkspace.LoadMemberProjection(fs, appworkspace.TeamMemberProjectionPath(copyRoot, member.Name))
					if err != nil {
						return err
					}
					memberBase := filepath.Join(copyRoot, member.Name)
					envVars := buildMemberEnv(runID, member.TeamMemberID, &team.ID, member.Name)
					fullCommand := buildFullCommand(envVars, memberProjection.Command, memberBase)
					terminalPlans = append(terminalPlans, apprun.MemberTerminal{
						TeamMemberID: member.TeamMemberID,
						Name:         member.Name,
						AgentType:    member.AgentType,
						Window:       i,
						Memberspace:  memberBase,
						Cwd:          memberBase,
						Command:      fullCommand,
					})
				}
				runner := apprun.NewRunner(newTerminal())
				plan, err := runner.Run(copyRoot, runID, runName, terminalPlans)
				if err != nil {
					return err
				}
				return printJSON(map[string]any{"run_id": runID, "session": plan.Session})
			default:
				return fmt.Errorf("unsupported working-copy kind %q", manifest.Kind)
			}
		},
	}
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <id>",
		Short: "Show run status and notes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}
			return printJSON(plan)
		},
	}
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <id>",
		Short: "Stop a running session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}

			svc := apprun.New(newTerminal())

			if err := svc.Stop(plan); err != nil {
				return err
			}

			plan.MarkStopped()
			if err := saveRunPlan(plan.RunID, plan); err != nil {
				return err
			}

			return printJSON(map[string]string{"stopped": plan.RunID})
		},
	}
}

func newRunAttachCmd() *cobra.Command {
	var memberFlag string

	cmd := &cobra.Command{
		Use:   "attach <run-id>",
		Short: "Watch agents in real time",
		Long: `Attach to the tmux session for an active run.

This command is intended for use from a normal user terminal.
It is not supported when clier itself is running inside an agent
environment.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			term := newTerminal()
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}

			var memberID *int64
			if memberFlag != "" {
				parsed, err := strconv.ParseInt(memberFlag, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid member id %q: %w", memberFlag, err)
				}
				memberID = &parsed
			}
			return term.Attach(plan, memberID)
		},
	}
	cmd.Flags().StringVar(&memberFlag, "member", "", "Attach to a specific team member ID")
	return cmd
}

func newRunTellCmd() *cobra.Command {
	var runFlag string
	var toMemberIDRaw int64

	cmd := &cobra.Command{
		Use:   "tell [content]",
		Short: "Send a message to an agent",
		Long: `Send a message to another member in the current team run.
Content can be provided as an argument or via stdin.

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
			plan, err := resolveRunPlan(runID)
			if err != nil {
				return err
			}

			toMemberID := &toMemberIDRaw

			svc := apprun.New(newTerminal())

			if err := svc.Send(plan, fromMemberID, toMemberID, content); err != nil {
				return err
			}

			if err := plan.AddMessage(fromMemberID, toMemberID, content); err != nil {
				return err
			}
			if err := saveRunPlan(runID, plan); err != nil {
				return err
			}

			return printJSON(map[string]any{
				"status": "delivered",
				"from":   fromMemberID,
				"to":     toMemberID,
				"run":    runID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to "+envClierRunID+")")
	cmd.Flags().Int64Var(&toMemberIDRaw, "to", 0, "Recipient member ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newRunNoteCmd() *cobra.Command {
	var runFlag string

	cmd := &cobra.Command{
		Use:   "note [content]",
		Short: "Record a progress note",
		Long: `Record a work log entry in the current run.

Content can be provided as an argument or via stdin. The note is
appended to the run file under ` + "`.clier/`" + `.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, memberID, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			svc := apprun.New(newTerminal())

			if err := svc.Note(memberID, content); err != nil {
				return err
			}

			plan, err := resolveRunPlan(runID)
			if err != nil {
				return err
			}
			if err := plan.AddNote(memberID, content); err != nil {
				return err
			}
			if err := saveRunPlan(runID, plan); err != nil {
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
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to "+envClierRunID+")")
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
func resolveRunContext(runFlag string) (runID string, memberID *int64, err error) {
	runID = strings.TrimSpace(runFlag)
	if runID == "" {
		runID = strings.TrimSpace(os.Getenv(envClierRunID))
	}
	if runID == "" {
		return "", nil, fmt.Errorf("--run flag or %s must be set", envClierRunID)
	}
	if raw := os.Getenv(envClierMemberID); raw != "" {
		v, parseErr := apprun.ParseTeamMemberID(raw)
		if parseErr != nil {
			return "", nil, fmt.Errorf("%s is not a valid int64: %w", envClierMemberID, parseErr)
		}
		memberID = &v
	}
	return runID, memberID, nil
}
