# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Harness multi-agent teams with a native CLI.**

Clier is a harness for AI coding agent teams. Define agents, compose them into teams, and run them locally in tmux — each agent gets its own terminal, git repo, and system prompt. You watch, steer, and intervene in real time.

## Why Clier?

Running multi-agent teams is tricky. Many tools have tried to solve this, but most of them just wrap already-powerful agents behind their own API and dashboard, then chase upstream to keep parity — leaving you with a layer that hides what the agent does and still lags whatever vendors ship next. And even when a harness actually works — the right roles, skills, and team shape — it tends to stay as one team's private know-how while every new team starts from scratch.

Clier takes a different approach:

**1. Native, not wrapped** — Agents run their own CLI directly. You see exactly what they see.

**2. Agent-first** — Commands and outputs are shaped for agents to parse. The agent drives, not a dashboard.

**3. Real terminals** — tmux gives each agent its own window. Observe, steer, and intervene live.

**4. Per-agent harness** — Each agent has its own instruction, workspace, skills, and settings. You control what it sees and does.

**5. Deep, multi-agent teams** — Compose agents into teams, then nest teams inside teams. No depth limit.

**6. Shareable harnesses** — Publish your agents, skills, and teams; fork someone else's. Everything is versioned, so you build on top, not from scratch.

## Quick Start

```bash
brew install jakeraft/tap/clier
```

### Just explore Clier resources

```bash
clier open dashboard
```

Opens the dashboard in your browser. Browse teams, skills, and docs in read-only mode.

<table>
  <tr>
    <td><img src="assets/docs.png" alt="Docs" /></td>
    <td><img src="assets/home.png" alt="Popular teams" /></td>
  </tr>
  <tr>
    <td><img src="assets/resources.png" alt="Resource list" /></td>
    <td><img src="assets/team.png" alt="Team detail" /></td>
  </tr>
</table>

### Full control using the CLI

Open your CLI agent and say:

```
I want to try clier. Explore the clier CLI using "clier --help" and walk me through the tutorial.
```

Under the hood, the agent drives. Steps marked **(you)** need your hands on the keyboard.

The agent starts by exploring:

```bash
clier --help
clier tutorial
clier auth status
```

It sees you're not logged in and says *"Please run `clier auth login`."*

```bash
clier auth login                            # (you) GitHub device flow
```

Back in the agent:

```bash
clier clone @clier/hello-claude
clier run start @clier/hello-claude
```

On first launch, vendor CLIs may need one-time approval. The agent says *"Attach and approve any prompts, then detach with Ctrl-b d."*

```bash
clier run attach <run-id>                   # (you) approve, then Ctrl-b d
```

The agent then sends the first instruction:

```bash
clier run tell --run <run-id> \
  --to @clier/hello-claude \
  "Have both team members greet each other and report the result."
```

Attach anytime to watch the exchange:

```bash
clier run attach <run-id>                   # (you) watch live
```

## License

[MIT](LICENSE)
