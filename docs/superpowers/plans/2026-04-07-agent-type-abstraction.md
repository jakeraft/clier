# Agent Type Abstraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename Claude-specific domain entities to agent-neutral names, add `AgentType` to Member, and introduce `AgentRuntime` port with Claude adapter — extracting hardcoded Claude logic from the app layer into a proper hexagonal port/adapter boundary.

**Architecture:** Bottom-up: domain changes first, then DB/store, then AgentRuntime port (app layer) + Claude adapter (adapter layer), then refactor app layer to use the port via DI, then propagate to CLI/UI. The `cmd` layer acts as composition root, wiring runtime adapters into the service.

**Tech Stack:** Go, SQLite, sqlc, React/TypeScript (UI), Cobra (CLI)

---

## Design Overview

### Naming Decisions

| Current | New | Rationale |
|---------|-----|-----------|
| `ClaudeMd` | `AgentDotMd` | Same content across agents; write path differs (CLAUDE.md, AGENTS.md, GEMINI.md) |
| `Settings` | `ClaudeSettings` | Format/schema differs per agent; Claude-specific `settings.json` |
| `ClaudeJson` | `ClaudeJson` | Already Claude-prefixed, stays as-is |
| `Skill` | `Skill` | Shared across agents (all use SKILL.md), stays as-is |
| (new) `Member.AgentType` | `string` | `"claude"` default; determines which runtime to use |

### Hexagonal Structure

```
cmd/ (composition root)
│   runtimes := map[string]task.AgentRuntime{
│       "claude": &runtime.ClaudeRuntime{},
│   }
│   svc := task.New(store, term, ws, base, homeDir, runtimes)
│
├── internal/app/task/                     ← 포트 (인터페이스)
│   ├── runtime.go                         AgentRuntime interface
│   ├── service.go                         Service { runtimes map[string]AgentRuntime }
│   ├── plan.go                            rt := s.runtimes[rm.AgentType]
│   ├── command.go                         buildCommand(rt, ...)
│   └── workspace_files.go                 buildWorkspaceFiles(rt, ...)
│
└── internal/adapter/runtime/              ← 어댑터 (구현체)
    └── claude.go                          ClaudeRuntime struct
```

### AgentRuntime Port

```go
// internal/app/task/runtime.go
type AgentRuntime interface {
    // Command building
    Binary() string                          // "claude" | "codex"
    ConfigDirEnv(memberspace string) string  // "CLAUDE_CONFIG_DIR=/ws/.claude"
    AuthEnvs(token string) []string          // ["CLAUDE_CODE_OAUTH_TOKEN=..."]

    // Workspace layout
    InstructionFile() string                 // "CLAUDE.md" | "AGENTS.md"
    ConfigDir() string                       // ".claude" | ".codex"
    SettingsFile() string                    // "settings.json" | "config.toml"
    ProjectConfigFile() string               // ".claude.json" | (none for codex)
    SkillsDir() string                       // ".claude/skills" | ".agents/skills"
    SystemConfig(memberspace string) string  // Claude onboarding JSON
}
```

### Flow Change

```
Before:
  cmd → task.New(store, term, ws, base, homeDir)
  plan.go → buildCommand(hardcoded "claude") → buildWorkspaceFiles(hardcoded ".claude/")

After:
  cmd → task.New(store, term, ws, base, homeDir, runtimes)
  plan.go → s.runtimes[rm.AgentType] → buildCommand(rt, ...) → buildWorkspaceFiles(rt, ...)
```

---

## File Structure

```
clier/
├── cmd/
│   ├── agentdotmd.go                     🆕 (claudemd.go에서 이름변경)
│   ├── claudemd.go                       🗑️
│   ├── claudejson.go                         유지
│   ├── claudesettings.go                 ✏️ (Settings → ClaudeSettings)
│   ├── dashboard.go                      ✏️ (view types, JSON keys)
│   ├── export.go                         ✏️ (envelope type)
│   ├── import.go                         ✏️ (envelope type)
│   ├── member.go                         ✏️ (flags, params)
│   ├── task.go                           ✏️ (runtimes wire-up)
│   ├── tutorial.go                       ✏️ (runtimes wire-up)
│   └── ...                                   유지
│
├── internal/
│   ├── adapter/
│   │   ├── runtime/                      🆕 신규 패키지
│   │   │   ├── claude.go                 🆕 ClaudeRuntime 어댑터
│   │   │   └── claude_test.go            🆕 ClaudeRuntime 테스트
│   │   ├── db/
│   │   │   ├── schema.sql                ✏️ (테이블명, 컬럼명)
│   │   │   ├── store.go                  ✏️ (메서드명)
│   │   │   ├── store_test.go             ✏️
│   │   │   ├── queries/
│   │   │   │   ├── agent_dot_md.sql      🆕 (claude_md.sql에서)
│   │   │   │   ├── claude_md.sql         🗑️
│   │   │   │   ├── claude_settings.sql   🆕 (settings.sql에서)
│   │   │   │   ├── settings.sql          🗑️
│   │   │   │   ├── member.sql            ✏️ (컬럼명)
│   │   │   │   └── ...                       유지
│   │   │   └── generated/                🔄 전체 재생성 (sqlc generate)
│   │   ├── settings/                         유지 (Claude auth adapter)
│   │   ├── terminal/                         유지
│   │   └── workspace/                        유지
│   │
│   ├── app/
│   │   └── task/
│   │       ├── runtime.go                🆕 AgentRuntime 인터페이스 (포트)
│   │       ├── command.go                ✏️ (rt AgentRuntime 파라미터)
│   │       ├── command_test.go           ✏️
│   │       ├── workspace_files.go        ✏️ (rt로 경로 결정 + mergeJSON 주석)
│   │       ├── workspace_files_test.go   ✏️
│   │       ├── plan.go                   ✏️ (s.runtimes[rm.AgentType])
│   │       ├── plan_test.go              ✏️
│   │       ├── expand.go                 ✏️ (파라미터명 정리)
│   │       ├── expand_test.go            ✏️
│   │       ├── service.go               ✏️ (runtimes 필드, TaskStore 변경)
│   │       ├── service_test.go           ✏️
│   │       └── prompt.go                     유지
│   │
│   └── domain/
│       ├── member.go                     ✏️ (AgentType 추가, 필드명)
│       ├── member_test.go                ✏️
│       ├── team.go                           유지
│       ├── team_test.go                      유지
│       ├── task.go                           유지
│       ├── task_test.go                      유지
│       └── resource/
│           ├── agent_dot_md.go           🆕 (claudemd.go에서)
│           ├── agent_dot_md_test.go      🆕 (claudemd_test.go에서)
│           ├── claudemd.go               🗑️
│           ├── claudemd_test.go          🗑️
│           ├── claude_settings.go        🆕 (settings.go에서)
│           ├── claude_settings_test.go   🆕 (settings_test.go에서)
│           ├── settings.go               🗑️
│           ├── settings_test.go          🗑️
│           ├── claudejson.go                 유지
│           ├── skill.go                      유지
│           └── ...                           유지
│
├── tutorials/todo-team/
│   ├── index.json                        ✏️
│   ├── agent-dot-md-tech-lead.json       🆕 (claude-md-tech-lead.json에서)
│   ├── agent-dot-md-coder.json           🆕 (claude-md-coder.json에서)
│   ├── agent-dot-md-reviewer.json        🆕 (claude-md-reviewer.json에서)
│   ├── claude-md-*.json                  🗑️ (3개)
│   ├── settings-default.json             ✏️ (envelope type)
│   ├── member-*.json                     ✏️ (필드명, 3개)
│   └── ...                                   유지
│
└── ui/src/
    ├── types.ts                          ✏️
    ├── api.ts                            ✏️
    ├── app.tsx                           ✏️ (라우트)
    ├── lib/entities.ts                   ✏️ (entity type)
    └── pages/
        ├── agent-dot-md-detail.tsx        🆕 (claude-md-detail.tsx에서)
        ├── claude-md-detail.tsx           🗑️
        ├── claude-settings-detail.tsx     🆕 (settings-detail.tsx에서)
        ├── settings-detail.tsx            🗑️
        ├── claude-config.tsx              ✏️ (ClaudeSettingsView)
        ├── prompts.tsx                    ✏️ (AgentDotMdView)
        ├── member-detail.tsx              ✏️ (필드명)
        └── ...                               유지
```

### Summary

| 상태 | 파일 수 |
|------|---------|
| 🆕 신규 | 13 (runtime 포트 1 + runtime 어댑터 2 + 도메인 rename 4 + DB queries 2 + cmd 1 + tutorial 3 + UI 2) |
| 🗑️ 삭제 | 13 (rename 전 원본들) |
| ✏️ 수정 | 30 |
| 🔄 재생성 | ~12 (sqlc generated) |

---

### Task 1: Domain Entity Renames + AgentType

**Files:**
- Create: `internal/domain/resource/agent_dot_md.go`
- Create: `internal/domain/resource/agent_dot_md_test.go`
- Create: `internal/domain/resource/claude_settings.go`
- Create: `internal/domain/resource/claude_settings_test.go`
- Delete: `internal/domain/resource/claudemd.go`, `claudemd_test.go`, `settings.go`, `settings_test.go`
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/member_test.go`

- [ ] **Step 1: Create agent_dot_md.go (replacing claudemd.go)**

```go
// internal/domain/resource/agent_dot_md.go
package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentDotMd is a project instruction file shared across agent types.
// Written as CLAUDE.md (Claude), AGENTS.md (Codex), or GEMINI.md (Gemini).
type AgentDotMd struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAgentDotMd(name, content string) (*AgentDotMd, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("agent dot md name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("agent dot md content must not be empty")
	}

	now := time.Now()
	return &AgentDotMd{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a *AgentDotMd) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("agent dot md name must not be empty")
		}
		a.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("agent dot md content must not be empty")
		}
		a.Content = trimmed
	}
	a.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 2: Create agent_dot_md_test.go**

```go
package resource

import "testing"

func TestNewAgentDotMd(t *testing.T) {
	md, err := NewAgentDotMd("my-rules", "# Project Rules\n\nAlways use TDD.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.ID == "" {
		t.Error("ID should not be empty")
	}
	if md.Name != "my-rules" {
		t.Errorf("name = %q, want %q", md.Name, "my-rules")
	}
}

func TestNewAgentDotMd_EmptyName(t *testing.T) {
	_, err := NewAgentDotMd("", "content")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNewAgentDotMd_EmptyContent(t *testing.T) {
	_, err := NewAgentDotMd("name", "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestAgentDotMd_Update(t *testing.T) {
	md, _ := NewAgentDotMd("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := md.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "new" {
		t.Errorf("name = %q", md.Name)
	}
}
```

- [ ] **Step 3: Create claude_settings.go (replacing settings.go)**

```go
// internal/domain/resource/claude_settings.go
package resource

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeSettings is a settings.json file for Claude Code.
// Written to CLAUDE_CONFIG_DIR/settings.json.
type ClaudeSettings struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeSettings(name, content string) (*ClaudeSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude settings name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude settings content must not be empty")
	}
	if !json.Valid([]byte(content)) {
		return nil, errors.New("claude settings content must be valid JSON")
	}
	now := time.Now()
	return &ClaudeSettings{ID: uuid.NewString(), Name: name, Content: content, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *ClaudeSettings) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude settings name must not be empty")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude settings content must not be empty")
		}
		if !json.Valid([]byte(trimmed)) {
			return errors.New("claude settings content must be valid JSON")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 4: Create claude_settings_test.go (replacing settings_test.go)**

```go
package resource

import "testing"

func TestNewClaudeSettings(t *testing.T) {
	s, err := NewClaudeSettings("skip-permissions", `{"skipDangerousModePermissionPrompt":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "skip-permissions" {
		t.Errorf("name = %q", s.Name)
	}
}

func TestNewClaudeSettings_EmptyName(t *testing.T) {
	_, err := NewClaudeSettings("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeSettings_EmptyContent(t *testing.T) {
	_, err := NewClaudeSettings("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestNewClaudeSettings_InvalidJSON(t *testing.T) {
	_, err := NewClaudeSettings("name", "not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestClaudeSettings_Update(t *testing.T) {
	s, _ := NewClaudeSettings("old", `{"old":true}`)
	newName := "new"
	newContent := `{"new":true}`
	if err := s.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "new" || s.Content != `{"new":true}` {
		t.Error("update did not apply")
	}
}
```

- [ ] **Step 5: Delete old files**

```bash
rm internal/domain/resource/claudemd.go internal/domain/resource/claudemd_test.go
rm internal/domain/resource/settings.go internal/domain/resource/settings_test.go
```

- [ ] **Step 6: Update Member — add AgentType, rename fields**

In `internal/domain/member.go`, the `Member` struct becomes:

```go
type Member struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	AgentType        string    `json:"agent_type"`
	Model            string    `json:"model"`
	Args             []string  `json:"args"`
	AgentDotMdID     string    `json:"agent_dot_md_id"`
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`
	ClaudeJsonID     string    `json:"claude_json_id"`
	EnvIDs           []string  `json:"env_ids"`
	GitRepoID        string    `json:"git_repo_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
```

`NewMember` adds `agentType` param (defaults to `"claude"` if empty).
`Update` adds `agentType *string` param.
`ResolvedMember` fields: `AgentType string`, `AgentDotMd *resource.AgentDotMd`, `ClaudeSettings *resource.ClaudeSettings`.

- [ ] **Step 7: Update member_test.go**

All `NewMember` calls add `"claude"` as second arg. Field assertions rename accordingly.

- [ ] **Step 8: Run domain tests**

Run: `go test ./internal/domain/... ./internal/domain/resource/...`

- [ ] **Step 9: Commit**

```bash
git add internal/domain/
git commit -m "refactor: rename ClaudeMd -> AgentDotMd, Settings -> ClaudeSettings, add Member.AgentType"
```

---

### Task 2: DB Schema + sqlc Regeneration

**Files:**
- Modify: `internal/adapter/db/schema.sql`
- Create: `internal/adapter/db/queries/agent_dot_md.sql`
- Create: `internal/adapter/db/queries/claude_settings.sql`
- Modify: `internal/adapter/db/queries/member.sql`
- Delete: `internal/adapter/db/queries/claude_md.sql`, `settings.sql`
- Regenerate: `internal/adapter/db/generated/*.go`

- [ ] **Step 1: Update schema.sql**

Rename tables: `claude_mds` → `agent_dot_mds`, `settings` → `claude_settings`.
Add `agent_type TEXT NOT NULL DEFAULT 'claude'` to members.
Rename columns: `claude_md_id` → `agent_dot_md_id`, `settings_id` → `claude_settings_id`.

- [ ] **Step 2: Create queries/agent_dot_md.sql**

Standard CRUD against `agent_dot_mds` table (CreateAgentDotMd, GetAgentDotMd, ListAgentDotMds, UpdateAgentDotMd, DeleteAgentDotMd).

- [ ] **Step 3: Create queries/claude_settings.sql**

Standard CRUD against `claude_settings` table (CreateClaudeSettings, GetClaudeSettings, ListClaudeSettings, UpdateClaudeSettings, DeleteClaudeSettings).

- [ ] **Step 4: Update queries/member.sql**

Add `agent_type` to INSERT/UPDATE. Rename `claude_md_id` → `agent_dot_md_id`, `settings_id` → `claude_settings_id`.

- [ ] **Step 5: Delete old query files and regenerate**

```bash
rm internal/adapter/db/queries/claude_md.sql internal/adapter/db/queries/settings.sql
cd internal/adapter/db && sqlc generate
```

- [ ] **Step 6: Verify build**

```bash
go build ./internal/adapter/db/generated/...
```

- [ ] **Step 7: Commit**

```bash
git add internal/adapter/db/
git commit -m "refactor: rename DB tables claude_mds -> agent_dot_mds, settings -> claude_settings, add agent_type"
```

---

### Task 3: Store Adapter

**Files:**
- Modify: `internal/adapter/db/store.go`

- [ ] **Step 1: Rename AgentDotMd CRUD methods**

`CreateClaudeMd` → `CreateAgentDotMd`, `GetClaudeMd` → `GetAgentDotMd`, etc.
All `resource.ClaudeMd` → `resource.AgentDotMd`, generated params likewise.

- [ ] **Step 2: Rename ClaudeSettings CRUD methods**

`CreateSettings` → `CreateClaudeSettings`, `GetSettings` → `GetClaudeSettings`, etc.
All `resource.Settings` → `resource.ClaudeSettings`, generated params likewise.

- [ ] **Step 3: Update Member CRUD — add AgentType, rename FK fields**

`ClaudeMdID` → `AgentDotMdID`, `SettingsID` → `ClaudeSettingsID` in CreateMember, GetMember, ListMembers, UpdateMember. Add `AgentType` field mapping.

- [ ] **Step 4: Run store tests**

```bash
go test ./internal/adapter/db/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/db/store.go
git commit -m "refactor: rename store methods for AgentDotMd/ClaudeSettings, add AgentType"
```

---

### Task 4: AgentRuntime Port + Claude Adapter

**Files:**
- Create: `internal/app/task/runtime.go` (port — interface only)
- Create: `internal/adapter/runtime/claude.go` (adapter)
- Create: `internal/adapter/runtime/claude_test.go` (adapter tests)

- [ ] **Step 1: Create runtime.go (port)**

```go
// internal/app/task/runtime.go
package task

// AgentRuntime provides agent-specific behavior for command building
// and workspace layout. Each supported agent type has its own implementation.
type AgentRuntime interface {
	// Command building
	Binary() string
	ConfigDirEnv(memberspace string) string
	AuthEnvs(token string) []string

	// Workspace layout
	InstructionFile() string
	ConfigDir() string
	SettingsFile() string
	ProjectConfigFile() string
	SkillsDir() string
	SystemConfig(memberspace string) string
}
```

- [ ] **Step 2: Write failing tests for ClaudeRuntime**

```go
// internal/adapter/runtime/claude_test.go
package runtime

import (
	"strings"
	"testing"
)

func TestClaudeRuntime_Binary(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.Binary() != "claude" {
		t.Errorf("Binary() = %q, want %q", rt.Binary(), "claude")
	}
}

func TestClaudeRuntime_ConfigDirEnv(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.ConfigDirEnv("/ws")
	if got != "CLAUDE_CONFIG_DIR=/ws/.claude" {
		t.Errorf("ConfigDirEnv() = %q", got)
	}
}

func TestClaudeRuntime_AuthEnvs(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.AuthEnvs("sk-token")
	if len(got) != 1 || got[0] != "CLAUDE_CODE_OAUTH_TOKEN=sk-token" {
		t.Errorf("AuthEnvs() = %v", got)
	}
}

func TestClaudeRuntime_InstructionFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.InstructionFile() != "CLAUDE.md" {
		t.Errorf("InstructionFile() = %q", rt.InstructionFile())
	}
}

func TestClaudeRuntime_ConfigDir(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.ConfigDir() != ".claude" {
		t.Errorf("ConfigDir() = %q", rt.ConfigDir())
	}
}

func TestClaudeRuntime_SettingsFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.SettingsFile() != "settings.json" {
		t.Errorf("SettingsFile() = %q", rt.SettingsFile())
	}
}

func TestClaudeRuntime_ProjectConfigFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.ProjectConfigFile() != ".claude.json" {
		t.Errorf("ProjectConfigFile() = %q", rt.ProjectConfigFile())
	}
}

func TestClaudeRuntime_SkillsDir(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.SkillsDir() != ".claude/skills" {
		t.Errorf("SkillsDir() = %q", rt.SkillsDir())
	}
}

func TestClaudeRuntime_SystemConfig(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.SystemConfig("/ws")
	if !strings.Contains(got, "hasCompletedOnboarding") {
		t.Errorf("SystemConfig missing expected content: %q", got)
	}
	if !strings.Contains(got, "/ws/project") {
		t.Errorf("SystemConfig missing workspace path: %q", got)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/adapter/runtime/... -v
```

Expected: FAIL (types not defined).

- [ ] **Step 4: Create claude.go (adapter)**

```go
// internal/adapter/runtime/claude.go
package runtime

import "fmt"

// ClaudeRuntime implements task.AgentRuntime for Claude Code.
type ClaudeRuntime struct{}

func (c *ClaudeRuntime) Binary() string { return "claude" }

func (c *ClaudeRuntime) ConfigDirEnv(memberspace string) string {
	return "CLAUDE_CONFIG_DIR=" + memberspace + "/.claude"
}

func (c *ClaudeRuntime) AuthEnvs(token string) []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN=" + token}
}

func (c *ClaudeRuntime) InstructionFile() string    { return "CLAUDE.md" }
func (c *ClaudeRuntime) ConfigDir() string           { return ".claude" }
func (c *ClaudeRuntime) SettingsFile() string        { return "settings.json" }
func (c *ClaudeRuntime) ProjectConfigFile() string   { return ".claude.json" }
func (c *ClaudeRuntime) SkillsDir() string           { return ".claude/skills" }

func (c *ClaudeRuntime) SystemConfig(memberspace string) string {
	return fmt.Sprintf(`{"hasCompletedOnboarding":true,"projects":{"%s/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`, memberspace)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/adapter/runtime/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/app/task/runtime.go internal/adapter/runtime/
git commit -m "feat: add AgentRuntime port + ClaudeRuntime adapter"
```

---

### Task 5: Refactor App Layer to Use AgentRuntime

**Files:**
- Modify: `internal/app/task/service.go` (add runtimes field)
- Modify: `internal/app/task/command.go` (rt param)
- Modify: `internal/app/task/command_test.go`
- Modify: `internal/app/task/workspace_files.go` (rt param + mergeJSON 주석)
- Modify: `internal/app/task/workspace_files_test.go`
- Modify: `internal/app/task/plan.go` (s.runtimes, rename fields)
- Modify: `internal/app/task/plan_test.go`
- Modify: `internal/app/task/expand.go` (rename param)
- Modify: `internal/app/task/expand_test.go`
- Modify: `internal/app/task/service_test.go`

- [ ] **Step 1: Update service.go — add runtimes, rename TaskStore**

```go
type TaskStore interface {
	// ... existing Task/Team methods ...
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetAgentDotMd(ctx context.Context, id string) (resource.AgentDotMd, error)
	GetSkill(ctx context.Context, id string) (resource.Skill, error)
	GetClaudeSettings(ctx context.Context, id string) (resource.ClaudeSettings, error)
	GetClaudeJson(ctx context.Context, id string) (resource.ClaudeJson, error)
	GetEnv(ctx context.Context, id string) (resource.Env, error)
	GetGitRepo(ctx context.Context, id string) (resource.GitRepo, error)
}

type Service struct {
	store    TaskStore
	terminal Terminal
	workspace Workspace
	base      string
	homeDir   string
	runtimes  map[string]AgentRuntime  // agent type -> runtime
}

func New(store TaskStore, term Terminal, ws Workspace, base, homeDir string, runtimes map[string]AgentRuntime) *Service {
	return &Service{store: store, terminal: term, workspace: ws, base: base, homeDir: homeDir, runtimes: runtimes}
}
```

Pass `s.runtimes` to `buildPlans()`.

- [ ] **Step 2: Refactor command.go — add rt param, remove hardcoded values**

Replace `configDirEnv()` and `authEnvs()` with runtime-delegating versions:

```go
func systemEnvs(rt AgentRuntime, memberspace, taskID, memberID string) []string {
	return []string{
		rt.ConfigDirEnv(memberspace),
		"CLIER_TASK_ID=" + taskID,
		"CLIER_MEMBER_ID=" + memberID,
	}
}

func buildAgentCommand(rt AgentRuntime, model string, args []string, workDir string) string {
	parts := []string{rt.Binary()}
	// ...
}

func buildCommand(rt AgentRuntime, model string, args []string, workDir, memberspace, teamName, memberName, taskID, memberID, authPlaceholder string, userEnvs []resource.Env) string {
	cmd := buildAgentCommand(rt, model, args, workDir)
	env := buildEnv(rt, memberspace, teamName, memberName, taskID, memberID, authPlaceholder, userEnvs)
	return buildEnvCommand(cmd, env)
}
```

- [ ] **Step 3: Refactor workspace_files.go — use runtime for paths**

```go
func buildWorkspaceFiles(rt AgentRuntime, memberspace, systemAgentDotMd, userAgentDotMd, userClaudeSettings, systemProjectConfig, userProjectConfig string, userSkills []resource.Skill) []domain.FileEntry {
	// rt.InstructionFile()    -> "CLAUDE.md"
	// rt.ConfigDir()          -> ".claude"
	// rt.SettingsFile()       -> "settings.json"
	// rt.ProjectConfigFile()  -> ".claude.json"
	// rt.SkillsDir()          -> ".claude/skills"
}
```

Remove standalone `buildSystemClaudeJson()` (logic now in `ClaudeRuntime.SystemConfig()`).

Add comment above `mergeJSON`:
```go
// NOTE: Claude-specific JSON merge. Other agent runtimes may need different merge strategy (e.g. TOML for Codex).
```

- [ ] **Step 4: Refactor plan.go — use s.runtimes**

```go
func buildPlans(resolved *domain.ResolvedTeam, taskID string, runtimes map[string]AgentRuntime) []domain.MemberPlan {
	// ...
	plan := buildMemberPlan(&rm, nameByID, resolved.Name, taskID, runtimes)
}

func buildMemberPlan(rm *domain.ResolvedMember, nameByID map[string]string, teamName, taskID string, runtimes map[string]AgentRuntime) domain.MemberPlan {
	rt := runtimes[rm.AgentType]
	if rt == nil {
		rt = runtimes["claude"]
	}

	systemAgentDotMd := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID)
	var userAgentDotMd string
	if rm.AgentDotMd != nil {
		userAgentDotMd = rm.AgentDotMd.Content
	}

	var userClaudeSettings string
	if rm.ClaudeSettings != nil {
		userClaudeSettings = rm.ClaudeSettings.Content
	}

	systemProjectConfig := rt.SystemConfig(PlaceholderMemberspace)
	// ...

	files := buildWorkspaceFiles(rt, PlaceholderMemberspace, systemAgentDotMd, userAgentDotMd, userClaudeSettings, systemProjectConfig, userProjectConfig, userSkills)

	cmd := buildCommand(rt, rm.Model, rm.Args, PlaceholderMemberspace+"/project",
		PlaceholderMemberspace, teamName, rm.Name, taskID, rm.TeamMemberID, PlaceholderAuthClaude, rm.Envs)
	// ...
}
```

In `resolveMember`: rename `claudeMd` → `agentDotMd`, `settings` → `claudeSettings`, use `member.AgentDotMdID`, `s.store.GetAgentDotMd`, etc. Add `AgentType: member.AgentType` to result.

- [ ] **Step 5: Update expand.go — rename param**

Rename `claudeToken` to `authToken` (cosmetic). `PlaceholderAuthClaude` stays as-is.

- [ ] **Step 6: Update all test files**

`command_test.go`: All build functions receive `&ClaudeRuntime{}` (import from `internal/adapter/runtime`).
`workspace_files_test.go`: Same — pass `&ClaudeRuntime{}` as first arg.
`plan_test.go`: Use `NewAgentDotMd`, `NewClaudeSettings`, `NewMember` with agentType. Pass runtimes map.
`service_test.go`: Rename stub methods, update `New()` calls to include runtimes.
`expand_test.go`: Rename `claudeToken` param.

- [ ] **Step 7: Run all app tests**

```bash
go test ./internal/app/...
```

- [ ] **Step 8: Commit**

```bash
git add internal/app/task/
git commit -m "refactor: thread AgentRuntime through command/workspace/plan via DI"
```

---

### Task 6: CLI Commands + Wire-up

**Files:**
- Create: `cmd/agentdotmd.go` (replaces `cmd/claudemd.go`)
- Delete: `cmd/claudemd.go`
- Modify: `cmd/claudesettings.go`
- Modify: `cmd/member.go`
- Modify: `cmd/dashboard.go`
- Modify: `cmd/task.go` (runtimes wire-up)
- Modify: `cmd/tutorial.go` (runtimes wire-up)

- [ ] **Step 1: Create cmd/agentdotmd.go**

Command `agent-dot-md` with CRUD subcommands. Uses `resource.NewAgentDotMd`, `store.CreateAgentDotMd`, etc.

- [ ] **Step 2: Delete cmd/claudemd.go**

- [ ] **Step 3: Update cmd/claudesettings.go**

`resource.NewSettings` → `resource.NewClaudeSettings`, `store.CreateSettings` → `store.CreateClaudeSettings`, etc.

- [ ] **Step 4: Update cmd/member.go**

Rename flags: `--claude-md` → `--agent-dot-md`, `--settings` → `--claude-settings`.
Add `"claude"` as agentType in `NewMember` calls.

- [ ] **Step 5: Wire up runtimes in cmd/task.go and cmd/tutorial.go**

```go
import agentrt "github.com/jakeraft/clier/internal/adapter/runtime"

runtimes := map[string]task.AgentRuntime{
	"claude": &agentrt.ClaudeRuntime{},
}
svc := task.New(store, term, ws, cfg.Paths.Workspaces(), cfg.Paths.HomeDir(), runtimes)
```

Apply to all 5 `task.New()` call sites (4 in task.go, 1 in tutorial.go).

- [ ] **Step 6: Update cmd/dashboard.go**

Rename view types: `claudeMdView` → `agentDotMdView`, `settingsView` → `claudeSettingsView`.
Rename JSON keys: `"claudeMds"` → `"agentDotMds"`, `"settings"` → `"claudeSettings"`.
Add `AgentType` to `memberView`.
Rename conversion functions and store method calls.

- [ ] **Step 7: Run cmd tests**

```bash
go test ./cmd/...
```

- [ ] **Step 8: Commit**

```bash
git add cmd/
git commit -m "refactor: rename CLI commands, wire AgentRuntime into task service"
```

---

### Task 7: Import/Export

**Files:**
- Modify: `cmd/import.go`
- Modify: `cmd/export.go`

- [ ] **Step 1: Update import.go**

Envelope `"claude_md"` → `"agent_dot_md"`, `"settings"` → `"claude_settings"`.
Types: `resource.ClaudeMd` → `resource.AgentDotMd`, `resource.Settings` → `resource.ClaudeSettings`.
Store calls: rename accordingly. Member import: `m.AgentDotMdID`, `m.ClaudeSettingsID`.

- [ ] **Step 2: Update export.go**

Probe types: `"claude_md"` → `"agent_dot_md"`, `"settings"` → `"claude_settings"`.
Store calls: rename accordingly.

- [ ] **Step 3: Commit**

```bash
git add cmd/import.go cmd/export.go
git commit -m "refactor: rename envelope types for import/export"
```

---

### Task 8: Tutorials

**Files:**
- Rename + modify: `tutorials/todo-team/claude-md-*.json` → `agent-dot-md-*.json`
- Modify: `tutorials/todo-team/settings-default.json`
- Modify: `tutorials/todo-team/member-*.json`
- Modify: `tutorials/todo-team/index.json`

- [ ] **Step 1: Rename and update tutorial files**

```bash
mv tutorials/todo-team/claude-md-tech-lead.json tutorials/todo-team/agent-dot-md-tech-lead.json
mv tutorials/todo-team/claude-md-coder.json tutorials/todo-team/agent-dot-md-coder.json
mv tutorials/todo-team/claude-md-reviewer.json tutorials/todo-team/agent-dot-md-reviewer.json
```

In each: `"type": "claude_md"` → `"type": "agent_dot_md"`.
In `settings-default.json`: `"type": "settings"` → `"type": "claude_settings"`.
In member JSONs: `"claude_md_id"` → `"agent_dot_md_id"`, `"settings_id"` → `"claude_settings_id"`, add `"agent_type": "claude"`.

- [ ] **Step 2: Update index.json**

Replace `claude-md-*` filenames with `agent-dot-md-*`.

- [ ] **Step 3: Commit**

```bash
git add tutorials/
git commit -m "refactor: rename tutorial files for agent type abstraction"
```

---

### Task 9: UI

**Files:**
- Modify: `ui/src/types.ts`, `api.ts`, `app.tsx`, `lib/entities.ts`
- Create: `ui/src/pages/agent-dot-md-detail.tsx`, `claude-settings-detail.tsx`
- Delete: `ui/src/pages/claude-md-detail.tsx`, `settings-detail.tsx`
- Modify: `ui/src/pages/claude-config.tsx`, `prompts.tsx`, `member-detail.tsx`

- [ ] **Step 1: Update types.ts**

`ClaudeMdView` → `AgentDotMdView`, `SettingsView` → `ClaudeSettingsView`.
Dashboard data keys: `claudeMds` → `agentDotMds`, `settings` → `claudeSettings`.
Member fields: `claudeMdId` → `agentDotMdId`, `settingsId` → `claudeSettingsId`, add `agentType`.

- [ ] **Step 2: Update api.ts**

`claudeMds` → `agentDotMds`, `settings` → `claudeSettings`.

- [ ] **Step 3: Update entities.ts**

Entity type `"claude-md"` → `"agent-dot-md"`. Route mappings update accordingly.

- [ ] **Step 4: Create agent-dot-md-detail.tsx, claude-settings-detail.tsx**

Copy from old files, rename component names and API refs. Delete originals.

- [ ] **Step 5: Update prompts.tsx, claude-config.tsx, member-detail.tsx**

Rename types, API refs, entity types, route bases, field names.

- [ ] **Step 6: Update app.tsx routes**

Import new detail pages, update route paths.

- [ ] **Step 7: Build UI**

```bash
cd ui && npm run build
```

- [ ] **Step 8: Commit**

```bash
git add ui/
git commit -m "refactor: rename UI types and routes for agent type abstraction"
```

---

### Task 10: Integration Test

- [ ] **Step 1: Run all Go tests**

```bash
go test ./...
```

- [ ] **Step 2: Build UI**

```bash
cd ui && npm run build
```

- [ ] **Step 3: Test tutorial import**

```bash
go run . import tutorials/todo-team
```

- [ ] **Step 4: Test dashboard**

```bash
go run . dashboard
```

- [ ] **Step 5: Final fixup commit if needed**
