# Clier

**Nested CLI agent teams, designed by you, under your control.**

## Features

|                  | CLI Agent alone              | With Clier                                     |
|------------------|------------------------------|-------------------------------------------------|
| Team             | Flat or 1-level deep         | Nested hierarchy with leader/peer edges         |
| Roles            | Full access, no boundaries   | Scoped — system prompts, git repos per member |
| Control          | One terminal                 | Each agent in its own terminal — watch, intervene, iterate |
| Agents           | Single agent                 | Mix Claude, Codex, Gemini in one team           |
| Communication    | Manual copy & paste          | Built-in messaging between teammates            |

## Why Clier?

**1. Nested teams** — Every existing CLI agent orchestrator is capped at depth 1. Clier has no depth limit.

**2. You design the roles** — Without scoping, agents see your entire local environment — too much context makes them unfocused, and unrestricted access is risky. Clier lets you define each member's system prompt, git repo, and CLI profile. Agents see only what they need.

**3. Under your control** — Agents run locally in your terminal. You see what they see, intervene when needed, and redirect in real time. No blind API calls.

## Quick Start

### Install

```bash
go install github.com/jakeraft/clier@latest
```

### Try the tutorial

```bash
# Seed a sample 7-agent writing team
clier tutorial run story-team

# Start a sprint
clier sprint start --team <team-id>

# Send a task to the chief
clier message send "Write a short story about a mysterious animal." \
  --sprint <sprint-id> --to <chief-member-id>
```

This creates a 3-level nested team:

```
chief
  +-->  se-1
  |       +-->  writer-1
  |       +-->  writer-2
  +-->  se-2
          +-->  writer-3
          +-->  writer-4
```

The chief plans the story, section editors split chapters into scenes, writers produce the text — all coordinated through `clier message send`.

### Build a team from scratch

```bash
# 1. Define roles
clier prompt create --name "lead" --prompt "You review PRs and coordinate fixes."
clier prompt create --name "dev" --prompt "You implement fixes assigned by your leader."
clier profile create --name "claude-opus" --preset-key "claude-opus"
clier repo create --name "my-repo" --url "https://github.com/you/your-repo.git"

# 2. Create scoped members
clier member create --name "lead-1" \
  --profile <profile-id> --prompts <lead-prompt-id> --repo <repo-id>
clier member create --name "dev-1" \
  --profile <profile-id> --prompts <dev-prompt-id> --repo <repo-id>
clier member create --name "dev-2" \
  --profile <profile-id> --prompts <dev-prompt-id> --repo <repo-id>

# 3. Build the team hierarchy
clier team create --name "review-team" --root <lead-1-id>
clier team member add --team <team-id> --member <dev-1-id>
clier team member add --team <team-id> --member <dev-2-id>
clier team relation add --team <team-id> --from <lead-1-id> --to <dev-1-id> --type leader
clier team relation add --team <team-id> --from <lead-1-id> --to <dev-2-id> --type leader

# 4. Run
clier sprint start --team <team-id>
```

## Requirements

- Go 1.25+
- [cmux](https://github.com/jakeraft/cmux) terminal multiplexer
- At least one supported CLI agent installed

## License

[Apache License 2.0](LICENSE)
