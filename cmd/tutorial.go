package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tutorial",
		Short:   "Learn to harness your first agent team",
		GroupID: rootGroupSettings,
		Long: `Learn to harness your first agent team.

The "todo-team" is a team of AI agents that implements a feature
on a real GitHub repo (github.com/jakeraft/clier_todo) with
PR-based code review:

  tech-lead (root)
  └── coder → reviewer    (implement → PR → review loop)

Follow the steps below to try it out.

Step 1. Log in

  clier auth login

  Authenticate with GitHub via device flow.

Step 2. Explore the pre-loaded todo-team

  clier list --kind team
  clier get jakeraft/todo-team

  The "jakeraft/todo-team" is already available.

Step 3. Fork the team to your namespace

  clier fork jakeraft/todo-team

  This creates your own fork. Now you can customize it.

Step 4. Customize your fork

  Check your copied team and give it a summary:

    clier get <your-login>/todo-team
    clier edit todo-team --summary "My first agent team"

  Use --help on any command to see all available flags.

Step 5. Clone and start the team

  clier clone <your-login>/todo-team
  cd todo-team
  clier run start

  This downloads a local working copy under ./todo-team/
  and launches all agents in tmux.
  Note the run ID from the output.

Step 6. Give the team a job

  clier run tell --run <run-id> --to <root-member-id> \
    "Add a list --done flag to filter completed todos."

  The tech-lead plans the work, the coder implements it on a branch,
  creates a PR, and the reviewer iterates on it until approved.
  The tech-lead writes a final report on the PR.

Step 7. Watch them work from the current local clone

  clier run attach <run-id>        Watch agents in real time
  clier run view <run-id>          Check progress notes and messages

  Note: run attach is intended for a normal user terminal.
  It is not supported when clier is running inside an agent.

Step 8. When done, stop the run from the current local clone

  clier run stop <run-id>

Step 9. See the result

  gh pr list -R jakeraft/clier_todo
  gh pr view <number> -R jakeraft/clier_todo --web

Step 10. Edit and push local changes

  Resources you clone are tracked locally, just like git.
  Edit a member's prompt, then push the change to the server:

    edit coder/CLAUDE.md        Open in your editor
    clier status                Check what changed
    clier push                  Push local changes to the server

Tip: Use "clier <command> --help" for details on each command.`,
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return c.Help()
	}
	return cmd
}
