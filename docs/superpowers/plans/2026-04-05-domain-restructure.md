# Domain Restructure: resolve / build / expand Consistency

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Express the Resource -> Member -> Team -> Session flow in code structure and establish a consistent resolve/build/expand vocabulary across the codebase.

**Architecture:** Extract resource types (Env, GitRepo, SystemPrompt, CliProfile) into `domain/resource/` package. Add `ResolvedMember` and `ResolvedTeam` domain types. Split `buildPlan()` into `resolveTeam()` + `buildPlans()`. Rename `resolvePlaceholders` -> `expandPlaceholders`. Remove ad-hoc snapshot types.

**Tech Stack:** Go 1.25, standard library only (no new deps)

---

### Task 1: Add ResolvedMember and ResolvedTeam to domain

**Files:**
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/team.go`

- [ ] **Step 1: Add ResolvedMember to member.go**

Append after the `Update` method at end of file:

```go
// ResolvedMember is a Member spec with all referenced resources loaded.
// Produced by the resolve phase; consumed by the build phase to create MemberPlan.
type ResolvedMember struct {
	TeamMemberID string
	Name         string
	Profile      CliProfile
	Prompts      []SystemPrompt
	Envs         []Env
	Repo         *GitRepo
	Relations    MemberRelations
}
```

- [ ] **Step 2: Add ResolvedTeam to team.go**

Append after `teamMemberIndex` at end of file:

```go
// ResolvedTeam is a Team with all members fully resolved.
// Produced by the resolve phase; consumed by the build phase.
type ResolvedTeam struct {
	Team
	Members []ResolvedMember
}
```

- [ ] **Step 3: Run tests to confirm nothing breaks**

Run: `go test ./internal/domain/... -v -count=1`
Expected: all PASS (additive changes only)

- [ ] **Step 4: Commit**

```bash
git add internal/domain/member.go internal/domain/team.go
git commit -m "feat: add ResolvedMember and ResolvedTeam domain types

Intermediate types that make the resolve -> build flow explicit.
ResolvedMember holds a Member spec with all resources loaded.
ResolvedTeam holds a Team with all members resolved."
```

---

### Task 2: Implement resolveTeam + buildPlans in app/session

**Files:**
- Modify: `internal/app/session/plan.go` — rewrite with resolve and build functions
- Modify: `internal/app/session/prompt.go:12` — change joinPrompts signature
- Modify: `internal/app/session/command.go:31-44,77-85` — change buildEnv/buildCommand signatures
- Modify: `internal/app/session/service.go:68-109` — update Start() to call resolve + build
- Test: `internal/app/session/plan_test.go` — update test for new entry points
- Test: `internal/app/session/prompt_test.go` — update for new signature
- Test: `internal/app/session/command_test.go` — update for new signature

- [ ] **Step 1: Rewrite plan.go with resolveTeam + buildPlans**

Replace the entire `internal/app/session/plan.go` with:

```go
package session

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderSessionID   = "{{CLIER_SESSION_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
)

// resolveTeam loads all referenced resources for every team member.
// This is the resolve phase: ID strings -> actual domain objects.
func (s *Service) resolveTeam(ctx context.Context, team domain.Team) (*domain.ResolvedTeam, error) {
	members := make([]domain.ResolvedMember, 0, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		rm, err := s.resolveMember(ctx, &team, tm)
		if err != nil {
			return nil, err
		}
		members = append(members, *rm)
	}
	return &domain.ResolvedTeam{Team: team, Members: members}, nil
}

// resolveMember loads the member spec and all its referenced resources.
func (s *Service) resolveMember(ctx context.Context, team *domain.Team, tm domain.TeamMember) (*domain.ResolvedMember, error) {
	member, err := s.store.GetMember(ctx, tm.MemberID)
	if err != nil {
		return nil, fmt.Errorf("get member %s: %w", tm.MemberID, err)
	}

	profile, err := s.store.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return nil, fmt.Errorf("get cli profile for %s: %w", tm.Name, err)
	}

	prompts := make([]domain.SystemPrompt, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := s.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, sp)
	}

	envs := make([]domain.Env, 0, len(member.EnvIDs))
	for _, id := range member.EnvIDs {
		env, err := s.store.GetEnv(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get env %s: %w", id, err)
		}
		envs = append(envs, env)
	}

	var repo *domain.GitRepo
	if member.GitRepoID != "" {
		r, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return nil, fmt.Errorf("get git repo for %s: %w", tm.Name, err)
		}
		repo = &r
	}

	relations := team.MemberRelations(tm.ID)

	return &domain.ResolvedMember{
		TeamMemberID: tm.ID,
		Name:         tm.Name,
		Profile:      profile,
		Prompts:      prompts,
		Envs:         envs,
		Repo:         repo,
		Relations:    relations,
	}, nil
}

// buildPlans constructs MemberPlans from a resolved team.
// This is the build phase: resolved objects -> execution plan with placeholders.
func buildPlans(resolved *domain.ResolvedTeam, sessionID string) ([]domain.MemberPlan, error) {
	nameByID := make(map[string]string, len(resolved.Members))
	for _, rm := range resolved.Members {
		nameByID[rm.TeamMemberID] = rm.Name
	}

	plans := make([]domain.MemberPlan, 0, len(resolved.Members))
	for _, rm := range resolved.Members {
		plan, err := buildMemberPlan(&rm, nameByID, resolved.Team.Name, sessionID)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// buildMemberPlan constructs a single MemberPlan from a resolved member.
func buildMemberPlan(rm *domain.ResolvedMember, nameByID map[string]string, teamName, sessionID string) (domain.MemberPlan, error) {
	memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, PlaceholderSessionID, rm.TeamMemberID)

	clierPrompt := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID)
	userPrompt := joinPrompts(rm.Prompts)
	prompt := "---\n\n" + clierPrompt + "\n---\n\n" + userPrompt

	authEnvs := setAuth()

	files, err := buildClaudeFiles(rm.Profile.DotConfig, PlaceholderMemberspace+"/project", PlaceholderMemberspace)
	if err != nil {
		return domain.MemberPlan{}, fmt.Errorf("build files for %s: %w", rm.Name, err)
	}

	cmd := buildCommand(rm.Profile, prompt, sessionID, rm.TeamMemberID, authEnvs, rm.Envs)

	launchPath := PlaceholderMemberspace + "/launch.sh"
	files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

	var gitRepo *domain.GitRepoRef
	if rm.Repo != nil {
		gitRepo = &domain.GitRepoRef{Name: rm.Repo.Name, URL: rm.Repo.URL}
	}

	return domain.MemberPlan{
		TeamMemberID: rm.TeamMemberID,
		MemberName:   rm.Name,
		Terminal:     domain.TerminalPlan{Command: ". " + launchPath},
		Workspace: domain.WorkspacePlan{
			Memberspace: memberspace,
			Files:       files,
			GitRepo:     gitRepo,
		},
	}, nil
}
```

- [ ] **Step 2: Update joinPrompts in prompt.go**

In `internal/app/session/prompt.go`, change the `joinPrompts` signature from `[]domain.PromptSnapshot` to `[]domain.SystemPrompt`:

```go
// joinPrompts combines multiple system prompts into a single string,
// separated by double newlines.
func joinPrompts(prompts []domain.SystemPrompt) string {
	parts := make([]string, 0, len(prompts))
	for _, sp := range prompts {
		parts = append(parts, sp.Prompt)
	}
	return strings.Join(parts, "\n\n---\n\n")
}
```

- [ ] **Step 3: Update buildEnv and buildCommand in command.go**

In `internal/app/session/command.go`, change `buildEnv` (line 31-44):

```go
// buildEnv assembles the full set of environment variables for a member command.
func buildEnv(sessionID, memberID string,
	authEnvs []string, userEnvs []domain.Env) []string {

	env := []string{
		configDirEnv(),
		"CLIER_SESSION_ID=" + sessionID,
		"CLIER_MEMBER_ID=" + memberID,
	}
	env = append(env, authEnvs...)
	for _, e := range userEnvs {
		env = append(env, e.Key+"="+e.Value)
	}
	return env
}
```

Change `buildCommand` (line 77-85) to accept `domain.CliProfile` directly:

```go
// buildCommand returns the complete shell command for launching an agent,
// including environment variable exports.
func buildCommand(profile domain.CliProfile, prompt, sessionID, memberID string,
	authEnvs []string, userEnvs []domain.Env) string {

	workDir := PlaceholderMemberspace + "/project"
	cmd := buildAgentCommand(profile.Model, profile.SystemArgs, profile.CustomArgs, prompt, workDir)
	env := buildEnv(sessionID, memberID, authEnvs, userEnvs)
	return buildEnvCommand(cmd, env)
}
```

- [ ] **Step 4: Update service.go Start() to use resolve + build**

In `internal/app/session/service.go`, replace the Start method (lines 68-109):

```go
// Start resolves the team, builds the execution plan, expands placeholders,
// prepares the workspace, and launches terminals for each member.
func (s *Service) Start(ctx context.Context, team domain.Team, auth AuthChecker) (*domain.Session, error) {
	// Resolve: ID references -> loaded domain objects
	resolved, err := s.resolveTeam(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("resolve team: %w", err)
	}

	sessionID := uuid.NewString()

	// Build: resolved objects -> execution plan (with placeholders)
	plan, err := buildPlans(resolved, sessionID)
	if err != nil {
		return nil, fmt.Errorf("build plans: %w", err)
	}

	session, err := domain.NewSession(sessionID, team.ID)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	session.Plan = plan

	// Expand: placeholders -> concrete paths
	claudeToken := resolveAuth(auth)
	members := make([]domain.MemberPlan, 0, len(plan))
	for _, m := range plan {
		members = append(members, resolvePlaceholders(m, s.base, s.homeDir, sessionID, claudeToken))
	}

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(sessionID)
		}
	}()

	// Start: prepare workspace + launch terminals
	if err := s.workspace.Prepare(ctx, members); err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	if err := s.terminal.Launch(sessionID, team.Name, members); err != nil {
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	success = true
	return session, nil
}
```

- [ ] **Step 5: Update plan_test.go**

Replace `internal/app/session/plan_test.go` — the test now calls `resolveTeam` + `buildPlans`:

```go
package session

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/domain"
)

func setupTestStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// createMinimalTeam creates a team with 2 team members (alice=root, bob=worker)
// and a leader relation. Returns the team and both TeamMember IDs.
func createMinimalTeam(t *testing.T, ctx context.Context, store *db.Store) (domain.Team, string, string) {
	t.Helper()

	sp, _ := domain.NewSystemPrompt("test-prompt", "do things")
	if err := store.CreateSystemPrompt(ctx, sp); err != nil {
		t.Fatalf("CreateSystemPrompt: %v", err)
	}

	profile, _ := domain.NewCliProfileRaw("test-profile", "claude-sonnet-4-6", domain.BinaryClaude,
		[]string{"--dangerously-skip-permissions"}, []string{}, domain.DotConfig{"key": "val"})
	if err := store.CreateCliProfile(ctx, profile); err != nil {
		t.Fatalf("CreateCliProfile: %v", err)
	}

	repo, _ := domain.NewGitRepo("test-repo", "https://example.com/repo.git")
	if err := store.CreateGitRepo(ctx, repo); err != nil {
		t.Fatalf("CreateGitRepo: %v", err)
	}

	root, _ := domain.NewMember("alice", profile.ID, []string{sp.ID}, repo.ID, nil)
	if err := store.CreateMember(ctx, root); err != nil {
		t.Fatalf("CreateMember root: %v", err)
	}

	worker, _ := domain.NewMember("bob", profile.ID, []string{sp.ID}, "", nil)
	if err := store.CreateMember(ctx, worker); err != nil {
		t.Fatalf("CreateMember worker: %v", err)
	}

	team, _ := domain.NewTeam("test-team", root.ID, "alice")
	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	workerTM := domain.TeamMember{ID: uuid.NewString(), MemberID: worker.ID, Name: "bob"}
	if err := store.AddTeamMember(ctx, team.ID, workerTM); err != nil {
		t.Fatalf("AddTeamMember: %v", err)
	}
	rel := domain.Relation{From: team.RootTeamMemberID, To: workerTM.ID}
	if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
		t.Fatalf("AddTeamRelation: %v", err)
	}

	got, err := store.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	return got, team.RootTeamMemberID, workerTM.ID
}

func TestResolveTeam(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user")

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	if len(resolved.Members) != 2 {
		t.Fatalf("resolved %d members, want 2", len(resolved.Members))
	}

	byID := make(map[string]domain.ResolvedMember)
	for _, rm := range resolved.Members {
		byID[rm.TeamMemberID] = rm
	}

	root := byID[rootTMID]
	if root.Name != "alice" {
		t.Errorf("root Name = %q, want alice", root.Name)
	}
	if root.Profile.Model == "" {
		t.Error("root Profile.Model is empty")
	}
	if len(root.Prompts) != 1 {
		t.Errorf("root Prompts = %d, want 1", len(root.Prompts))
	}
	if root.Repo == nil {
		t.Error("root Repo should not be nil")
	}
	if len(root.Relations.Workers) != 1 {
		t.Errorf("root Workers = %d, want 1", len(root.Relations.Workers))
	}

	worker := byID[workerTMID]
	if worker.Name != "bob" {
		t.Errorf("worker Name = %q, want bob", worker.Name)
	}
	if worker.Repo != nil {
		t.Error("worker Repo should be nil")
	}
	if len(worker.Relations.Leaders) != 1 {
		t.Errorf("worker Leaders = %d, want 1", len(worker.Relations.Leaders))
	}
}

func TestBuildPlans(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	team, rootTMID, workerTMID := createMinimalTeam(t, ctx, store)

	svc := New(store, &stubTerminal{}, &stubWorkspace{}, "/tmp/base", "/home/user")

	resolved, err := svc.resolveTeam(ctx, team)
	if err != nil {
		t.Fatalf("resolveTeam: %v", err)
	}

	plans, err := buildPlans(resolved, "test-session")
	if err != nil {
		t.Fatalf("buildPlans: %v", err)
	}

	if len(plans) != 2 {
		t.Fatalf("plans = %d members, want 2", len(plans))
	}

	planByTMID := make(map[string]domain.MemberPlan)
	for _, p := range plans {
		planByTMID[p.TeamMemberID] = p
	}

	rootPlan := planByTMID[rootTMID]
	if rootPlan.MemberName != "alice" {
		t.Errorf("root MemberName = %q, want alice", rootPlan.MemberName)
	}
	if rootPlan.Terminal.Command == "" {
		t.Error("root Terminal.Command is empty")
	}
	if rootPlan.Workspace.GitRepo == nil {
		t.Error("root should have git repo")
	}

	workerPlan := planByTMID[workerTMID]
	if workerPlan.MemberName != "bob" {
		t.Errorf("worker MemberName = %q, want bob", workerPlan.MemberName)
	}
	if workerPlan.Workspace.GitRepo != nil {
		t.Error("worker should not have git repo")
	}
}
```

- [ ] **Step 6: Update prompt_test.go**

In `internal/app/session/prompt_test.go`, change `PromptSnapshot` to `SystemPrompt`:

```go
package session

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestJoinPrompts(t *testing.T) {
	t.Run("SinglePrompt_ReturnsAsIs", func(t *testing.T) {
		prompts := []domain.SystemPrompt{
			{Name: "style", Prompt: "Be concise."},
		}

		got := joinPrompts(prompts)
		if got != "Be concise." {
			t.Errorf("got %q, want %q", got, "Be concise.")
		}
	})

	t.Run("MultiplePrompts_JoinedWithSeparator", func(t *testing.T) {
		prompts := []domain.SystemPrompt{
			{Name: "style", Prompt: "Be concise."},
			{Name: "role", Prompt: "You are a Go developer."},
			{Name: "rules", Prompt: "Follow best practices."},
		}

		got := joinPrompts(prompts)
		want := "Be concise.\n\n---\n\nYou are a Go developer.\n\n---\n\nFollow best practices."
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("NoPrompts_ReturnsEmpty", func(t *testing.T) {
		got := joinPrompts(nil)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("EmptySlice_ReturnsEmpty", func(t *testing.T) {
		got := joinPrompts([]domain.SystemPrompt{})
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}
```

- [ ] **Step 7: Update command_test.go**

In `internal/app/session/command_test.go`, change `EnvSnapshot` to `Env` in tests:

Replace `TestBuildEnv`'s `userEnvs` (line 47-49):
```go
		userEnvs := []domain.Env{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
		}
```

Replace `TestBuildCommand` "WithUserEnvs" block's `userEnvs` (line 136-139):
```go
		userEnvs := []domain.Env{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
			{Key: "SSH_AUTH_SOCK", Value: "/tmp/ssh.sock"},
		}
```

Update `TestBuildCommand` "AllArgs" test (line 107-132) to use the new `buildCommand` signature that accepts `domain.CliProfile`:

```go
	t.Run("AllArgs_IncludesPlaceholders", func(t *testing.T) {
		authEnvs := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + PlaceholderAuthClaude}

		profile := domain.CliProfile{
			Model:      "claude-sonnet-4-6",
			SystemArgs: []string{"--dangerously-skip-permissions"},
			CustomArgs: []string{"--verbose"},
		}
		cmd := buildCommand(profile, "you are a coder", "session-1", "m1", authEnvs, nil)

		for _, want := range []string{
			"claude",
			"--model 'claude-sonnet-4-6'",
			"--dangerously-skip-permissions",
			"--verbose",
			"--append-system-prompt",
			"export CLAUDE_CONFIG_DIR='" + PlaceholderMemberspace + "/.claude'",
			"export CLIER_SESSION_ID='session-1'",
			"export CLIER_MEMBER_ID='m1'",
			"export CLAUDE_CODE_OAUTH_TOKEN='" + PlaceholderAuthClaude + "'",
			"cd '" + PlaceholderMemberspace + "/project'",
		} {
			if !strings.Contains(cmd, want) {
				t.Errorf("missing %q in:\n%s", want, cmd)
			}
		}
	})
```

Update the "WithUserEnvs" test to use the new signature:

```go
	t.Run("WithUserEnvs_BakedIntoCommand", func(t *testing.T) {
		userEnvs := []domain.Env{
			{Key: "GITHUB_TOKEN", Value: "ghp_xxx"},
			{Key: "SSH_AUTH_SOCK", Value: "/tmp/ssh.sock"},
		}

		profile := domain.CliProfile{Model: "opus"}
		cmd := buildCommand(profile, "", "session-1", "m1", nil, userEnvs)

		if !strings.Contains(cmd, "export GITHUB_TOKEN='ghp_xxx'") {
			t.Errorf("missing GITHUB_TOKEN in:\n%s", cmd)
		}
		if !strings.Contains(cmd, "export SSH_AUTH_SOCK='/tmp/ssh.sock'") {
			t.Errorf("missing SSH_AUTH_SOCK in:\n%s", cmd)
		}
	})
```

- [ ] **Step 8: Run all tests**

Run: `go test ./internal/... -v -count=1`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add internal/app/session/plan.go internal/app/session/plan_test.go \
       internal/app/session/prompt.go internal/app/session/prompt_test.go \
       internal/app/session/command.go internal/app/session/command_test.go \
       internal/app/session/service.go
git commit -m "refactor: split buildPlan into resolveTeam + buildPlans

Establish consistent resolve/build vocabulary:
- resolveTeam: ID refs -> loaded domain objects (ResolvedTeam)
- buildPlans: resolved objects -> MemberPlan with placeholders

Update joinPrompts and buildCommand signatures to accept
domain types directly instead of snapshot intermediaries."
```

---

### Task 3: Remove PromptSnapshot/EnvSnapshot, merge plan.go into session.go

**Files:**
- Modify: `internal/domain/session.go` — absorb plan types
- Delete: `internal/domain/plan.go`

- [ ] **Step 1: Move plan types into session.go**

Append the following to the end of `internal/domain/session.go` (after the `NewLog` function). These types are copied from `plan.go`, minus `PromptSnapshot` and `EnvSnapshot`:

```go
// MemberPlan is a fully-resolved execution plan for a single team member.
// Binary, Model, Envs are NOT stored — they are already resolved into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
//
// Plan retains {{CLIER_*}} placeholders; these are expanded at session start
// into concrete paths. The stored plan is safe for name/ID lookups but should
// not be used to reconstruct the workspace without re-expanding placeholders.
type MemberPlan struct {
	TeamMemberID string        `json:"team_member_id"`
	MemberName   string        `json:"member_name"`
	Terminal     TerminalPlan  `json:"terminal"`
	Workspace    WorkspacePlan `json:"workspace"`
}

// TerminalPlan holds the shell command that launches the member agent.
type TerminalPlan struct {
	Command string `json:"command"`
}

// WorkspacePlan holds the filesystem setup for a member's isolated environment.
type WorkspacePlan struct {
	Memberspace string      `json:"memberspace"`
	Files       []FileEntry `json:"files"`
	GitRepo     *GitRepoRef `json:"git_repo"`
}

type GitRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FileEntry is a resolved config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to memberspace
	Content string `json:"content"`
}
```

- [ ] **Step 2: Delete plan.go**

```bash
rm internal/domain/plan.go
```

- [ ] **Step 3: Run all tests**

Run: `go test ./internal/... -v -count=1`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/domain/session.go
git rm internal/domain/plan.go
git commit -m "refactor: merge plan types into session.go, remove snapshots

MemberPlan and related types belong with Session (they are its
execution plan). PromptSnapshot and EnvSnapshot are no longer used
after the resolve/build refactor — removed."
```

---

### Task 4: Rename resolvePlaceholders -> expandPlaceholders

**Files:**
- Rename: `internal/app/session/resolve.go` -> `internal/app/session/expand.go`
- Rename: `internal/app/session/resolve_test.go` -> `internal/app/session/expand_test.go`
- Modify: `internal/app/session/service.go` — update call site

- [ ] **Step 1: Rename files and function**

```bash
git mv internal/app/session/resolve.go internal/app/session/expand.go
git mv internal/app/session/resolve_test.go internal/app/session/expand_test.go
```

- [ ] **Step 2: Update function name in expand.go**

In `internal/app/session/expand.go`, rename the function and update the doc comment:

```go
// expandPlaceholders replaces all {{CLIER_*}} placeholders in a MemberPlan
// and expands ~/ paths to the user's home directory.
// This is the expand phase: plan with placeholders -> plan with concrete paths.
func expandPlaceholders(m domain.MemberPlan, base, homeDir, sessionID, claudeToken string) domain.MemberPlan {
```

- [ ] **Step 3: Update function name in expand_test.go**

In `internal/app/session/expand_test.go`, rename the test function and call:

```go
func TestExpandPlaceholders(t *testing.T) {
```

And update the function call inside from `resolvePlaceholders(` to `expandPlaceholders(`.

- [ ] **Step 4: Update call site in service.go**

In `internal/app/session/service.go`, change `resolvePlaceholders` to `expandPlaceholders`:

```go
		members = append(members, expandPlaceholders(m, s.base, s.homeDir, sessionID, claudeToken))
```

- [ ] **Step 5: Rename resolveAuth -> readAuth in service.go**

In `internal/app/session/service.go`, "resolveAuth" is not a resolve (no DB lookup) — it reads a token. Rename to `readAuth`:

```go
	claudeToken := readAuth(auth)
```

And at the bottom of service.go:

```go
// readAuth reads the Claude auth token.
func readAuth(auth AuthChecker) string {
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/... -v -count=1`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/app/session/expand.go internal/app/session/expand_test.go \
       internal/app/session/service.go
git rm internal/app/session/resolve.go internal/app/session/resolve_test.go
git commit -m "refactor: rename resolvePlaceholders -> expandPlaceholders

Reserve 'resolve' for ID -> DB object loading.
Placeholder substitution is string expansion, so 'expand' is the
correct verb. Also rename resolveAuth -> readAuth for consistency."
```

---

### Task 5: Extract domain/resource/ package

**Files:**
- Create: `internal/domain/resource/env.go`
- Create: `internal/domain/resource/gitrepo.go`
- Create: `internal/domain/resource/systemprompt.go`
- Create: `internal/domain/resource/cliprofile.go`
- Move tests: `internal/domain/resource/env_test.go`
- Move tests: `internal/domain/resource/gitrepo_test.go`
- Move tests: `internal/domain/resource/systemprompt_test.go`
- Move tests: `internal/domain/resource/cliprofile_test.go`
- Modify: all files that reference `domain.{Env,GitRepo,SystemPrompt,CliProfile,DotConfig,CliBinary,BinaryClaude,...}`

- [ ] **Step 1: Create resource directory and move files**

```bash
mkdir -p internal/domain/resource
git mv internal/domain/env.go internal/domain/resource/env.go
git mv internal/domain/env_test.go internal/domain/resource/env_test.go
git mv internal/domain/gitrepo.go internal/domain/resource/gitrepo.go
git mv internal/domain/gitrepo_test.go internal/domain/resource/gitrepo_test.go
git mv internal/domain/systemprompt.go internal/domain/resource/systemprompt.go
git mv internal/domain/systemprompt_test.go internal/domain/resource/systemprompt_test.go
git mv internal/domain/cliprofile.go internal/domain/resource/cliprofile.go
git mv internal/domain/cliprofile_test.go internal/domain/resource/cliprofile_test.go
```

- [ ] **Step 2: Update package declarations in resource files**

In all 8 moved files, change `package domain` to `package resource` (source files) and `package domain_test` to `package resource_test` (test files).

Source files (`env.go`, `gitrepo.go`, `systemprompt.go`, `cliprofile.go`):
```go
package resource
```

Test files (`env_test.go`, `gitrepo_test.go`, `systemprompt_test.go`, `cliprofile_test.go`):
```go
package resource_test
```

- [ ] **Step 3: Update test imports in resource test files**

Each test file imports `"github.com/jakeraft/clier/internal/domain"` and references `domain.X`. Update to import the local package and remove the domain prefix. Since these are now `package resource_test` (external test package), they need to import `resource`:

```go
import (
	"testing"

	"github.com/jakeraft/clier/internal/domain/resource"
)
```

And change all `domain.NewEnv` to `resource.NewEnv`, `domain.NewGitRepo` to `resource.NewGitRepo`, etc.

- [ ] **Step 4: Update domain/member.go to import resource**

`ResolvedMember` references `CliProfile`, `SystemPrompt`, `Env`, `GitRepo`. These now live in `resource` package:

```go
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain/resource"
)
```

Update `ResolvedMember`:
```go
type ResolvedMember struct {
	TeamMemberID string
	Name         string
	Profile      resource.CliProfile
	Prompts      []resource.SystemPrompt
	Envs         []resource.Env
	Repo         *resource.GitRepo
	Relations    MemberRelations
}
```

`Member` itself is unchanged — it only has string IDs, no resource type references.

- [ ] **Step 5: Update domain/session.go**

`DotConfig` is now `resource.DotConfig`. Check if `session.go` references it — it doesn't directly (DotConfig is used in `CliProfile` which is in resource). But `FileEntry` and `MemberPlan` don't reference resource types, so `session.go` needs no resource import.

No changes needed.

- [ ] **Step 6: Update internal/adapter/db/store.go**

Add resource import, update all resource type references:

```go
import (
	...
	"github.com/jakeraft/clier/internal/domain/resource"
)
```

All occurrences of:
- `domain.CliProfile` -> `resource.CliProfile`
- `domain.CliBinary` -> `resource.CliBinary`
- `domain.DotConfig` -> `resource.DotConfig`
- `domain.SystemPrompt` -> `resource.SystemPrompt`
- `domain.Env` -> `resource.Env`
- `domain.GitRepo` -> `resource.GitRepo`

- [ ] **Step 7: Update internal/app/session/ files**

Files: `service.go`, `plan.go`, `command.go`, `config.go`, `config_test.go`, `command_test.go`, `plan_test.go`

Add resource import and update type references:

`service.go` — SessionStore interface:
- `GetCliProfile` returns `resource.CliProfile`
- `GetSystemPrompt` returns `resource.SystemPrompt`
- `GetEnv` returns `resource.Env`
- `GetGitRepo` returns `resource.GitRepo`

`plan.go`:
- `resolveMember` returns `domain.ResolvedMember` which contains resource types (no direct ref needed if ResolvedMember is in domain)
- `buildMemberPlan` references `rm.Profile.DotConfig` — this is `resource.DotConfig` but accessed through the struct, so no import needed in plan.go unless `domain.DotConfig` is referenced directly

`command.go`:
- `buildEnv` param: `userEnvs []resource.Env`
- `buildCommand` param: `profile resource.CliProfile`, `userEnvs []resource.Env`

`config.go`:
- `buildClaudeFiles` param: `dotConfig resource.DotConfig`

Update corresponding test files (`command_test.go`, `config_test.go`, `plan_test.go`) similarly.

- [ ] **Step 8: Update internal/app/session/service_test.go**

Update `stubStore` methods to return resource types:
```go
func (s *stubStore) GetCliProfile(_ context.Context, _ string) (resource.CliProfile, error) {
	return resource.CliProfile{}, errors.New("not implemented")
}
func (s *stubStore) GetSystemPrompt(_ context.Context, _ string) (resource.SystemPrompt, error) {
	return resource.SystemPrompt{}, errors.New("not implemented")
}
func (s *stubStore) GetEnv(_ context.Context, _ string) (resource.Env, error) {
	return resource.Env{}, errors.New("not implemented")
}
func (s *stubStore) GetGitRepo(_ context.Context, _ string) (resource.GitRepo, error) {
	return resource.GitRepo{}, errors.New("not implemented")
}
```

- [ ] **Step 9: Update internal/app/team/service.go**

Update Store interface:
```go
import (
	...
	"github.com/jakeraft/clier/internal/domain/resource"
)
```

```go
type Store interface {
	// Read
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)

	// Write (used by Import)
	CreateSystemPrompt(ctx context.Context, sp *resource.SystemPrompt) error
	CreateEnv(ctx context.Context, e *resource.Env) error
	CreateGitRepo(ctx context.Context, r *resource.GitRepo) error
	CreateCliProfile(ctx context.Context, p *resource.CliProfile) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	UpdateSystemPrompt(ctx context.Context, sp *resource.SystemPrompt) error
	UpdateEnv(ctx context.Context, e *resource.Env) error
	UpdateGitRepo(ctx context.Context, r *resource.GitRepo) error
	UpdateCliProfile(ctx context.Context, p *resource.CliProfile) error
	UpdateMember(ctx context.Context, m *domain.Member) error
	UpdateTeam(ctx context.Context, t *domain.Team) error
	AddTeamMember(ctx context.Context, teamID string, tm domain.TeamMember) error
	AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error
	ReplaceTeamComposition(ctx context.Context, t *domain.Team) error
}
```

- [ ] **Step 10: Update internal/app/team/service_test.go**

```go
import (
	...
	"github.com/jakeraft/clier/internal/domain/resource"
)
```

Update resource constructor calls:
- `domain.NewSystemPrompt` -> `resource.NewSystemPrompt`
- `domain.NewCliProfileRaw` -> `resource.NewCliProfileRaw`
- `domain.BinaryClaude` -> `resource.BinaryClaude`
- `domain.DotConfig` -> `resource.DotConfig`
- `domain.NewGitRepo` -> `resource.NewGitRepo`

- [ ] **Step 11: Update cmd/ files**

Files: `cmd/profile.go`, `cmd/prompt.go`, `cmd/env.go`, `cmd/repo.go`, `cmd/import.go`, `cmd/dashboard.go`

Add `"github.com/jakeraft/clier/internal/domain/resource"` import and update type references.

`cmd/profile.go`: `domain.NewCliProfile` -> `resource.NewCliProfile`
`cmd/prompt.go`: `domain.NewSystemPrompt` -> `resource.NewSystemPrompt`
`cmd/env.go`: `domain.NewEnv` -> `resource.NewEnv`
`cmd/repo.go`: `domain.NewGitRepo` -> `resource.NewGitRepo`
`cmd/import.go`: `domain.CliProfile` -> `resource.CliProfile`, `domain.SystemPrompt` -> `resource.SystemPrompt`, `domain.GitRepo` -> `resource.GitRepo`, `domain.Env` -> `resource.Env`
`cmd/dashboard.go`: `domain.CliProfile` -> `resource.CliProfile`, `domain.SystemPrompt` -> `resource.SystemPrompt`, `domain.GitRepo` -> `resource.GitRepo`, `domain.Env` -> `resource.Env`

- [ ] **Step 12: Update internal/adapter/terminal/ and internal/adapter/workspace/ if needed**

These files use `domain.MemberPlan` which stays in `domain` — likely no changes needed. Verify by checking imports.

- [ ] **Step 13: Run full test suite**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 14: Run linter**

Run: `golangci-lint run ./...`
Expected: PASS

- [ ] **Step 15: Commit**

```bash
git add -A
git commit -m "refactor: extract resource types into domain/resource package

Move Env, GitRepo, SystemPrompt, CliProfile to domain/resource/
to structurally express the domain hierarchy:

  resource (building blocks) -> member -> team -> session

Resource types are independent entities with no cross-references.
The domain package imports resource for ResolvedMember composition."
```

---

## Final Directory Structure

```
internal/domain/
  resource/
    env.go              ← resource.Env
    env_test.go
    gitrepo.go          ← resource.GitRepo
    gitrepo_test.go
    systemprompt.go     ← resource.SystemPrompt
    systemprompt_test.go
    cliprofile.go       ← resource.CliProfile, resource.DotConfig
    cliprofile_test.go
  member.go             ← domain.Member, domain.ResolvedMember
  member_test.go
  team.go               ← domain.Team, domain.TeamMember, domain.Relation, domain.ResolvedTeam
  team_test.go
  session.go            ← domain.Session, domain.MemberPlan, domain.Message, domain.Log
  session_test.go

internal/app/session/
  plan.go               ← resolveTeam(), resolveMember(), buildPlans(), buildMemberPlan()
  expand.go             ← expandPlaceholders()
  expand_test.go
  prompt.go             ← buildClierPrompt(), joinPrompts([]resource.SystemPrompt)
  command.go            ← buildCommand(resource.CliProfile, ..., []resource.Env)
  config.go             ← buildClaudeFiles(resource.DotConfig, ...)
  auth.go               ← setAuth()
  service.go            ← Start() with resolve -> build -> expand -> start flow
```

## Vocabulary Convention

| Verb | Meaning | Example |
|------|---------|---------|
| **resolve** | ID string -> DB -> domain object | `resolveTeam()`, `resolveMember()` |
| **build** | domain objects -> execution artifacts | `buildPlans()`, `buildCommand()`, `buildClierPrompt()` |
| **expand** | placeholder -> concrete value | `expandPlaceholders()` |
| **start** | execute prepared plan | `workspace.Prepare()`, `terminal.Launch()` |
| **read** | read from external source (not DB) | `readAuth()` |
