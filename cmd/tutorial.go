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

Step 1. Log in

  clier auth login

  Authenticate with GitHub via device flow.

Step 2. Explore the pre-loaded todo-team

  clier explore teams
  clier team view jakeraft/todo-team

  The "jakeraft/todo-team" is already available on the server.

Step 3. Fork and run the team

  clier team fork jakeraft/todo-team
  clier team run todo-team

  This forks the team to your namespace, creates workspaces
  for each member, and launches all agents in tmux.
  Copy the run ID from the output.

Step 4. Give the team a job

  clier run tell --run <run-id> --to <root-member-id> \
    "Add a list --done flag to filter completed todos."

  The tech-lead plans the work, the coder implements it on a branch,
  creates a PR, and the reviewer iterates on it until approved.
  The tech-lead writes a final report on the PR.

Step 5. Watch them work from the current workspace

  clier run attach <run-id>        Watch agents in real time
  clier run view <run-id>          Check progress notes and messages

Step 6. When done, stop the run from the current workspace

  clier run stop <run-id>

Step 7. See the result

  gh pr list -R jakeraft/clier_todo
  gh pr view <number> -R jakeraft/clier_todo --web

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Tip: Use "clier <command> --help" for details on each command.`,
	}
}
