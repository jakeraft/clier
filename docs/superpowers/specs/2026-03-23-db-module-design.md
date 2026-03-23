# DB Module Design

## Overview

SQLite persistence layer for domain entities in `internal/db/`.
Migrating from TypeScript/Drizzle ORM (clier-legacy) to Go with sqlc.

## Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| SQLite driver | modernc.org/sqlite | CGO-free, easy cross-compilation for CLI |
| SQL management | sqlc | Type-safe Go codegen from raw SQL (idiomatic Go) |
| Interface location | Consumer-side | Go idiom: accept interfaces, return structs |
| Migration tool | None | Fresh start, no legacy data to migrate |
| Schema init | embed.FS | Single binary, schema.sql embedded and run on startup |
| TeamSnapshot storage | JSON text column | Immutable capture, no need to query individual fields |

## Schema

8 core tables + 4 junction tables. Matches legacy schema with `messages` table added.

### Core Tables

- `cli_profiles` — agent CLI configuration (binary, model, args, dotconfig)
- `system_prompts` — reusable prompt templates
- `environments` — environment variable definitions
- `git_repos` — git repository references
- `members` — agent instances (FK to cli_profile, optional FK to git_repo)
- `teams` — groups of members with a root member
- `sprints` — execution sessions with TeamSnapshot as JSON
- `messages` — inter-agent messages scoped to a sprint

### Junction Tables

- `team_members` — team ↔ member (many-to-many)
- `team_relations` — directed relations between members in a team (leader/peer)
- `member_system_prompts` — member ↔ system_prompt (many-to-many)
- `member_environments` — member ↔ environment (many-to-many)

### Timestamps

All core tables use `INTEGER` for `created_at`/`updated_at` (Unix timestamp).

## File Structure

```
internal/db/
  schema.sql              -- table definitions (sqlc + embed shared)
  queries/
    cli_profile.sql
    system_prompt.sql
    environment.sql
    git_repo.sql
    member.sql            -- includes member_system_prompts, member_environments
    team.sql              -- includes team_members, team_relations
    sprint.sql
    message.sql
  sqlc.yaml
  generated/              -- sqlc auto-generated (committed to git)
  store.go                -- DB init, connection, schema embed
```

## Query Scope

| Entity | Queries |
|--------|---------|
| cli_profile | Create, GetByID, List, Update, Delete |
| system_prompt | Create, GetByID, List, Update, Delete |
| environment | Create, GetByID, List, Update, Delete |
| git_repo | Create, GetByID, List, Update, Delete |
| member | Create, GetByID, List, Update, Delete |
| member junctions | AddSystemPrompt, RemoveSystemPrompt, ListSystemPrompts, AddEnvironment, RemoveEnvironment, ListEnvironments |
| team | Create, GetByID, List, Update, Delete |
| team junctions | AddMember, RemoveMember, ListMembers, AddRelation, RemoveRelation, ListRelations |
| sprint | Create, GetByID, List, UpdateState, Delete |
| message | Create, ListBySprintID, ListBySprintAndMember |

## Consumer Interface Pattern

Consumers define their own interfaces. Example:

```go
// internal/sprint/sprint.go
type MessageStore interface {
    CreateMessage(ctx context.Context, arg ...) error
    ListMessagesBySprint(ctx context.Context, sprintID string) ([]..., error)
}
```

The concrete `db.Store` satisfies these interfaces implicitly via Go's structural typing.
