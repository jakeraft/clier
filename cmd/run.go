package cmd

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jakeraft/clier/cmd/present"
	"github.com/jakeraft/clier/cmd/view"
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
			repo, err := newRunRepository()
			if err != nil {
				return err
			}
			runs, err := repo.List()
			if err != nil {
				return err
			}
			slices.SortFunc(runs, func(a, b *apprun.RunPlan) int {
				return b.StartedAt.Compare(a.StartedAt)
			})
			return present.Success(cmd.OutOrStdout(), view.RunListOf(runs))
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

Vendor CLIs may still show their own approval prompts in their pane on
first launch. Use "clier run attach <run-id>" from a normal terminal
when you need to inspect or approve those prompts.`,
		Args: requireOneArg("clier run start <owner/name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := splitResourceID(args[0])
			if err != nil {
				return err
			}
			if err := validateOwner(owner); err != nil {
				return err
			}
			base, err := workingCopyPath(owner, name)
			if err != nil {
				return err
			}

			fs := newFileMaterializer()
			manifest, err := appworkspace.LoadManifest(fs, base)
			if err != nil {
				return classifyWorkingCopyError(owner, name, base, err)
			}
			if err := appworkspace.ValidateWorkingCopy(base, manifest, fs, newGitRepo()); err != nil {
				return err
			}
			if err := rejectIfRunActive(base); err != nil {
				return err
			}

			runID, err := newRunID()
			if err != nil {
				return err
			}

			agents, err := appworkspace.CollectRunnableAgents(manifest)
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
			repo, err := newRunRepository()
			if err != nil {
				return err
			}
			runner := apprun.NewRunner(newTerminal(), repo)
			plan, err := runner.Run(base, runID, runName, terminalPlans)
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.RunStartOf(runID, plan.Session))
		},
	}
}

func newRunViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <run-id>",
		Short: "Show run status and notes",
		Args:  requireOneArg("clier run view <run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}
			return present.Success(cmd.OutOrStdout(), view.RunDetailOf(plan))
		},
	}
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Stop a running session",
		Args:  requireOneArg("clier run stop <run-id>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := resolveRunPlan(args[0])
			if err != nil {
				return err
			}

			svc, err := newRunOrchestrator()
			if err != nil {
				return err
			}

			if err := svc.Stop(plan); err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.RunStopOf(plan.RunID))
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
		Args: requireOneArg("clier run attach <run-id>"),
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

			svc, err := newRunOrchestrator()
			if err != nil {
				return err
			}

			if err := svc.Send(plan, fromAgent, toAgent, content); err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.RunTellOf(runID, fromAgent, toAgent))
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

			svc, err := newRunOrchestrator()
			if err != nil {
				return err
			}

			if err := svc.Note(plan, agentName, content); err != nil {
				return err
			}

			return present.Success(cmd.OutOrStdout(), view.RunNoteOf(runID, agentName))
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
