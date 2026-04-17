# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Harness multi-agent teams with a native CLI.**

## Why Clier?

Running multiple AI agents across different roles and repos gets messy fast. Most tools solve this by wrapping already-powerful services like Claude Code and Codex behind their own API and dashboard, then spending most of their effort chasing upstream just to keep parity with what the underlying CLI already does. You end up with a layer that hides what the agent actually does, and still lags whatever vendors ship next.

Clier takes the opposite path: use them as-is. Each agent runs its own native CLI exactly as its vendor ships it, and Clier wraps a transparent harness around it — scoping its role, workspace, skills, and teammates.

## How it works

**1. Per-agent harness** — Each agent is harnessed with its own instruction, workspace, skills, and settings. You control what each agent sees and does.

**2. Deep, multi-agent teams** — No depth limit. Build hierarchies with agents in a single team.

**3. Native, not wrapped** — No API wrappers hiding your tools. Agents run their own CLI directly. You see exactly what the agent sees.

**4. Agent-first** — Every command, output, and hint is designed for agents to parse and act on. You talk to agents in their terminal, not click buttons.

**5. Built on real terminals** — tmux gives each agent its own isolated terminal window. You observe, steer, and intervene in real time.

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
