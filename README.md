# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Harness multi-agent teams with a native CLI.**

## Why Clier?

Running multi-agent teams is tricky. Most tools wrap already-powerful agents behind their own API and dashboard, then chase upstream just to keep parity — you get a layer that hides what the agent does and still lags whatever vendors ship next. And a harness that actually works — the right roles, skills, and team shape — tends to stay buried in one repo while every new team starts from scratch.

I think agents are most productive when used interactively in their own CLI. Clier extends that to teams:

**1. Native, not wrapped** — Agents run their own CLI directly, and you see exactly what they see.

**2. Per-agent harness** — Each agent gets its own instruction, workspace, skills, and settings. You control what it sees and does.

**3. Deep, multi-agent teams** — Compose agents into teams, then nest teams inside teams. No depth limit.

**4. Agent-first** — Every command and output is shaped for agents to parse and act on. The agent drives, not a dashboard; you watch from the terminal.

**5. Real terminals** — tmux gives each agent its own isolated window. You observe, steer, and intervene in real time.

**6. Shareable harnesses** — Publish your agents, skills, and teams; fork someone else's. Everything is versioned, so you build on top instead of starting over.

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
clier clone @clier/hello-claude             # download to ~/.clier/workspace/@clier/hello-claude/
clier run start @clier/hello-claude         # launch agents in tmux
clier run tell --run <run-id> \
  --to @clier/hello-claude \
  "Have both team members greet each other and report the result."
clier run attach <run-id>                   # watch agents in real time
```

## License

[MIT](LICENSE)
