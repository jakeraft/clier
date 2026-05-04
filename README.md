# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Run multi-agent teams in tmux. Thin client of clier-server.**

Clier is a tmux harness for AI coding agent teams. The server mints a
`RunManifest` per call (one `AgentSpec` per BFS-visited team — what to
clone, optional rendered protocol markdown, vendor command + composed
args). The CLI clones, drops the protocol files, and runs `tmux
send-keys` once per agent. Each agent owns its own window — observe,
steer, and intervene live.

## Why Clier?

Most multi-agent harnesses wrap vendor CLIs behind their own dashboards
and chase upstream forever. Clier flips it:

- **Native, not wrapped** — Agents run their own CLI directly (Claude
  Code, Codex, …). You see exactly what they see.
- **Server-owned protocol** — Peer list, communication primitives, and
  operating rules are composed server-side and arrive as agent-typed
  args (`--append-system-prompt-file` for claude, `-c
  developer_instructions='''…'''` for codex). The CLI is vendor-blind.
- **Real terminals** — tmux gives each agent its own window. `attach` to
  watch live; `tell` to message; `stop` to tear down.
- **Per-run ephemeral** — Each `clier run start` clones into its own
  scratch dir under `~/.clier/runs/<run_id>/`. No shared state, no leaks
  across runs.
- **Browse and author in the dashboard** — Team browsing, sorting, and
  search live in the web UI. Author from CLI or dashboard — both hit
  the same endpoints.

## Install

```bash
brew install jakeraft/tap/clier
```

## Commands

```text
clier auth login                            Log in via GitHub device flow
clier auth status                           Show login status
clier auth logout                           Revoke the current session

clier team list                             Browse the catalog (sort/q/cursor)
clier team get <namespace/name>             Show one team
clier team create <namespace/name>          Register a new team
clier team update <namespace/name>          Patch a team (RFC 7396 merge patch)
clier team delete <namespace/name>          Delete a team
clier team star <namespace/name>            Star (idempotent)
clier team unstar <namespace/name>          Unstar (idempotent)

clier run start <namespace/name>            Mint a run, clone, launch in tmux
clier run attach <run-id>                   Watch and intervene live
clier run tell --run <id> --to <agent>      Message an agent (arg or stdin)
clier run stop <run-id>                     Kill the session, free clones
clier run list                              List runs (newest first)
clier run view <run-id>                     Show full run state

clier open dashboard                        Open the web UI
clier tutorial                              Walk through your first run
```

Every command emits JSON on stdout and a single line on stderr for errors
(non-zero exit). Agents and shell pipelines parse the same surface.

## Quick start

```bash
clier auth login
clier run start jakeraft/hello-clier
# → run_id 20260430-101530-abc12345
clier run attach 20260430-101530-abc12345    # detach with Ctrl-b d
clier run tell --run 20260430-101530-abc12345 \
  --to jakeraft.hello-clier "greet the peer and report what you learned"
clier run stop 20260430-101530-abc12345
```

Operators type the URL form `<namespace>/<team>` to `clier run start`
and `clier team *`. Inside the runtime layer (tmux window names, agent
IDs in `--to`) the workspace-flat slug `<namespace>.<team>` appears.

## How it works

For each `clier run start`, the CLI:

1. `POST /api/v1/teams/{ns}/{name}/runs` (public) → `RunManifest{run_id,
   agents[]}`. Each `agents[i]` carries `prepare.git` (always),
   optional `prepare.protocol` (file-based vendors only), and `run`
   (vendor command + server-composed args).
2. `mkdir -p ~/.clier/runs/<run_id>/` and, for every agent that
   carries `prepare.protocol`, write `prepare.protocol.content` to
   `<run_id>/<prepare.protocol.dest>` verbatim.
3. `git clone --depth 1 <prepare.git.repo_url>` into
   `<run_id>/<prepare.git.dest>` per agent (one clone per `agents[]`
   entry — no diamond dedup).
4. For each agent: `tmux new-session/new-window -c <abs_cwd>`
   (cwd = `<run_id>/<prepare.git.dest>` + optional subpath), then a
   single `tmux send-keys -l "<run.command> <shell-escape(run.args[]…)>"`
   followed by `Enter`.
5. For codex windows, send `1` + `Enter` to dismiss the
   "Do you trust this directory?" prompt (auto-trust).
6. Polls each agent's pane title for the vendor's ready marker (claude
   shows "Claude"; codex skips), 60s timeout.
7. Persists the plan at `~/.clier/runs/<run_id>/run.json`.

`clier run tell` checks the tmux session is alive, then send-keys the
message into the target agent's window. `clier run stop` sends each
agent's exit command, kills the session, marks the plan stopped, and
purges the agent clones + `protocols/` while keeping `run.json` for
retrospect (ADR-0004 §4).

## Configuration

Defaults work against a local `make dev` server. Override with env:

```bash
CLIER_SERVER_URL=https://clier.example.com
CLIER_DASHBOARD_URL=https://clier.example.com
```

Credentials live at `~/.clier/credentials.json` (mode `0600`).

## License

[MIT](LICENSE)
