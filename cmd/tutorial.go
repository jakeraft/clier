package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tutorial",
		Short: "Learn the clier workflow with an example team",
		Long: `Learn the clier workflow with an example team.

The "todo-team" is a team of AI agents that implements a feature
on a real GitHub repo (github.com/jakeraft/clier_todo) with
PR-based code review:

  tech-lead (root)
  └── coder → reviewer    (implement → PR → review loop)

Follow the steps below to try it out.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Step 1. Import the todo-team resources

  clier import tutorials/todo-team

  This loads all building blocks (agent-dot-mds, skills, settings,
  members, and the team) into the local database.

Step 2. Check what was imported

  clier team list
  clier member list

Step 3. Start a task

  clier task start <team-id>

  This clones the git repo for each member, sets up workspaces,
  and launches all agents in tmux. Copy the task ID from the output.

Step 4. Give the team a job

  clier task tell --task <task-id> --to <root-member-id> \
    "Add a list --done flag to filter completed todos."

  The tech-lead plans the work, the coder implements it on a branch,
  creates a PR, and the reviewer iterates on it until approved.
  The tech-lead writes a final report on the PR.

Step 5. Watch them work

  clier task attach <task-id>        Watch agents in real time
  clier task notes <task-id>         Check progress notes
  clier dashboard                    Open the dashboard

Step 6. When done, stop the task

  clier task stop <task-id>

Step 7. See the result

  gh pr list -R jakeraft/clier_todo
  gh pr view <number> -R jakeraft/clier_todo --web

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Tip: Use "clier <command> --help" for details on each command.`,
	}
}
