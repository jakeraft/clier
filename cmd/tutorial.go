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
		Long: fmt.Sprintf(`Walk through the built-in @clier/hello-claude team.

The tutorial has two phases:

  Phase 1 — Run the canned team to verify the install (Steps 1–10)
  Phase 2 — Fork it into your own namespace and iterate (Steps 11–19)

The team you start with:

  hello-claude (root, Claude)
  └── hello-codex (child, Codex)

clier owns the working-copy layout — every clone lives at
<workspace_dir>/<owner>.<name>/ (default workspace_dir is
~/.clier/workspace). Run subcommands work from any cwd; identify
working copies with <owner>/<name> and runs with their run-id.

================================================================
Phase 1 — Try the canned team
================================================================

Step 1. Log in

  clier auth login

  Authenticate with GitHub via device flow.

Step 2. See what's there in the dashboard

  clier open dashboard

  Opens the configured dashboard URL (default http://localhost:5173).
  This is the visual overview of teams, resources, and runs — get
  a feel for the whole picture before diving into the CLI. Change
  the URL with: clier config set dashboard-url <url>

Step 3. Explore the pre-loaded team from the CLI

  clier list --kind team
  clier get @clier/hello-claude

Step 4. Clone the team

  clier clone @clier/hello-claude

  Downloads the working copy to
  ~/.clier/workspace/@clier.hello-claude/. No cd is needed.

Step 5. Inspect the working copy

  clier status @clier/hello-claude
  clier run list

  You should see a clean working copy and no active runs yet.

Step 6. Start the team

  clier run start @clier/hello-claude

  This launches both members in tmux. Note the run ID.

  On the first start in a fresh working copy, the output includes a
  one-time %q field. Vendor CLIs (e.g., Codex) may show their own
  approval prompts in their pane on first launch. Run
  "clier run attach <run-id>" from your terminal, approve those
  prompts, and detach (Ctrl-b d) before sending messages.

Step 7. Ask hello-claude to greet

  clier run tell --run <run-id> --to @clier/hello-claude \
    "Have both team members greet each other and report the result."

Step 8. Watch and verify

  clier run attach <run-id>        Watch agents in real time
  clier run view <run-id>          Inspect messages and notes

  Confirm both members participated and the greeting completed.

Step 9. Stop the run

  clier run stop <run-id>

Step 10. Tear down the canned team

  clier remove @clier/hello-claude

  Removes the working copy and any associated run plans. Phase 2
  starts from a fresh fork in your own namespace.

================================================================
Phase 2 — Fork it and iterate
================================================================

This is where clier's value loop lives: clone → use → refine →
push → others pull. To make a team yours, fork it on the server,
then clone the fork.

About fork depth:

  fork is shallow — it copies only the team itself into your
  namespace. Leaf resources (instruction, settings, skills) and
  child teams remain owned by the original author. To rewrite a
  leaf, fork that leaf too and rewire your team to point at it.

Step 11. Fork the team into your namespace

  clier fork @clier/hello-claude
  → creates <yourname>/hello-claude

Step 12. Fork the instruction so you own the prompt text

  clier fork @clier/greeting-prompt
  → creates <yourname>/greeting-prompt

Step 13. Rewire your team to use your instruction

  clier edit <yourname>/hello-claude \
    --instruction <yourname>/greeting-prompt@1

  This bumps your team's version because its ref changed.

Step 14. Clone your fork

  clier clone <yourname>/hello-claude

Step 15. Edit the prompt and check status

  Locate CLAUDE.md inside the working copy and add your refinement.

  clier status <yourname>/hello-claude
  → "modified <yourname>/greeting-prompt"

Step 16. Push your refinement

  clier push <yourname>/hello-claude

  Bumps <yourname>/greeting-prompt to v2 and your team's ref
  follows.

Step 17. Verify the version bump

  clier get <yourname>/greeting-prompt
  → latest_version: 2

Step 18. Pull keeps you (and others) in sync

  clier pull <yourname>/hello-claude

  Anyone who has cloned your team can run pull to receive the
  improvement. Iteration is now a true loop — keep refining and
  pushing as you use the team.

Step 19. Cleanup

  clier remove <yourname>/hello-claude

================================================================
Going further
================================================================

Collaborating without forking — clier org

  Members of the same organization share write access to that
  owner's resources, so you can skip fork-rewire and iterate
  together on a shared namespace. See:

    clier org --help          create, invite, list, members

Tip: Use "clier <command> --help" for details on each command.`, hintField),
	}
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return c.Help()
	}
	return cmd
}
