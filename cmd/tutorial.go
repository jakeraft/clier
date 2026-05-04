package cmd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(newTutorialCmd())
}

func newTutorialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tutorial",
		Short: "Walk through your first multi-agent run",
		Long: `Walk through clier end-to-end with the canned hello-clier ↔
hello-codex team pair (jakeraft/hello-clier).

The team you start with:

  jakeraft/hello-clier (root, Claude)
  └── jakeraft/hello-codex (child, Codex)

Both repos are public on GitHub. The server sends the CLI a manifest
describing each agent (what to clone, what protocol markdown to drop,
how to launch in tmux), and the CLI executes that manifest verbatim.

Browsing the catalog (team list / team get) and starting a run
(run start) work without logging in — public repos are reachable
anonymously. Login is only required to author teams (team create /
update / delete) and to star (team star / unstar).

================================================================
Phase 1 — Try the canned team
================================================================

Step 1. (Optional) Log in

  clier auth login

  Authenticates with GitHub via device flow. Skip this for now if you
  only want to browse and run public teams.

Step 2. Open the dashboard for context

  clier open dashboard

  Opens the configured dashboard URL (default http://localhost:5173,
  override via CLIER_DASHBOARD_URL). The web UI is the canonical
  surface for browsing the team catalog and previewing each team's
  run manifest.

Step 3. Browse the catalog from the CLI

  clier team list --sort stars_desc
  clier team get jakeraft/hello-clier

  list defaults to sort=stars_desc (Popular). Add --q <substring>
  for substring search, --namespace <ns> to scope, --page-token
  <cursor> to paginate.

Step 4. Start the team

  clier run start jakeraft/hello-clier

  The server returns a fresh run_id; the CLI clones both repos,
  writes the protocol file for hello-clier (claude reads it via
  --append-system-prompt-file), and launches a tmux window per
  agent. Note the run_id printed on stdout.

  codex's first launch shows a "Do you trust this directory?"
  dialog; the runner auto-presses "1" so you do not have to attach
  for it.

Step 5. Ask the root agent to greet the peer

  clier run tell --run <run-id> --to jakeraft.hello-clier <<'EOF'
  Greet hello-codex and report back what you learned.
  EOF

  The agent IDs are workspace-flat slugs (namespace.team) — the
  protocol markdown the server emitted at run start already embeds
  the fully-qualified clier run tell line for every peer, so the
  agent can copy/paste it verbatim.

Step 6. Watch live, then detach

  clier run attach <run-id>

  Watch both agents in real time. Detach with Ctrl-b d to leave them
  running.

  --agent jakeraft.hello-codex selects a specific window first.

Step 7. Inspect the local plan

  clier run list
  clier run view <run-id>

  All run state lives in ~/.clier/runs/<run-id>/run.json. list and
  view never call the server.

Step 8. Stop the run

  clier run stop <run-id>

  Sends each agent's exit command, kills the tmux session, purges
  the agent clones plus protocols/, and preserves run.json so
  clier run view keeps working post-stop.

================================================================
Phase 2 — Make it your own
================================================================

Once Phase 1 works, register your own team that points at your own
repo. clier never edits the repo — your repo is the source of truth
for content. The server holds composition only (team metadata plus
the protocol markdown template).

Step 9. Create a team in your namespace

  clier team create <yourns>/my-agent \
    --agent-type claude \
    --command 'claude --setting-sources project --strict-mcp-config --dangerously-skip-permissions' \
    --git-repo-url https://github.com/<yourns>/my-agent \
    --description 'My first multi-agent team'

  Pass --subteam <ns/name> repeatedly to attach existing teams as
  subteams (the server walks the graph at run start).

Step 10. Tweak the team

  clier team update <yourns>/my-agent --description 'Updated copy'

  Only the flags you pass are sent (JSON Merge Patch). Use
  --patch-json '{"subteams":[{"namespace":"x","name":"y"}]}' for
  complex bodies. Immutable fields (namespace / name / agent_type)
  cannot be patched — use delete + create.

Step 11. Star teams you want to find later

  clier team star jakeraft/hello-clier
  clier team list --sort stars_desc

  Stars are caller-aware — they show up only when you are logged in.

Step 12. Run your team

  clier run start <yourns>/my-agent

  Same flow as Step 4. Iterate by editing your repo, committing /
  pushing, then re-running — clier-server clones at HEAD of the
  default branch (main) on every mint, so each run sees your
  latest commit.

================================================================
Going further
================================================================

Use 'clier <command> --help' for details on each command.`,
	}
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return c.Help()
	}
	return cmd
}
