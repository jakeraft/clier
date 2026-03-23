# Clier

CLI-first project.
Orchestrate AI coding agent teams in isolated workspaces.
Compose teams of agents (Claude, Codex), define roles and relations, and run sprints in isolated environments.

## Tech Stack

- Go 1.25, Cobra (CLI framework)

### DB

- SQLite driver: modernc.org/sqlite — CGO-free, easy cross-compilation, ideal for CLI distribution
- SQL codegen: sqlc — write SQL directly, generate type-safe Go code (idiomatic Go)

## Project Structure

gh (GitHub CLI) style. `cmd/` is thin, `internal/` has all logic.

```
main.go               ← entry point (cmd.Execute() only)
cmd/                   ← Cobra commands (parse args → call internal/)
internal/
  domain/              ← entities
  sprint/              ← sprint execution engine
  process/             ← agent process management (terminal multiplexer abstraction)
  clispawn/            ← agent spawn (Claude, Codex)
  db/                  ← SQLite
  settings/            ← local config/credentials
```

## Test Conventions

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
