# Clier

CLI-first project.
Multi-agent CLI orchestrator — spawns and manages multiple AI coding agents (Claude, Codex) via tmux.

## Tech Stack

- Go 1.25, Cobra (CLI framework), SQLite

## Project Structure

gh (GitHub CLI) style. `cmd/` is thin, `internal/` has all logic.

```
main.go               ← entry point (cmd.Execute() only)
cmd/                   ← Cobra commands (parse args → call internal/)
internal/
  domain/              ← entities
  sprint/              ← sprint execution engine
  tmux/                ← tmux session management
  clispawn/            ← agent spawn (Claude, Codex)
  db/                  ← SQLite
  settings/            ← local config/credentials
```

- No port/adapter layer. Interfaces only where genuinely needed.
- Each `internal/` package has one clear responsibility.

## Test Conventions

- Standard `testing` package only. No testify.
- Naming: `TestEntity/Method/StateUnderTest_ExpectedBehavior`
- Nested `t.Run` for grouping by method.
- Test helpers in `helpers_test.go` within the same package.

```go
func TestTeam(t *testing.T) {
    t.Run("AddRelation", func(t *testing.T) {
        t.Run("ValidLeader_AddsRelation", func(t *testing.T) {
            // ...
        })
    })
}
```

## Code Style

- `gofmt` is mandatory. Run before every commit.
- No duplication. Same logic must have a single entry point.
- No workarounds. If code is hard to understand, fix the design.
- Consistency across all files. Follow existing patterns or change all at once.
