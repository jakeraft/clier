# Clier

[![CI](https://github.com/jakeraft/clier/actions/workflows/ci.yml/badge.svg)](https://github.com/jakeraft/clier/actions/workflows/ci.yml)

**Harness multi-agent teams with a native CLI.**

## Why Clier?

As projects grow, you end up running multiple AI agents — different roles, different repos, different contexts. You need a way to set them up, coordinate them, and see what they're doing.

Most tools solve this by wrapping native agents behind another API layer, then building a dashboard on top. It looks impressive, but it hides what the agent is actually doing — and every upstream update breaks something.

We tried the wrapper approach. Performance was unacceptable. Debugging was opaque. We came back to using the CLIs directly — and realized the tool should be designed for the agent, not the human watching it.

Let agents run their native CLIs directly. Give each one its own harness — scoped to its role, transparent to the developer. Compose them into teams. Run them in real terminals. Let the agent do the work, and let the developer see everything.

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
