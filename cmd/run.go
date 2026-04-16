package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
		Short:   "Manage running agent sessions",
		GroupID: rootGroupRuntime,
		Long: `Start, stop, and interact with agents running in tmux.

Use start to launch agents, tell to send them instructions,
and attach to watch them work.`,
		RunE: subcommandRequired,
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
				return printJSON([]*apprun.RunPlan{})
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
		Short: "Launch the current working copy in tmux",
		Long: `Start all agents in the current working copy. Works with both
leaf teams (single agent) and composite teams (multiple agents).

Agents start idle. Use run tell to send them instructions.`,
		Args: cobra.NoArgs,
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

			agents, err := collectRunnableAgents(manifest)
			if err != nil {
				return err
			}

			runName := sessionName(manifest.Name, runID)
			var terminalPlans []apprun.AgentTerminal
			for i, agent := range agents {
				agentBase := filepath.Join(copyRoot, filepath.FromSlash(agent.LocalBase))
				envVars := buildAgentEnv(runID, agent.ID, appworkspace.ResourceID(manifest.Owner, manifest.Name))
				fullCommand := buildFullCommand(envVars, agent.Projection.Command, agentBase)
				terminalPlans = append(terminalPlans, apprun.AgentTerminal{
					ID:        agent.ID,
					Name:      agent.Name,
					AgentType: agent.Projection.AgentType,
					Window:    i,
					Workspace: agentBase,
					Cwd:       agentBase,
					Command:   fullCommand,
				})
			}
			runner := apprun.NewRunner(newTerminal())
			plan, err := runner.Run(copyRoot, runID, runName, terminalPlans)
			if err != nil {
				return err
			}
			return printJSON(map[string]any{"run_id": runID, "session": plan.Session})
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

			store, err := newPlanStore()
			if err != nil {
				return err
			}
			svc := apprun.New(newTerminal(), store)

			if err := svc.Stop(plan); err != nil {
				return err
			}

			return printJSON(map[string]string{"stopped": plan.RunID})
		},
	}
}

func newRunAttachCmd() *cobra.Command {
	var agentFlag string

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

			var agentName *string
			if agentFlag != "" {
				agentName = &agentFlag
			}
			return term.Attach(plan, agentName)
		},
	}
	cmd.Flags().StringVar(&agentFlag, "agent", "", "Attach to a specific agent ID (owner/name)")
	return cmd
}

func newRunTellCmd() *cobra.Command {
	var runFlag string
	var toAgentName string

	cmd := &cobra.Command{
		Use:   "tell [content]",
		Short: "Send a message to an agent",
		Long: `Send a message to another agent in the current team run.
Content can be provided as an argument or via stdin.

Examples:
  clier run tell --to <owner/name> "simple message"
  echo "message with special chars" | clier run tell --to <owner/name>
  clier run tell --to <owner/name> <<'EOF'
  message with ` + "`backticks`" + ` and --flags
  EOF`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := readContent(args)
			if err != nil {
				return err
			}

			runID, fromAgent, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}
			plan, err := resolveRunPlan(runID)
			if err != nil {
				return err
			}

			var toAgent *string
			if toAgentName != "" {
				toAgent = &toAgentName
			}

			store, err := newPlanStore()
			if err != nil {
				return err
			}
			svc := apprun.New(newTerminal(), store)

			if err := svc.Send(plan, fromAgent, toAgent, content); err != nil {
				return err
			}

			return printJSON(map[string]any{
				"status": "delivered",
				"from":   fromAgent,
				"to":     toAgent,
				"run":    runID,
			})
		},
	}
	cmd.Flags().StringVar(&runFlag, "run", "", "Run ID (defaults to "+envClierRunID+")")
	cmd.Flags().StringVar(&toAgentName, "to", "", "Recipient agent ID (owner/name)")
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

			runID, agentName, err := resolveRunContext(runFlag)
			if err != nil {
				return err
			}

			plan, err := resolveRunPlan(runID)
			if err != nil {
				return err
			}

			store, err := newPlanStore()
			if err != nil {
				return err
			}
			svc := apprun.New(newTerminal(), store)

			if err := svc.Note(plan, agentName, content); err != nil {
				return err
			}

			var agentVal any
			if agentName != nil {
				agentVal = *agentName
			}
			return printJSON(map[string]any{
				"status": "posted",
				"agent":  agentVal,
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

// resolveRunContext resolves run ID and agent ID from env vars set by clier.
func resolveRunContext(runFlag string) (runID string, agentName *string, err error) {
	runID = strings.TrimSpace(runFlag)
	if runID == "" {
		runID = strings.TrimSpace(os.Getenv(envClierRunID))
	}
	if runID == "" {
		return "", nil, fmt.Errorf("--run flag or %s must be set", envClierRunID)
	}
	if raw := strings.TrimSpace(os.Getenv(envClierAgentName)); raw != "" {
		agentName = &raw
	}
	return runID, agentName, nil
}
