# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Run multi-agent teams in tmux. Thin client of clier-server.**

Clier is a tmux harness for AI coding agent teams. The server resolves a
team into a manifest (mounts to clone, agents to launch with the right
flags). The CLI clones, runs `tmux send-keys` once per agent, and gets out
of the way. Each agent has its own window — observe, steer, and intervene
live.

## Why Clier?

Most multi-agent harnesses wrap vendor CLIs behind their own dashboards
and chase upstream forever. Clier flips it:

- **Native, not wrapped** — Agents run their own CLI directly (Claude
  Code, Codex, …). You see exactly what they see.
- **Server-owned protocol** — Peer list, communication primitives, and
  operating rules are composed server-side and arrive as agent-typed
  args (`--append-system-prompt` for claude, `-c developer_instructions=…`
  for codex). The CLI is vendor-blind.
- **Real terminals** — tmux gives each agent its own window. `attach` to
  watch live; `tell` to message; `stop` to tear down.
- **Per-run ephemeral** — Each `clier run start` clones into its own
  scratch dir under `~/.clier/runs/<runID>/`. No shared state, no leaks
  across runs.
- **Browse and author in the dashboard** — Team browsing, forking, and
  editing live in the web UI. The CLI is for running, not authoring.

## Install

```bash
brew install jakeraft/tap/clier
```

## Commands

```text
clier auth login                           Log in via GitHub device flow
clier auth status                          Show login status
clier auth logout                          Revoke the current session

clier run start <namespace/name>           Resolve, clone, and launch in tmux
clier run attach <run-id>                  Watch and intervene live
clier run tell --run <id> --to <agent>     Message an agent (content via arg or stdin)
clier run stop <run-id>                    Kill the tmux session, free clones
clier run list                             List runs (newest first)
clier run view <run-id>                    Show full run state
```

Every command emits JSON on stdout and a single line on stderr for errors
(non-zero exit). Agents and shell pipelines parse the same surface.

## Quick start

```bash
clier auth login
clier run start jakeraft/clier-qa-claude
# → run_id 20260430-101530-abc12345
clier run attach 20260430-101530-abc12345    # (you) detach with Ctrl-b d
clier run tell --run 20260430-101530-abc12345 \
  --to jakeraft.clier-qa-claude "smoke test yourself, write reports/SMOKE.md"
clier run stop 20260430-101530-abc12345
```

Agent IDs use the workspace-flat slug `<namespace>.<team>`. Operators
type the URL form `<namespace>/<team>` to `clier run start`.

## How it works

For each `clier run start`, the CLI:

1. `GET /api/v1/teams/{ns}/{name}/resolve` (public) → `RunManifest{mounts, agents[]}`
2. Mints a runID + tmux session name; cleans up the scratch dir + session
   on any failure during start.
3. `git clone --depth 1` each mount into `~/.clier/runs/<runID>/mounts/<mount>/`.
4. For each agent: `tmux new-session/new-window -c <abs-cwd>`, then a
   single `tmux send-keys -l "<command> <shell-escape(args[]…)>"; Enter`.
5. Polls each agent's pane title for the vendor's ready marker (claude
   shows "Claude" in title; codex skips), 60s timeout.
6. Persists the plan at `~/.clier/runs/<runID>/run.json`.

`clier run tell` checks the tmux session is alive, then send-keys the
message into the target agent's window. `clier run stop` sends the
agent's exit command, kills the session, marks the plan stopped, and
purges `mounts/` while keeping `run.json` for retrospection.

## Configuration

Defaults work against a local `make dev` server. Override with env:

```bash
CLIER_SERVER_URL=https://clier.example.com
CLIER_DASHBOARD_URL=https://clier.example.com
```

Credentials live at `~/.clier/credentials.json` (mode `0600`).

## License

[MIT](LICENSE)
