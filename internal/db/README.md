# internal/db

SQLite persistence layer using sqlc (code generation) and modernc.org/sqlite (CGO-free driver).

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

### Runtime — Query Usage

```go
err := store.Queries.CreateCliProfile(ctx, generated.CreateCliProfileParams{
    ID: "abc-123", Name: "my-profile", Model: "claude-sonnet-4-6", ...
})

profile, err := store.Queries.GetCliProfile(ctx, "abc-123")
```

Consumers define their own interfaces (Go idiom) satisfied by `*generated.Queries` implicitly.

## Regenerating Code

After modifying `schema.sql` or `queries/*.sql`:

```bash
cd internal/db && sqlc generate
```

Commit the regenerated `generated/` directory.
