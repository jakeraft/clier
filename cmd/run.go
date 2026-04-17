package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
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
		Short: "List runs across all working copies",
		RunE: func(cmd *cobra.Command, args []string) error {
			runs, err := apprun.ListPlans(runsDir())
			if err != nil {
				return err
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
		Use:   "start <owner/name>",
		Short: "Launch a working copy in tmux",
		Long: `Start all agents for the working copy at
<workspace_dir>/<owner>.<name>/.

Agents start idle. Use run tell to send them instructions.

On the first start in a fresh working copy, the JSON output includes
a one-time "hint" field. Vendor CLIs (e.g., Codex) may show their
own approval prompts in their pane on first launch. clier does not
modify vendor configs on your behalf — ask the user to run
"clier run attach <run-id>" from a normal terminal, approve those
prompts, and detach (Ctrl-b d) before sending messages.`,
		Args: requireExactArgs(1, "clier run start <owner/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}
			base := workingCopyPath(owner, name)

			fs := newFileMaterializer()
			manifest, err := appworkspace.LoadManifest(fs, base)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			if err := validateWorkingCopy(base, manifest); err != nil {
				return err
			}
			if err := rejectIfRunActive(base); err != nil {
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
				agentBase := filepath.Join(base, filepath.FromSlash(agent.LocalBase))
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
			plan, err := runner.Run(runsDir(), base, runID, runName, terminalPlans)
			if err != nil {
				return err
			}

			result := map[string]any{"run_id": runID, "session": plan.Session}
			if hint, mark := firstRunHint(manifest, runID); hint != "" {
				result[hintField] = hint
				manifest.FirstRunAt = mark
				// Best-effort persist: if the manifest write fails, the
				// hint reappears on the next run — no data loss, just a
				// duplicate hint. Failing the run for this would be worse.
				_ = appworkspace.SaveManifest(fs, base, manifest)
			}
			return printJSON(result)
		},
	}
}

// hintField is the JSON key for the optional next-step hint that
// commands may emit when state warrants it. Shared by run.go output,
// tutorial / docs references, and the root help convention.
const hintField = "hint"

// firstRunHint returns the one-time hint and the timestamp to mark on
// the manifest when this is the workspace's first start. Returns ("", nil)
// if FirstRunAt is already set.
func firstRunHint(manifest *appworkspace.Manifest, runID string) (string, *time.Time) {
	if manifest.FirstRunAt != nil {
		return "", nil
	}
	now := time.Now().UTC()
	text := fmt.Sprintf(
		"First start in this workspace. Before sending messages, ask your user to run 'clier run attach %s' from a normal terminal and approve any one-time vendor prompts (e.g., Codex's directory trust), then detach (Ctrl-b d).",
		runID,
	)
	return text, &now
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <run-id>",
		Short: "Show run status and notes",
		Args:  requireExactArgs(1, "clier run view <run-id>"),
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
		Use:   "stop <run-id>",
		Short: "Stop a running session",
		Args:  requireExactArgs(1, "clier run stop <run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}

			svc := apprun.New(newTerminal(), newPlanStore())

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
		Short: "Watch and interact with agents in real time",
		Long: `Attach to the tmux session for an active run.

attach is interactive — you can observe agents and type into their
panes. Use it to approve any one-time vendor prompts (e.g., Codex's
directory trust) on a workspace's first start, then detach with
Ctrl-b d.

This command is intended for use from a normal user terminal.
It is not supported when clier itself is running inside an agent
environment.

Verifying without attaching:
  When attach can't be used (running inside another tmux session,
  CI, automated QA), inspect any agent pane non-interactively:
    tmux capture-pane -p -t <session>:<window>
  Use 'clier run view <run-id>' to look up the session name and
  per-agent window indices.`,
		Args: requireExactArgs(1, "clier run attach <run-id>"),
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
		Long: `Send a message to another agent in a run.
Content can be provided as an argument or via stdin.

Examples:
  clier run tell --run <run-id> --to <owner/name> "simple message"
  echo "message with special chars" | clier run tell --run <run-id> --to <owner/name>
  clier run tell --run <run-id> --to <owner/name> <<'EOF'
  message with ` + "`backticks`" + ` and --flags
  EOF`,
		Args: requireMaxArgs(1, "clier run tell --run <run-id> --to <owner/name> [content]"),
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

			svc := apprun.New(newTerminal(), newPlanStore())

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
		Long: `Record a work log entry in a run.

Content can be provided as an argument or via stdin. The note is
appended to the run file under <workspace_dir>/.runs/.`,
		Args: requireMaxArgs(1, "clier run note [content]"),
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

			svc := apprun.New(newTerminal(), newPlanStore())

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
		return "", &domain.Fault{Kind: domain.KindContentRequired}
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
		return "", nil, &domain.Fault{
			Kind:    domain.KindRunIDRequired,
			Subject: map[string]string{"env": envClierRunID},
		}
	}
	if raw := strings.TrimSpace(os.Getenv(envClierAgentName)); raw != "" {
		agentName = &raw
	}
	return runID, agentName, nil
}
