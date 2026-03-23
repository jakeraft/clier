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

gh (GitHub CLI) style + hexagonal architecture (ports & adapters).

```
main.go               ← entry point (cmd.Execute() only)
cmd/                   ← driving adapter (cobra commands, 의존성 조립)
internal/
  domain/              ← entities + business rules (no external deps)
  app/
    sprint/            ← sprint use case (port definitions + orchestration)
  adapter/
    db/                ← driven adapter (sqlc generated → domain conversion)
    terminal/          ← driven adapter (cmux CLI)
    settings/          ← driven adapter (local config/credentials)
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
