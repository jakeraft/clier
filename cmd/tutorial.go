package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tutorial",
		Short:   "Walk through the hello-claude team",
		GroupID: rootGroupSettings,
		Long: fmt.Sprintf(`Walk through the built-in hello-claude team.

The "@clier/hello-claude" tutorial is the quickest way to verify that
clier can clone a team, launch members locally, pass messages between
them, and sync tracked changes.

The team has two members:

  hello-claude (root, Claude)
  └── hello-codex (child, Codex)

Follow the steps below to try it out.

Step 1. Log in

  clier auth login

  Authenticate with GitHub via device flow.

Step 2. Explore the pre-loaded hello-claude team

  clier list --kind team
  clier get @clier/hello-claude

  The "@clier/hello-claude" team is already available.

Step 3. Clone the team locally

  clier clone @clier/hello-claude
  cd @clier/hello-claude

  This downloads a local working copy with one directory per agent.

Step 4. Inspect the local working copy

  clier status
  clier run list

  You should see a clean working copy and no active runs yet.

Step 5. Start the team

  clier run start

  This launches both members in tmux from the current local clone.
  Note the run ID from the output.

  On the first start in a fresh working copy, the output includes a
  one-time %q field. Vendor CLIs (e.g., Codex) may show their own
  approval prompts in their pane on first launch. Run
  "clier run attach <run-id>" from your terminal, approve those
  prompts, and detach (Ctrl-b d) before sending messages in the
  next step.

Step 6. Ask hello-claude to have both members greet each other

  clier run tell --run <run-id> --to @clier/hello-claude \
    "Have both team members greet each other and report the result."

  A healthy run should show hello-claude contacting hello-codex,
  hello-codex replying, and hello-claude reporting the greeting result.

Step 7. Watch the run from the current local clone

  clier run attach <run-id>        Watch agents in real time
  clier run view <run-id>          Check progress notes and messages

  Note: run attach is intended for a normal user terminal.
  It is not supported when clier is running inside an agent.

Step 8. Verify the result

  Confirm all of the following:

  - both members participated
  - the greeting exchange completed
  - run view reflects the messages you observed

Step 9. Stop the run from the current local clone

  clier run stop <run-id>

Step 10. Edit a tracked file locally

  Resources you clone are tracked locally, similar to git.
  Edit the root agent instruction, then inspect local state:

    Open @clier/hello-claude/CLAUDE.md in your editor
    clier status                Check what changed

Step 11. Try sync flows

  From the same local clone, verify:

  - clean pull updates tracked files from the server
  - dirty pull refuses to overwrite local changes unless forced
  - push publishes your local tracked edits back to the server

  Use:

    clier pull
    clier pull --force
    clier push

Tip: Use "clier <command> --help" for details on each command.`, firstRunHintField),
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return c.Help()
	}
	return cmd
}
