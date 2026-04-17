# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Harness multi-agent teams with a native CLI.**

## Why Clier?

Running multiple AI agents across different roles and repos gets messy fast. Most tools solve this by wrapping already-powerful services like Claude Code and Codex behind their own API and dashboard, then chasing upstream just to keep parity with what the underlying agents already do. You get a layer that hides what the agent does, and still lags whatever vendors ship next.

Even as developers, we're most productive using these agents interactively in their CLI — not through a wrapped API. Clier extends that to teams: use them as-is. Each agent runs its native CLI as shipped, wrapped in a transparent harness that scopes its role, workspace, skills, and teammates. And since the agent — not a dashboard — does the work, every Clier command and output is shaped for agents to parse and act on, with you watching in the terminal.

## How it works

**1. Native, not wrapped** — No API wrappers between you and the agent. Agents run their own CLI directly, and you see exactly what they see.

**2. Per-agent harness** — Each agent gets its own instruction, workspace, skills, and settings. You control what it sees and does.

**3. Deep, multi-agent teams** — Compose agents into teams, then nest teams inside teams. No depth limit.

**4. Agent-first** — Every command and output is shaped for agents to parse and act on. You talk to agents in their terminal, not click buttons on a dashboard.

**5. Built on real terminals** — tmux gives each agent its own isolated window. You observe, steer, and intervene in real time.

## Quick Start

```bash
brew install jakeraft/tap/clier
```

Open your CLI agent and say:

```
I want to try clier. Explore the CLI and walk me through the tutorial.
```

Under the hood, your agent will:

```bash
clier --help                                # explore available commands
clier tutorial                              # read the tutorial steps
clier auth login                            # authenticate with GitHub
clier clone @clier/hello-claude             # download the tutorial team
cd @clier/hello-claude
clier run start                             # launch agents in tmux
clier run tell --run <run-id> \
  --to @clier/hello-claude \
  "Have both team members greet each other and report the result."
clier run attach <run-id>                   # watch agents in real time
```

## License

[MIT](LICENSE)
