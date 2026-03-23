# internal/adapter/db

SQLite persistence layer using sqlc (code generation) and modernc.org/sqlite (CGO-free driver).

## Architecture

`Store` is a domain-aware facade that wraps sqlc-generated queries and converts between DB rows and domain entities. Consumers never touch `generated/` directly — they depend on `Store` methods that return `domain.*` types.

```
store.go         ← domain-aware facade (generated row <-> domain entity)
schema.sql       ← DDL (embedded, auto-init on first run)
queries/         ← SQL source for sqlc codegen
generated/       ← sqlc output (DO NOT edit manually)
```

## How It Works

### Build Time

```
schema.sql    ─┐
                ├──▶ sqlc generate ──▶ generated/
queries/*.sql  ┘                      ├── db.go        (Queries struct)
                                      ├── models.go    (Go structs per table)
                                      └── *.sql.go     (each query becomes a Go method)
```

SQL query annotations like:

```sql
-- name: GetCliProfile :one
SELECT * FROM cli_profiles WHERE id = ?;
```

become type-safe Go methods:

```go
func (q *Queries) GetCliProfile(ctx context.Context, id string) (CliProfile, error)
```

### Runtime — NewStore

```go
store, err := db.NewStore("~/.clier/data.db")
```

1. `sql.Open("sqlite", path)` — open DB file via modernc.org/sqlite driver
2. `PRAGMA foreign_keys = ON` — SQLite has FK enforcement off by default
3. Execute embedded `schema.sql` — `CREATE TABLE IF NOT EXISTS` creates tables on first run, no-op after
4. `generated.New(db)` — inject DB connection into sqlc Queries struct

### Runtime — Store Usage

```go
// Store methods accept/return domain types, not generated types.
sprint, err := store.GetSprint(ctx, sprintID)    // returns domain.Sprint
err := store.CreateSprint(ctx, &sprint)           // accepts *domain.Sprint
err := store.CreateMessage(ctx, &msg)             // accepts *domain.Message
```

Consumers (app layer) define their own port interfaces satisfied by `*Store`:

```go
// app/sprint/sprint.go
type Store interface {
    GetTeam(ctx context.Context, id string) (domain.Team, error)
    CreateSprint(ctx context.Context, sprint *domain.Sprint) error
    // ...
}
```

## Regenerating Code

After modifying `schema.sql` or `queries/*.sql`:

```bash
cd internal/adapter/db && sqlc generate
```

Commit the regenerated `generated/` directory.
