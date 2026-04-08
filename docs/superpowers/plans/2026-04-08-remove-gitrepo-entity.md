# Remove GitRepo Entity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove `GitRepo` as a separate domain entity and inline `GitRepoURL string` directly into `Member`.

**Architecture:** GitRepo is a local-only config (clone URL), not a shared entity. We replace the FK reference pattern (`Member.GitRepoID` → `git_repos` table) with a simple `Member.GitRepoURL` string field. All GitRepo infrastructure (table, Store methods, CLI commands, UI page) is removed.

**Tech Stack:** Go, SQLite, sqlc, React/TypeScript

---

### Task 1: Domain — Remove GitRepo entity, update Member and plan types

**Files:**
- Delete: `internal/domain/resource/gitrepo.go`
- Delete: `internal/domain/resource/gitrepo_test.go`
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/member_test.go`
- Modify: `internal/domain/task.go`

- [ ] **Step 1: Update `Member` struct — replace `GitRepoID` with `GitRepoURL`**

In `internal/domain/member.go`, change the `Member` struct field:

```go
// Before
GitRepoID        string    `json:"git_repo_id"`         // empty string = not set (nullable FK)

// After
GitRepoURL       string    `json:"git_repo_url"`        // empty string = no repo
```

Update `NewMember` signature and body — replace `gitRepoID string` param with `gitRepoURL string`:

```go
func NewMember(name, agentType, model string, args []string,
	agentDotMdID string, skillIDs []string,
	claudeSettingsID, claudeJsonID string,
	envIDs []string, gitRepoURL string) (*Member, error) {
```

In the function body, replace `GitRepoID: gitRepoID` with `GitRepoURL: gitRepoURL`.

Update `Update` method — replace `gitRepoID *string` param with `gitRepoURL *string`:

```go
func (m *Member) Update(name, agentType, model *string, args *[]string,
	agentDotMdID *string, skillIDs *[]string,
	claudeSettingsID, claudeJsonID *string,
	envIDs *[]string, gitRepoURL *string) error {
```

In the body, replace:
```go
if gitRepoID != nil {
    m.GitRepoID = *gitRepoID
}
```
with:
```go
if gitRepoURL != nil {
    m.GitRepoURL = *gitRepoURL
}
```

- [ ] **Step 2: Update `ResolvedMember` — replace `Repo` with `GitRepoURL`**

In `internal/domain/member.go`, update `ResolvedMember`:

```go
// Before
Repo           *resource.GitRepo

// After
GitRepoURL     string
```

Remove the `resource` import if no other field uses it (check: `AgentDotMd`, `Skills`, `ClaudeSettings`, `ClaudeJson`, `Envs` still use `resource` types, so keep the import).

- [ ] **Step 3: Update `task.go` — remove `GitRepoRef`, simplify `WorkspacePlan`**

In `internal/domain/task.go`:

Remove the `GitRepoRef` struct entirely:
```go
// DELETE this entire block
type GitRepoRef struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}
```

Update `WorkspacePlan`:
```go
// Before
type WorkspacePlan struct {
    Memberspace string      `json:"memberspace"`
    Files       []FileEntry `json:"files"`
    GitRepo     *GitRepoRef `json:"git_repo"`
}

// After
type WorkspacePlan struct {
    Memberspace string      `json:"memberspace"`
    Files       []FileEntry `json:"files"`
    GitRepoURL  string      `json:"git_repo_url"`
}
```

- [ ] **Step 4: Delete `resource/gitrepo.go` and `resource/gitrepo_test.go`**

```bash
rm internal/domain/resource/gitrepo.go internal/domain/resource/gitrepo_test.go
```

- [ ] **Step 5: Update `member_test.go`**

In `internal/domain/member_test.go`:

`TestNewMember` — change the last arg from `"repo-1"` to a URL, and assertion from `GitRepoID` to `GitRepoURL`:
```go
m, err := NewMember("coder", "claude", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
    "claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
    []string{"env-1"}, "https://github.com/example/repo.git")
// ...
if m.GitRepoURL != "https://github.com/example/repo.git" {
    t.Errorf("git_repo_url = %q, want %q", m.GitRepoURL, "https://github.com/example/repo.git")
}
```

`TestMember_Update` — change `newRepo` value and assertion:
```go
newRepo := "https://github.com/example/new-repo.git"
// ...
if m.GitRepoURL != "https://github.com/example/new-repo.git" {
    t.Errorf("git_repo_url = %q", m.GitRepoURL)
}
```

- [ ] **Step 6: Run domain tests**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/...`
Expected: PASS (will fail on compile errors in other packages, but domain package itself should pass)

- [ ] **Step 7: Commit**

```bash
git add internal/domain/
git commit -m "refactor: remove GitRepo entity, inline GitRepoURL into Member"
```

---

### Task 2: DB — Schema, queries, sqlc, store

**Files:**
- Modify: `internal/adapter/db/schema.sql`
- Delete: `internal/adapter/db/queries/git_repo.sql`
- Modify: `internal/adapter/db/queries/member.sql`
- Regenerate: `internal/adapter/db/generated/` (sqlc)
- Modify: `internal/adapter/db/store.go`

- [ ] **Step 1: Update `schema.sql` — drop `git_repos` table, update `members`**

In `internal/adapter/db/schema.sql`:

Delete the entire `git_repos` table block:
```sql
-- DELETE this block
CREATE TABLE IF NOT EXISTS git_repos (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

In the `members` table, replace:
```sql
git_repo_id       TEXT REFERENCES git_repos(id) ON DELETE RESTRICT,
```
with:
```sql
git_repo_url      TEXT NOT NULL DEFAULT '',
```

- [ ] **Step 2: Delete `queries/git_repo.sql`**

```bash
rm internal/adapter/db/queries/git_repo.sql
```

- [ ] **Step 3: Update `queries/member.sql` — `git_repo_id` → `git_repo_url`**

In `internal/adapter/db/queries/member.sql`:

Replace `git_repo_id` with `git_repo_url` in CreateMember and UpdateMember queries:

```sql
-- name: CreateMember :execresult
INSERT INTO members (id, name, agent_type, model, args, agent_dot_md_id, claude_settings_id, claude_json_id, git_repo_url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMember :execresult
UPDATE members SET name = ?, agent_type = ?, model = ?, args = ?, agent_dot_md_id = ?, claude_settings_id = ?, claude_json_id = ?, git_repo_url = ?, updated_at = ? WHERE id = ?;
```

GetMember, ListMembers, DeleteMember use `SELECT *` / `DELETE` so they auto-pick up the new column name.

- [ ] **Step 4: Regenerate sqlc**

```bash
cd /Users/jake_kakao/jakeraft/clier/internal/adapter/db && sqlc generate
```

This will:
- Remove `generated/git_repo.sql.go` entirely
- Update `generated/models.go`: remove `GitRepo` struct, change `Member.GitRepoID sql.NullString` → `Member.GitRepoUrl string`
- Update `generated/member.sql.go`: update param structs to use `GitRepoUrl string` instead of `GitRepoID sql.NullString`

- [ ] **Step 5: Update `store.go` — remove GitRepo methods, update Member CRUD**

In `internal/adapter/db/store.go`:

**Delete** the entire GitRepo section (lines 693-760, the 5 methods: `CreateGitRepo`, `GetGitRepo`, `ListGitRepos`, `UpdateGitRepo`, `DeleteGitRepo`).

**Update `CreateMember`** (around line 276): change the `GitRepoID` param:
```go
// Before
GitRepoID:    sql.NullString{String: m.GitRepoID, Valid: m.GitRepoID != ""},

// After
GitRepoUrl:   m.GitRepoURL,
```

**Update `GetMember`** (around line 328): change the field mapping:
```go
// Before
GitRepoID:    row.GitRepoID.String,

// After
GitRepoURL:   row.GitRepoUrl,
```

**Update `UpdateMember`** (around line 370): change the param:
```go
// Before
GitRepoID:    sql.NullString{String: m.GitRepoID, Valid: m.GitRepoID != ""},

// After
GitRepoUrl:   m.GitRepoURL,
```

Remove the `resource` import from `store.go` only if no other method uses it (other methods like `CreateClaudeMd` still use `resource` types, so keep it).

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/db/
git commit -m "refactor: drop git_repos table, inline git_repo_url in members"
```

---

### Task 3: App Layer — plan.go resolve/build, workspace.go

**Files:**
- Modify: `internal/app/task/plan.go`
- Modify: `internal/app/task/service.go`
- Modify: `internal/adapter/workspace/workspace.go`

- [ ] **Step 1: Update `plan.go` `resolveMember` — remove GitRepo store lookup**

In `internal/app/task/plan.go`, replace lines 84-91:
```go
// DELETE this block
var repo *resource.GitRepo
if member.GitRepoID != "" {
    r, err := s.store.GetGitRepo(ctx, member.GitRepoID)
    if err != nil {
        return nil, fmt.Errorf("get git repo for %s: %w", tm.Name, err)
    }
    repo = &r
}
```

And in the return value, replace `Repo: repo` with `GitRepoURL: member.GitRepoURL`:
```go
return &domain.ResolvedMember{
    // ... other fields ...
    GitRepoURL:     member.GitRepoURL,
    Relations:      relations,
}, nil
```

Remove `resource` from imports if no longer used (check: `resource.AgentDotMd`, `resource.Skill`, etc. are still used, so keep it if needed. Actually check — if `resource.GitRepo` was the only `resource` type used for a variable, but `resource.AgentDotMd`, `resource.Skill`, `resource.ClaudeSettings`, `resource.ClaudeJson`, `resource.Env` are also used in this file, so keep import).

- [ ] **Step 2: Update `plan.go` `buildMemberPlan` — use GitRepoURL directly**

In `internal/app/task/plan.go`, replace lines 178-181:
```go
// DELETE this block
var gitRepo *domain.GitRepoRef
if rm.Repo != nil {
    gitRepo = &domain.GitRepoRef{Name: rm.Repo.Name, URL: rm.Repo.URL}
}
```

And update the WorkspacePlan in the return:
```go
Workspace: domain.WorkspacePlan{
    Memberspace: memberspace,
    Files:       files,
    GitRepoURL:  rm.GitRepoURL,
},
```

- [ ] **Step 3: Update `service.go` TaskStore interface — remove `GetGitRepo`**

In `internal/app/task/service.go`, remove from the `TaskStore` interface:
```go
GetGitRepo(ctx context.Context, id string) (resource.GitRepo, error)
```

Check if `resource` import is still needed in service.go (it provides `resource.ClaudeMd`, `resource.Skill`, etc. — keep it).

- [ ] **Step 4: Update `workspace.go` — use `GitRepoURL` string**

In `internal/adapter/workspace/workspace.go`, update `setupGit`:
```go
// Before
func (w *Workspace) setupGit(ctx context.Context, ws domain.WorkspacePlan, workDir string) error {
    if ws.GitRepo == nil {
        return exec.CommandContext(ctx, "git", "init", workDir).Run()
    }
    if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", ws.GitRepo.URL, workDir).Run(); err != nil {
        return fmt.Errorf("git clone %s: %w", ws.GitRepo.URL, err)
    }
    return nil
}

// After
func (w *Workspace) setupGit(ctx context.Context, ws domain.WorkspacePlan, workDir string) error {
    if ws.GitRepoURL == "" {
        return exec.CommandContext(ctx, "git", "init", workDir).Run()
    }
    if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", ws.GitRepoURL, workDir).Run(); err != nil {
        return fmt.Errorf("git clone %s: %w", ws.GitRepoURL, err)
    }
    return nil
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/app/task/ internal/adapter/workspace/
git commit -m "refactor: remove GitRepo resolve, use GitRepoURL directly in plan/workspace"
```

---

### Task 4: CLI — delete repo.go, update member.go, dashboard.go, export.go, import.go

**Files:**
- Delete: `cmd/repo.go`
- Modify: `cmd/member.go`
- Modify: `cmd/dashboard.go`
- Modify: `cmd/export.go`
- Modify: `cmd/import.go`

- [ ] **Step 1: Delete `cmd/repo.go`**

```bash
rm cmd/repo.go
```

- [ ] **Step 2: Update `cmd/member.go` — `--repo` flag changes from ID to URL**

In `cmd/member.go`:

`newMemberCreateCmd`: change help text for the `--repo` flag:
```go
// Before
cmd.Flags().StringVar(&repo, "repo", "", "Git repo ID")

// After
cmd.Flags().StringVar(&repo, "repo", "", "Git repo URL")
```

The call to `domain.NewMember(name, model, cliArgs, claudeMd, skills, settings, claudeJson, repo)` stays the same — the param name `repo` now carries a URL instead of an ID.

`newMemberUpdateCmd`: same flag help change:
```go
// Before
cmd.Flags().StringVar(&repo, "repo", "", "New git repo ID")

// After
cmd.Flags().StringVar(&repo, "repo", "", "New git repo URL")
```

- [ ] **Step 3: Update `cmd/dashboard.go` — remove GitRepos, update memberView**

Remove from `dashboardData` struct:
```go
GitRepos    []gitRepoView    `json:"gitRepos"`
```

Remove the entire `gitRepoView` struct:
```go
// DELETE
type gitRepoView struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    URL       string    `json:"url"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

Remove the `convertGitRepos` function entirely.

In `memberView`, replace `GitRepoID`/`GitRepoName` fields:
```go
// DELETE these two
GitRepoID      *string   `json:"gitRepoId"`
GitRepoName    *string   `json:"gitRepoName"`

// ADD
GitRepoURL     string    `json:"gitRepoUrl"`
```

In `memberPlanView`, replace `GitRepo`:
```go
// Before
GitRepo      *memberPlanGitRepoRef `json:"gitRepo"`

// After
GitRepoURL   string                `json:"gitRepoUrl"`
```

Delete the `memberPlanGitRepoRef` struct:
```go
// DELETE
type memberPlanGitRepoRef struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}
```

In `collectDashboardData`:
- Remove the `repos` list/err block:
```go
// DELETE
repos, err := store.ListGitRepos(ctx)
if err != nil {
    return dashboardData{}, err
}
```
- Remove `repoNames` nameMap:
```go
// DELETE
repoNames := nameMap(repos, func(r resource.GitRepo) (string, string) { return r.ID, r.Name })
```
- Remove `repoNames` from `convertMembers` call and `GitRepos` from return:
```go
// Before
Members:     convertMembers(members, claudeMdNames, skillNames, settingsNames, claudeJsonNames, repoNames, envNames),
// ...
GitRepos:    convertGitRepos(repos),

// After
Members:     convertMembers(members, claudeMdNames, skillNames, settingsNames, claudeJsonNames, envNames),
// (remove GitRepos line entirely)
```

Update `convertMembers` signature — remove `repoNames` param:
```go
func convertMembers(members []domain.Member, claudeMdNames, skillNames, settingsNames, claudeJsonNames, envNames map[string]string) []memberView {
```

In `convertMembers` body, replace the GitRepo block:
```go
// DELETE
if m.GitRepoID != "" {
    mv.GitRepoID = &m.GitRepoID
    name := repoNames[m.GitRepoID]
    mv.GitRepoName = &name
}

// ADD (in the memberView literal initialization)
mv.GitRepoURL = m.GitRepoURL
```

Actually, since `GitRepoURL` is a plain string (not pointer), just set it when building `mv`:
```go
mv := memberView{
    // ... existing fields ...
    GitRepoURL: m.GitRepoURL,
    // ...
}
```

In `convertTasks`, update the plan conversion. Replace:
```go
var gitRepo *memberPlanGitRepoRef
if mp.Workspace.GitRepo != nil {
    gitRepo = &memberPlanGitRepoRef{Name: mp.Workspace.GitRepo.Name, URL: mp.Workspace.GitRepo.URL}
}
```
And in the planView literal:
```go
GitRepo: gitRepo,
```

With:
```go
GitRepoURL: mp.Workspace.GitRepoURL,
```

Remove unused `resource` import if `resource.GitRepo` was the only type used from it (check: `resource.ClaudeMd`, `resource.Skill`, `resource.Settings`, `resource.ClaudeJson`, `resource.Env` are still used — keep import).

- [ ] **Step 4: Update `cmd/export.go` — remove `git_repo` probe**

In `cmd/export.go`, delete the git_repo probe line:
```go
// DELETE
{"git_repo", func() (any, error) { r, e := store.GetGitRepo(ctx, id); return r, e }},
```

- [ ] **Step 5: Update `cmd/import.go` — remove `git_repo` case, update member import**

Delete the entire `case "git_repo":` block (lines 245-263).

In the `case "member":` block, update the `Update` call to match the new signature. Replace:
```go
if err := existing.Update(&m.Name, &m.Model, &m.Args, &m.ClaudeMdID, &m.SkillIDs,
    &m.SettingsID, &m.ClaudeJsonID, &m.EnvIDs, &m.GitRepoID); err != nil {
```
with:
```go
if err := existing.Update(&m.Name, &m.Model, &m.Args, &m.ClaudeMdID, &m.SkillIDs,
    &m.SettingsID, &m.ClaudeJsonID, &m.EnvIDs, &m.GitRepoURL); err != nil {
```

Note: The field names in import.go's member Update call use the domain field names as they exist. Adjust if the current branch uses `AgentDotMdID`/`ClaudeSettingsID` naming. Match the actual `Member.Update()` param names from Task 1.

- [ ] **Step 6: Compile check**

```bash
cd /Users/jake_kakao/jakeraft/clier && go build ./cmd/...
```
Expected: builds cleanly.

- [ ] **Step 7: Commit**

```bash
git add cmd/
git commit -m "refactor: remove repo CLI, update member/dashboard/export/import for GitRepoURL"
```

---

### Task 5: Team Service + Task Service — update Store interfaces and test stubs

**Files:**
- Modify: `internal/app/team/service.go`
- Modify: `internal/app/task/service_test.go`
- Modify: `internal/app/team/service_test.go`

- [ ] **Step 1: Update team `Store` interface — remove GitRepo methods**

In `internal/app/team/service.go`, remove from the `Store` interface:
```go
CreateGitRepo(ctx context.Context, r *resource.GitRepo) error
UpdateGitRepo(ctx context.Context, r *resource.GitRepo) error
```

Check if `resource` import is still needed (other methods use `resource.ClaudeMd`, `resource.Skill`, etc. — keep it).

- [ ] **Step 2: Update `service_test.go` stub — remove `GetGitRepo`**

In `internal/app/task/service_test.go`, delete:
```go
func (s *stubStore) GetGitRepo(_ context.Context, _ string) (resource.GitRepo, error) {
    return resource.GitRepo{}, errors.New("not implemented")
}
```

If `resource` import is no longer needed in this file (check: `resource.ClaudeMd`, `resource.Skill`, etc. are still in stubStore — keep import).

- [ ] **Step 3: Update `team/service_test.go` if it has GitRepo stub methods**

Check `internal/app/team/service_test.go` for `CreateGitRepo`/`UpdateGitRepo` stub methods and remove them.

- [ ] **Step 4: Commit**

```bash
git add internal/app/
git commit -m "refactor: remove GetGitRepo from service interfaces and test stubs"
```

---

### Task 6: Integration tests — update plan_test.go

**Files:**
- Modify: `internal/app/task/plan_test.go`

- [ ] **Step 1: Update `createMinimalTeam` — remove GitRepo creation, use URL directly**

In `internal/app/task/plan_test.go`, remove the GitRepo creation block:
```go
// DELETE
repo, _ := resource.NewGitRepo("test-repo", "https://example.com/repo.git")
if err := store.CreateGitRepo(ctx, repo); err != nil {
    t.Fatalf("CreateGitRepo: %v", err)
}
```

Update the root member creation — replace `repo.ID` with a URL:
```go
// Before
root, _ := domain.NewMember("alice", "claude-sonnet-4-6",
    []string{"--dangerously-skip-permissions"},
    claudeMd.ID, nil, settings.ID, claudeJson.ID, repo.ID)

// After
root, _ := domain.NewMember("alice", "claude-sonnet-4-6",
    []string{"--dangerously-skip-permissions"},
    claudeMd.ID, nil, settings.ID, claudeJson.ID, "https://example.com/repo.git")
```

Note: match the actual `NewMember` signature from Task 1 (it may include `agentType` and `envIDs` params — use the actual current signature).

- [ ] **Step 2: Update `TestResolveTeam` assertions**

Replace:
```go
if root.Repo == nil {
    t.Error("root Repo should not be nil")
}
// ...
if worker.Repo != nil {
    t.Error("worker Repo should be nil")
}
```

With:
```go
if root.GitRepoURL != "https://example.com/repo.git" {
    t.Errorf("root GitRepoURL = %q, want URL", root.GitRepoURL)
}
// ...
if worker.GitRepoURL != "" {
    t.Errorf("worker GitRepoURL = %q, want empty", worker.GitRepoURL)
}
```

- [ ] **Step 3: Update `TestBuildPlans` assertions**

Replace:
```go
if rootPlan.Workspace.GitRepo == nil {
    t.Error("root should have git repo")
}
// ...
if workerPlan.Workspace.GitRepo != nil {
    t.Error("worker should not have git repo")
}
```

With:
```go
if rootPlan.Workspace.GitRepoURL != "https://example.com/repo.git" {
    t.Errorf("root GitRepoURL = %q, want URL", rootPlan.Workspace.GitRepoURL)
}
// ...
if workerPlan.Workspace.GitRepoURL != "" {
    t.Errorf("worker GitRepoURL = %q, want empty", workerPlan.Workspace.GitRepoURL)
}
```

- [ ] **Step 4: Run all Go tests**

```bash
cd /Users/jake_kakao/jakeraft/clier && go test ./...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/task/plan_test.go
git commit -m "test: update plan tests for GitRepoURL"
```

---

### Task 7: UI — types, API, pages, routing

**Files:**
- Modify: `ui/src/types.ts`
- Modify: `ui/src/api.ts`
- Modify: `ui/src/app.tsx`
- Modify: `ui/src/app-layout.tsx`
- Delete: `ui/src/pages/git-repos.tsx`
- Modify: `ui/src/pages/members.tsx`
- Modify: `ui/src/pages/member-detail.tsx`
- Modify: `ui/src/pages/task-detail.tsx`

- [ ] **Step 1: Update `types.ts`**

Delete `GitRepoView` interface entirely.

Remove `gitRepos` from `DashboardData`:
```typescript
// DELETE
gitRepos: GitRepoView[];
```

In `MemberView`, replace:
```typescript
// DELETE
gitRepoId: string | null;
gitRepoName: string | null;

// ADD
gitRepoUrl: string;
```

In `MemberPlanView`, replace:
```typescript
// Before
gitRepo: { name: string; url: string } | null;

// After
gitRepoUrl: string;
```

- [ ] **Step 2: Update `api.ts`**

Remove `GitRepoView` from the import and re-export:
```typescript
// DELETE from import
GitRepoView,

// DELETE from export
GitRepoView,
```

Delete the entire `gitRepos` namespace:
```typescript
// DELETE
gitRepos: {
    list: (): Promise<GitRepoView[]> => Promise.resolve(getData().gitRepos),
    get: (id: string): Promise<GitRepoView> => {
      const item = findById(getData().gitRepos, id);
      return item ? Promise.resolve(item) : Promise.reject(new Error("Not found"));
    },
},
```

- [ ] **Step 3: Delete `pages/git-repos.tsx`**

```bash
rm ui/src/pages/git-repos.tsx
```

- [ ] **Step 4: Update `app.tsx` — remove git-repos route**

Remove the `GitRepos` import:
```typescript
// DELETE
import { GitRepos } from "@/pages/git-repos";
```

Remove the route:
```tsx
// DELETE
<Route path="/git-repos" element={<GitRepos />} />
```

- [ ] **Step 5: Update `app-layout.tsx` — remove Repo nav item**

In the `NAV_ITEMS` array, delete:
```typescript
// DELETE
{ to: "/git-repos", label: "Repo", icon: FolderGit2 },
```

Remove `FolderGit2` from the lucide-react import if no longer used.

- [ ] **Step 6: Update `pages/members.tsx` — show URL instead of EntityBadge**

Replace the "Git Repo" column:
```tsx
// Before
{
    header: "Git Repo",
    cell: (m) =>
      m.gitRepoId ? (
        <EntityBadge to="/git-repos">{m.gitRepoName || EMPTY_DATA}</EntityBadge>
      ) : (
        <EmptyEntityBadge entity="git-repo" />
      ),
    flex: 2,
},

// After
{
    header: "Git Repo",
    cell: (m) => m.gitRepoUrl || EMPTY_DATA,
    flex: 2,
},
```

Remove unused `EntityBadge` and `EmptyEntityBadge` imports if they are no longer used in this file (check — they might not be used elsewhere in this file).

- [ ] **Step 7: Update `pages/member-detail.tsx` — show URL text**

Replace the Git Repo overview row:
```tsx
// Before
{
    label: "Git Repo",
    children: member.gitRepoId ? (
        <EntityBadge to="/git-repos">{member.gitRepoName || EMPTY_DATA}</EntityBadge>
    ) : (
        <EmptyEntityBadge entity="git-repo" />
    ),
},

// After
{
    label: "Git Repo",
    children: member.gitRepoUrl ? (
        <span className={typography[5]}>{member.gitRepoUrl}</span>
    ) : (
        EMPTY_DATA
    ),
},
```

Remove unused imports (`EntityBadge`, `EmptyEntityBadge`) if no longer used in this file. Check: CLAUDE.md and settings rows still use `EntityBadge`, so keep it. `EmptyEntityBadge` may still be used for other fields — check before removing.

- [ ] **Step 8: Update `pages/task-detail.tsx` — simplify gitRepo display**

In `PlanMemberSection`, replace:
```tsx
// Before
{
    label: "GitRepo",
    children: member.gitRepo ? (
        <span className={typography[5]}>{member.gitRepo.url}</span>
    ) : (
        <span className={typography[6]}>-</span>
    ),
},

// After
{
    label: "GitRepo",
    children: member.gitRepoUrl ? (
        <span className={typography[5]}>{member.gitRepoUrl}</span>
    ) : (
        <span className={typography[6]}>-</span>
    ),
},
```

- [ ] **Step 9: Build UI**

```bash
cd /Users/jake_kakao/jakeraft/clier/ui && npm run build
```
Expected: builds cleanly.

- [ ] **Step 10: Commit**

```bash
git add ui/
git commit -m "refactor: remove GitRepos UI page, inline gitRepoUrl in member views"
```

---

### Task 8: Tutorials — update JSON fixtures

**Files:**
- Delete: `tutorials/todo-team/todo-repo.json`
- Modify: `tutorials/todo-team/index.json`
- Modify: `tutorials/todo-team/member-tech-lead.json`
- Modify: `tutorials/todo-team/member-coder.json`
- Modify: `tutorials/todo-team/member-reviewer.json`

- [ ] **Step 1: Delete `todo-repo.json`**

```bash
rm tutorials/todo-team/todo-repo.json
```

- [ ] **Step 2: Update `index.json` — remove `todo-repo.json` reference**

```json
{
  "files": [
    "settings-default.json",
    "claude-json-default.json",
    "claude-md-tech-lead.json",
    "claude-md-coder.json",
    "claude-md-reviewer.json",
    "skill-report-writing.json",
    "member-tech-lead.json",
    "member-coder.json",
    "member-reviewer.json",
    "team.json"
  ]
}
```

- [ ] **Step 3: Update member JSONs — `git_repo_id` → `git_repo_url`**

`member-tech-lead.json`:
```json
{
  "type": "member",
  "data": {
    "id": "c3030303-0001-4000-8000-000000000001",
    "name": "tech-lead",
    "model": "claude-sonnet-4-6",
    "args": ["--dangerously-skip-permissions"],
    "claude_md_id": "a1010101-0001-4000-8000-000000000001",
    "skill_ids": ["f6060606-0001-4000-8000-000000000001"],
    "settings_id": "e5050505-0001-4000-8000-000000000001",
    "claude_json_id": "e5050505-0002-4000-8000-000000000002",
    "env_ids": [],
    "git_repo_url": "https://github.com/jakeraft/clier_todo.git"
  }
}
```

`member-coder.json`:
```json
{
  "type": "member",
  "data": {
    "id": "c3030303-0002-4000-8000-000000000002",
    "name": "coder",
    "model": "claude-sonnet-4-6",
    "args": ["--dangerously-skip-permissions"],
    "claude_md_id": "a1010101-0002-4000-8000-000000000002",
    "skill_ids": [],
    "settings_id": "e5050505-0001-4000-8000-000000000001",
    "claude_json_id": "e5050505-0002-4000-8000-000000000002",
    "env_ids": [],
    "git_repo_url": "https://github.com/jakeraft/clier_todo.git"
  }
}
```

`member-reviewer.json`:
```json
{
  "type": "member",
  "data": {
    "id": "c3030303-0003-4000-8000-000000000003",
    "name": "reviewer",
    "model": "claude-sonnet-4-6",
    "args": ["--dangerously-skip-permissions"],
    "claude_md_id": "a1010101-0003-4000-8000-000000000003",
    "skill_ids": [],
    "settings_id": "e5050505-0001-4000-8000-000000000001",
    "claude_json_id": "e5050505-0002-4000-8000-000000000002",
    "env_ids": [],
    "git_repo_url": "https://github.com/jakeraft/clier_todo.git"
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add tutorials/
git commit -m "refactor: remove todo-repo.json, inline git_repo_url in member fixtures"
```

---

### Task 9: Final verification

- [ ] **Step 1: Full build**

```bash
cd /Users/jake_kakao/jakeraft/clier && go build ./...
```

- [ ] **Step 2: Full test suite**

```bash
cd /Users/jake_kakao/jakeraft/clier && go test ./...
```

- [ ] **Step 3: UI build**

```bash
cd /Users/jake_kakao/jakeraft/clier/ui && npm run build
```

- [ ] **Step 4: Grep for leftover references**

```bash
grep -r "GitRepoID\|git_repo_id\|gitRepoId\|gitRepoName\|GitRepoName\|GitRepoRef\|gitRepoView\|GitRepoView\|convertGitRepos\|ListGitRepos\|CreateGitRepo\|GetGitRepo\|UpdateGitRepo\|DeleteGitRepo\|git_repos\|todo-repo" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.sql" --include="*.json" . | grep -v ".claude/worktrees" | grep -v "node_modules" | grep -v ".worktrees"
```
Expected: no matches (except possibly in generated files that should have been regenerated).
