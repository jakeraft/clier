# Remove Env Entity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Completely remove the Env entity from Clier — domain, DB, CLI, task execution, and UI.

**Architecture:** Pure deletion across all layers. Bottom-up approach: DB schema/queries first, then domain, then app logic, then CLI/UI. sqlc regeneration bridges DB→Go. Each task produces a compilable (or at least independently reviewable) unit.

**Tech Stack:** Go (domain/CLI/DB), sqlc (code generation), React/TypeScript (UI), SQLite (schema)

---

### Task 1: Delete env-only files

**Files:**
- Delete: `internal/domain/resource/env.go`
- Delete: `internal/domain/resource/env_test.go`
- Delete: `internal/adapter/db/queries/env.sql`
- Delete: `internal/adapter/db/generated/env.sql.go`
- Delete: `cmd/env.go`
- Delete: `ui/src/pages/envs.tsx`

- [ ] **Step 1: Delete the six env-only files**

```bash
rm internal/domain/resource/env.go
rm internal/domain/resource/env_test.go
rm internal/adapter/db/queries/env.sql
rm internal/adapter/db/generated/env.sql.go
rm cmd/env.go
rm ui/src/pages/envs.tsx
```

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "refactor: delete env-only files (domain, db, cmd, ui)"
```

---

### Task 2: Remove env from DB schema and member queries

**Files:**
- Modify: `internal/adapter/db/schema.sql` — drop `envs` table and `member_envs` table
- Modify: `internal/adapter/db/queries/member.sql` — remove AddMemberEnv, RemoveMemberEnv, ListMemberEnvIDs, DeleteMemberEnvs

- [ ] **Step 1: Edit schema.sql — remove envs table (lines 41-48)**

Remove this block:
```sql
CREATE TABLE IF NOT EXISTS envs (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    key        TEXT NOT NULL,
    value      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

- [ ] **Step 2: Edit schema.sql — remove member_envs table (lines 69-73)**

Remove this block:
```sql
CREATE TABLE IF NOT EXISTS member_envs (
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    env_id    TEXT NOT NULL REFERENCES envs(id)    ON DELETE RESTRICT,
    PRIMARY KEY (member_id, env_id)
);
```

- [ ] **Step 3: Edit member.sql — remove env junction queries (lines 29-39)**

Remove these four queries:
```sql
-- name: AddMemberEnv :execresult
INSERT INTO member_envs (member_id, env_id) VALUES (?, ?);

-- name: RemoveMemberEnv :execresult
DELETE FROM member_envs WHERE member_id = ? AND env_id = ?;

-- name: ListMemberEnvIDs :many
SELECT env_id FROM member_envs WHERE member_id = ? ORDER BY rowid;

-- name: DeleteMemberEnvs :execresult
DELETE FROM member_envs WHERE member_id = ?;
```

- [ ] **Step 4: Regenerate sqlc**

```bash
cd internal/adapter/db && sqlc generate
```

This will update `generated/member.sql.go` to remove the four env junction functions.

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/db/
git commit -m "refactor: remove env tables and queries from DB layer"
```

---

### Task 3: Remove env from domain layer (Member, ResolvedMember)

**Files:**
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/member_test.go`

- [ ] **Step 1: Edit member.go — remove EnvIDs from Member struct**

Remove this line from the Member struct:
```go
EnvIDs       []string  `json:"env_ids"`
```

- [ ] **Step 2: Edit member.go — remove envIDs from NewMember signature and body**

Change NewMember signature from:
```go
func NewMember(name, model string, args []string,
	claudeMdID string, skillIDs []string,
	settingsID, claudeJsonID string,
	envIDs []string, gitRepoID string) (*Member, error) {
```
To:
```go
func NewMember(name, model string, args []string,
	claudeMdID string, skillIDs []string,
	settingsID, claudeJsonID string,
	gitRepoID string) (*Member, error) {
```

Remove from body:
```go
if envIDs == nil {
	envIDs = []string{}
}
```

Remove from the return struct:
```go
EnvIDs:       envIDs,
```

- [ ] **Step 3: Edit member.go — remove envIDs from Update signature and body**

Change Update signature from:
```go
func (m *Member) Update(name, model *string, args *[]string,
	claudeMdID *string, skillIDs *[]string,
	settingsID, claudeJsonID *string,
	envIDs *[]string, gitRepoID *string) error {
```
To:
```go
func (m *Member) Update(name, model *string, args *[]string,
	claudeMdID *string, skillIDs *[]string,
	settingsID, claudeJsonID *string,
	gitRepoID *string) error {
```

Remove from body:
```go
if envIDs != nil {
	m.EnvIDs = *envIDs
}
```

- [ ] **Step 4: Edit member.go — remove Envs from ResolvedMember struct**

Remove this line:
```go
Envs         []resource.Env
```

And remove the `resource` import if no longer used (check if other fields still use it — ClaudeMd, Skills, Settings, ClaudeJson, GitRepo all use resource, so the import stays).

- [ ] **Step 5: Edit member_test.go — update all call sites**

TestNewMember: change from:
```go
m, err := NewMember("coder", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
	"claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
	[]string{"env-1"}, "repo-1")
```
To:
```go
m, err := NewMember("coder", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
	"claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
	"repo-1")
```

TestNewMember_EmptyName: change from:
```go
_, err := NewMember("", "model", nil, "", nil, "", "", nil, "")
```
To:
```go
_, err := NewMember("", "model", nil, "", nil, "", "", "")
```

TestNewMember_EmptyModel: change from:
```go
_, err := NewMember("name", "", nil, "", nil, "", "", nil, "")
```
To:
```go
_, err := NewMember("name", "", nil, "", nil, "", "", "")
```

TestMember_NilSlicesDefault: change from:
```go
m, err := NewMember("coder", "claude-sonnet-4-6", nil, "", nil, "", "", nil, "")
```
To:
```go
m, err := NewMember("coder", "claude-sonnet-4-6", nil, "", nil, "", "", "")
```

Remove EnvIDs nil check:
```go
if m.EnvIDs == nil {
	t.Error("EnvIDs should be empty slice, not nil")
}
```

TestMember_Update: change from:
```go
m, _ := NewMember("old", "old-model", nil, "", nil, "", "", nil, "")
```
To:
```go
m, _ := NewMember("old", "old-model", nil, "", nil, "", "", "")
```

Remove `newEnvs` variable and update Update call from:
```go
newEnvs := []string{"e-1"}
newRepo := "r-1"
if err := m.Update(&newName, &newModel, &newArgs, &newMdID, &newSkills, &newSettings, &newCJ, &newEnvs, &newRepo); err != nil {
```
To:
```go
newRepo := "r-1"
if err := m.Update(&newName, &newModel, &newArgs, &newMdID, &newSkills, &newSettings, &newCJ, &newRepo); err != nil {
```

- [ ] **Step 6: Run domain tests**

```bash
go test ./internal/domain/...
```
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/domain/
git commit -m "refactor: remove env from Member and ResolvedMember"
```

---

### Task 4: Remove env from DB store layer

**Files:**
- Modify: `internal/adapter/db/store.go`

- [ ] **Step 1: Remove Env CRUD methods**

Delete these functions entirely:
- `func (s *Store) CreateEnv(...)`
- `func (s *Store) GetEnv(...)`
- `func (s *Store) ListEnvs(...)`
- `func (s *Store) UpdateEnv(...)`
- `func (s *Store) DeleteEnv(...)`

And the `// Env` section comment.

- [ ] **Step 2: Remove env junction logic from CreateMember**

In `CreateMember`, remove:
```go
for _, envID := range m.EnvIDs {
	if _, err := qtx.AddMemberEnv(ctx, generated.AddMemberEnvParams{
		MemberID: m.ID, EnvID: envID,
	}); err != nil {
		return err
	}
}
```

- [ ] **Step 3: Remove env from GetMember**

Remove:
```go
envIDs, err := s.queries.ListMemberEnvIDs(ctx, id)
```
and the nil check, and the `EnvIDs: envIDs` assignment in the return struct.

- [ ] **Step 4: Remove env junction logic from UpdateMember**

Remove:
```go
if _, err := qtx.DeleteMemberEnvs(ctx, m.ID); err != nil {
	return err
}
for _, envID := range m.EnvIDs {
	if _, err := qtx.AddMemberEnv(ctx, generated.AddMemberEnvParams{
		MemberID: m.ID, EnvID: envID,
	}); err != nil {
		return err
	}
}
```

- [ ] **Step 5: Update DeleteMember comment**

Change:
```go
// DeleteMember deletes a member. CASCADE: member_skills, member_envs.
```
To:
```go
// DeleteMember deletes a member. CASCADE: member_skills.
```

- [ ] **Step 6: Remove unused imports if any**

Check if `resource` package import is still needed in store.go after removing Env methods.

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/db/store.go
git commit -m "refactor: remove env from DB store layer"
```

---

### Task 5: Remove env from task execution layer

**Files:**
- Modify: `internal/app/task/command.go`
- Modify: `internal/app/task/command_test.go`
- Modify: `internal/app/task/plan.go`
- Modify: `internal/app/task/service.go`
- Modify: `internal/app/task/service_test.go`

- [ ] **Step 1: Edit command.go — remove userDefinedEnvs function**

Delete entirely:
```go
// userDefinedEnvs converts user-created Env resources to KEY=VALUE strings.
func userDefinedEnvs(envs []resource.Env) []string {
	out := make([]string, len(envs))
	for i, e := range envs {
		out[i] = e.Key + "=" + e.Value
	}
	return out
}
```

- [ ] **Step 2: Edit command.go — remove userEnvs param from buildEnv**

Change from:
```go
func buildEnv(teamName, memberName, taskID, memberID string, userEnvs []resource.Env) []string {
	var env []string
	env = append(env, systemEnvs(taskID, memberID)...)
	env = append(env, authEnvs()...)
	env = append(env, identityEnvs(teamName, memberName)...)
	env = append(env, userDefinedEnvs(userEnvs)...)
	return env
}
```
To:
```go
func buildEnv(teamName, memberName, taskID, memberID string) []string {
	var env []string
	env = append(env, systemEnvs(taskID, memberID)...)
	env = append(env, authEnvs()...)
	env = append(env, identityEnvs(teamName, memberName)...)
	return env
}
```

- [ ] **Step 3: Edit command.go — remove userEnvs param from buildCommand**

Change from:
```go
func buildCommand(model string, args []string, workDir, teamName, memberName, taskID, memberID string,
	userEnvs []resource.Env) string {
	cmd := buildAgentCommand(model, args, workDir)
	env := buildEnv(teamName, memberName, taskID, memberID, userEnvs)
	return buildEnvCommand(cmd, env)
}
```
To:
```go
func buildCommand(model string, args []string, workDir, teamName, memberName, taskID, memberID string) string {
	cmd := buildAgentCommand(model, args, workDir)
	env := buildEnv(teamName, memberName, taskID, memberID)
	return buildEnvCommand(cmd, env)
}
```

Remove the `resource` import from command.go (no longer needed).

- [ ] **Step 4: Edit plan.go — remove env resolve loop from resolveMember**

Remove lines 75-82:
```go
envs := make([]resource.Env, 0, len(member.EnvIDs))
for _, id := range member.EnvIDs {
	env, err := s.store.GetEnv(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get env %s: %w", id, err)
	}
	envs = append(envs, env)
}
```

Remove `Envs: envs,` from the return struct.

- [ ] **Step 5: Edit plan.go — remove userEnvs from buildMemberPlan**

Remove:
```go
userEnvs := rm.Envs
```

Change buildCommand call from:
```go
cmd := buildCommand(model, args, PlaceholderMemberspace+"/project", teamName, rm.Name, taskID, rm.TeamMemberID, userEnvs)
```
To:
```go
cmd := buildCommand(model, args, PlaceholderMemberspace+"/project", teamName, rm.Name, taskID, rm.TeamMemberID)
```

- [ ] **Step 6: Edit service.go — remove GetEnv from TaskStore interface**

Remove:
```go
GetEnv(ctx context.Context, id string) (resource.Env, error)
```

Remove `resource` from imports if no longer used (check — other methods still reference resource types, so it likely stays).

- [ ] **Step 7: Edit service_test.go — remove GetEnv from stubStore**

Remove:
```go
func (s *stubStore) GetEnv(_ context.Context, _ string) (resource.Env, error) {
	return resource.Env{}, errors.New("not implemented")
}
```

Remove `resource` import if no longer used.

- [ ] **Step 8: Edit command_test.go — update tests**

In TestBuildEnv/IncludesAllCategories, change:
```go
userEnvs := []resource.Env{
	{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
}
env := buildEnv("my-team", "reviewer", "task-1", "m1", userEnvs)
```
To:
```go
env := buildEnv("my-team", "reviewer", "task-1", "m1")
```

Remove the GITHUB_TOKEN assertion from the envMap checks.

In TestBuildEnv/NoUserEnvs, change:
```go
env := buildEnv("my-team", "coder", "task-1", "m2", nil)
// system(3) + auth(1) + identity(4) = 8
if len(env) != 8 {
```
To:
```go
env := buildEnv("my-team", "coder", "task-1", "m2")
// system(3) + auth(1) + identity(4) = 8
if len(env) != 8 {
```

In TestBuildCommand/AllArgs, change:
```go
cmd := buildCommand("claude-sonnet-4-6",
	[]string{"--dangerously-skip-permissions", "--verbose"},
	PlaceholderMemberspace+"/project",
	"my-team", "coder", "task-1", "m1", nil)
```
To:
```go
cmd := buildCommand("claude-sonnet-4-6",
	[]string{"--dangerously-skip-permissions", "--verbose"},
	PlaceholderMemberspace+"/project",
	"my-team", "coder", "task-1", "m1")
```

In TestBuildCommand/WithUserEnvs — delete entire test case (it tested user env injection which no longer exists).

Remove `resource` import from command_test.go.

- [ ] **Step 9: Run task tests**

```bash
go test ./internal/app/task/...
```
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/app/task/
git commit -m "refactor: remove env from task execution layer"
```

---

### Task 6: Remove env from team service interface

**Files:**
- Modify: `internal/app/team/service.go`

- [ ] **Step 1: Remove CreateEnv and UpdateEnv from Store interface**

Remove:
```go
CreateEnv(ctx context.Context, e *resource.Env) error
UpdateEnv(ctx context.Context, e *resource.Env) error
```

Check if `resource` import is still needed (yes — other Create/Update methods still use resource types).

- [ ] **Step 2: Commit**

```bash
git add internal/app/team/service.go
git commit -m "refactor: remove env from team service interface"
```

---

### Task 7: Remove env from CLI commands

**Files:**
- Modify: `cmd/member.go`
- Modify: `cmd/import.go`
- Modify: `cmd/export.go`
- Modify: `cmd/dashboard.go`

- [ ] **Step 1: Edit member.go — remove --envs flag from create command**

Remove variable declaration:
```go
var name, model, claudeMd, settings, claudeJson, repo string
var cliArgs, skills, envs []string
```
Change to:
```go
var name, model, claudeMd, settings, claudeJson, repo string
var cliArgs, skills []string
```

Change NewMember call from:
```go
m, err := domain.NewMember(name, model, cliArgs, claudeMd, skills, settings, claudeJson, envs, repo)
```
To:
```go
m, err := domain.NewMember(name, model, cliArgs, claudeMd, skills, settings, claudeJson, repo)
```

Remove flag registration:
```go
cmd.Flags().StringSliceVar(&envs, "envs", nil, "Env IDs (comma-separated)")
```

- [ ] **Step 2: Edit member.go — remove --envs flag from update command**

Remove variable declaration change similarly (remove `envs` from `var cliArgs, skills, envs []string`).

Remove the envsPtr block:
```go
var envsPtr *[]string
if cmd.Flags().Changed("envs") {
	envsPtr = &envs
}
```

Change Update call from:
```go
if err := m.Update(namePtr, modelPtr, argsPtr, claudeMdPtr, skillsPtr, settingsPtr, claudeJsonPtr, envsPtr, repoPtr); err != nil {
```
To:
```go
if err := m.Update(namePtr, modelPtr, argsPtr, claudeMdPtr, skillsPtr, settingsPtr, claudeJsonPtr, repoPtr); err != nil {
```

Remove flag registration:
```go
cmd.Flags().StringSliceVar(&envs, "envs", nil, "New env IDs (comma-separated)")
```

- [ ] **Step 3: Edit import.go — remove "env" case**

Delete the entire `case "env":` block (lines 265-283).

In the `case "member":` block, the Update call passes `&m.EnvIDs` — change from:
```go
if err := existing.Update(&m.Name, &m.Model, &m.Args, &m.ClaudeMdID, &m.SkillIDs,
	&m.SettingsID, &m.ClaudeJsonID, &m.EnvIDs, &m.GitRepoID); err != nil {
```
To:
```go
if err := existing.Update(&m.Name, &m.Model, &m.Args, &m.ClaudeMdID, &m.SkillIDs,
	&m.SettingsID, &m.ClaudeJsonID, &m.GitRepoID); err != nil {
```

- [ ] **Step 4: Edit export.go — remove "env" probe**

Remove:
```go
{"env", func() (any, error) { e, err := store.GetEnv(ctx, id); return e, err }},
```

- [ ] **Step 5: Edit dashboard.go — remove all env references**

Remove from collectDashboardData:
```go
envs, err := store.ListEnvs(ctx)
if err != nil {
	return dashboardData{}, err
}
```

Remove:
```go
envNames := nameMap(envs, func(e resource.Env) (string, string) { return e.ID, e.Name })
```

Change convertMembers call — remove `envNames` parameter.

Remove `Envs: convertEnvs(envs),` from the return struct.

Remove the `convertEnvs` function entirely.

Remove the `envView` struct entirely.

Remove `Envs` field from `dashboardData` struct.

Remove `envNames` parameter from `convertMembers` function signature and remove env-related logic inside it:
```go
eNames := make([]string, 0, len(m.EnvIDs))
for _, id := range m.EnvIDs {
	eNames = append(eNames, envNames[id])
}
```

Remove `EnvIDs` and `EnvNames` from the `memberView` struct and the construction inside convertMembers.

- [ ] **Step 6: Verify compilation**

```bash
go build ./...
```
Expected: SUCCESS

- [ ] **Step 7: Run all Go tests**

```bash
go test ./...
```
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/
git commit -m "refactor: remove env from CLI commands (member, import, export, dashboard)"
```

---

### Task 8: Remove env from UI layer

**Files:**
- Modify: `ui/src/app.tsx`
- Modify: `ui/src/app-layout.tsx`
- Modify: `ui/src/lib/entities.ts`
- Modify: `ui/src/api.ts`
- Modify: `ui/src/types.ts`
- Modify: `ui/src/pages/member-detail.tsx`
- Modify: `ui/src/index.css`

- [ ] **Step 1: Edit types.ts — remove EnvView and env fields**

Remove the `EnvView` interface entirely.

Remove from dashboard data type: `envs: EnvView[];`

Remove from MemberView type: `envIds: string[];` and `envNames: string[];`

- [ ] **Step 2: Edit api.ts — remove env API methods and EnvView import/export**

Remove `EnvView` from import/export.

Remove:
```typescript
envs: {
  list: (): Promise<EnvView[]> => Promise.resolve(getData().envs),
  get: (id: string): Promise<EnvView> => {
    const item = findById(getData().envs, id);
    ...
  },
},
```

- [ ] **Step 3: Edit app.tsx — remove Envs route**

Remove import:
```typescript
import { Envs } from "@/pages/envs";
```

Remove route:
```tsx
<Route path="/envs" element={<Envs />} />
```

- [ ] **Step 4: Edit app-layout.tsx — remove Env nav item**

Remove `KeyRound` from lucide-react import.

Remove from NAV_ITEMS:
```typescript
{ to: "/envs", label: "Env", icon: KeyRound },
```

- [ ] **Step 5: Edit entities.ts — remove env from Entity type and maps**

Remove `"env"` from the Entity union type.

Remove from ENTITY_STYLE:
```typescript
["env", "bg-entity-env/10 text-entity-env hover:bg-entity-env/20 [a&]:hover:bg-entity-env/20"],
```

Remove from ENTITY_ICON:
```typescript
["env", KeyRound],
```

Remove `KeyRound` from lucide-react import.

Remove from SEGMENT_TO_ENTITY:
```typescript
["envs", "env"],
```

Remove `envs` from the regex in entityFromPath.

- [ ] **Step 6: Edit member-detail.tsx — remove Env section**

Remove the entire Env row from the OverviewTable rows array:
```tsx
{
  label: "Env",
  children: (
    <EntityBadgeList
      entity="env"
      items={member.envIds.map((id, i) => ({
        id,
        name: member.envNames[i] ?? EMPTY_DATA,
        to: `/envs/${id}`,
      }))}
    />
  ),
},
```

- [ ] **Step 7: Edit index.css — remove env CSS variables**

Remove from :root:
```css
--color-entity-env: var(--entity-env);
```

Remove from light theme:
```css
--entity-env: oklch(0.546 0.16 30);
```

Remove from dark theme:
```css
--entity-env: oklch(0.714 0.14 30);
```

- [ ] **Step 8: Build UI**

```bash
cd ui && pnpm build
```
Expected: SUCCESS

- [ ] **Step 9: Commit**

```bash
git add ui/src/
git commit -m "refactor: remove env from UI (pages, nav, types, styles)"
```

---

### Task 9: Final verification

- [ ] **Step 1: Run full Go test suite**

```bash
go test ./...
```
Expected: ALL PASS

- [ ] **Step 2: Grep for orphan env references**

```bash
grep -rn 'EnvID\|EnvIDs\|envIDs\|env_id\|env_ids\|envNames\|envView\|EnvView\|member_envs\|userDefinedEnvs\|userEnvs\|convertEnvs\|ListEnvs\|CreateEnv\|GetEnv\|UpdateEnv\|DeleteEnv' --include='*.go' --include='*.ts' --include='*.tsx' --include='*.sql' --include='*.css' .
```
Expected: No matches (excluding node_modules, .worktrees)

- [ ] **Step 3: Final commit if any fixups needed**
