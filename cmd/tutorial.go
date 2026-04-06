package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/terminal"
	"github.com/jakeraft/clier/internal/adapter/workspace"
	"github.com/jakeraft/clier/internal/app/task"
	"github.com/spf13/cobra"
)

const (
	tutorialImportURL    = "https://raw.githubusercontent.com/jakeraft/clier/main/tutorials/todo-team"
	tutorialTeamID       = "d4040404-0001-4000-8000-000000000001"       // Source: tutorials/todo-team/15-team.json
	tutorialRootMemberID = "d4040404-aa01-4000-8000-000000000001"       // Source: tutorials/todo-team/15-team.json
	tutorialMessage      = "Add a list --done flag to filter completed todos."
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tutorial",
		Short: "Learn the clier workflow with an example team",
		Long: `Learn the clier workflow with an example team.

This tutorial uses the "todo-team" — a team of AI agents that fixes
bugs and audits a deliberately incomplete Todo CLI app:

  tech-lead (root)
  ├── coder → reviewer    (fix track: implement feature → code review)
  └── auditor             (audit track: review codebase → file GitHub issues)

The todo app (github.com/jakeraft/clier_todo) has intentional bugs:
  - list: status always shows "[ ]" regardless of done value
  - done: sets done=0 instead of done=1
  - delete: no existence check

Run "clier tutorial start" to kick off the team:

  1. Import the todo-team definition
  2. Start a task (launches all agents in tmux)
  3. Tell the tech-lead: "` + tutorialMessage + `"

The coder implements the feature and gets it reviewed,
while the auditor independently discovers existing bugs
and files GitHub issues — self-healing in action.

Monitor progress:

  clier task notes <task-id>       Check agent progress notes
  clier task attach <task-id>      Watch agents work in real time

When the task is done:

  clier task stop <task-id>        Stop all agents
  gh issue list -R jakeraft/clier_todo   See issues the auditor filed
  cd ~/.clier/workspaces/<task-id>/<member-id>/project && git log   See commits`,
	}
	cmd.AddCommand(newTutorialStartCmd())
	return cmd
}

func newTutorialStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "start",
		Short:       "Run the tutorial",
		Annotations: map[string]string{mutates: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			ctx := cmd.Context()

			// Step 1: Import todo-team.
			fmt.Fprintln(os.Stderr, "Step 1/3: Importing todo-team...")
			src := strings.TrimRight(tutorialImportURL, "/") + "/index.json"
			data, err := readSource(src)
			if err != nil {
				return fmt.Errorf("fetch tutorial: %w", err)
			}

			var idx indexFile
			if err := json.Unmarshal(data, &idx); err != nil {
				return fmt.Errorf("parse index.json: %w", err)
			}
			base := basePath(src)
			for _, f := range idx.Files {
				fileSrc := joinPath(base, f)
				fileData, err := readSource(fileSrc)
				if err != nil {
					return fmt.Errorf("read %s: %w", fileSrc, err)
				}
				if err := importEnvelope(ctx, store, fileData); err != nil {
					return fmt.Errorf("import %s: %w", f, err)
				}
			}

			// Step 2: Start task.
			fmt.Fprintln(os.Stderr, "Step 2/3: Starting task...")
			t, err := store.GetTeam(ctx, tutorialTeamID)
			if err != nil {
				return fmt.Errorf("get team: %w", err)
			}

			term := terminal.NewTmuxTerminal(store)
			ws := workspace.New(cfg.Paths.Workspaces())
			svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir())

			tk, err := svc.Start(ctx, t, cfg.Auth)
			if err != nil {
				return fmt.Errorf("start task: %w", err)
			}

			// Step 3: Tell root member to start.
			fmt.Fprintln(os.Stderr, "Step 3/3: Telling tech-lead to start...")
			if err := svc.Send(ctx, tk.ID, "", tutorialRootMemberID, tutorialMessage); err != nil {
				return fmt.Errorf("tell message: %w", err)
			}

			fmt.Fprintln(os.Stderr, "\nTutorial task started successfully.")
			fmt.Fprintln(os.Stderr, "\nNext steps:")
			fmt.Fprintf(os.Stderr, "  clier task notes %s\n", tk.ID)
			fmt.Fprintf(os.Stderr, "  clier task attach %s\n", tk.ID)

			return printJSON(tk)
		},
	}
}
