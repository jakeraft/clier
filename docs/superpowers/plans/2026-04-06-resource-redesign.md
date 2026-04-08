# Resource Redesign: Claude Code Transparent Mapping

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace CliProfile and SystemPrompt with building blocks that transparently map 1:1 to Claude Code features (ClaudeMd, Skill, Settings, ClaudeJson), and restructure Member to hold Model/Args directly.

**Architecture:** Remove the CliProfile bundle and SystemPrompt abstraction. Each new resource corresponds to exactly one file Claude Code reads. Member directly holds model and args (CLI flags). The build phase clearly separates user-defined building blocks from Clier system-generated infrastructure. Every resource follows the same pattern: user provides pure content, Clier system merges runtime-generated parts (no placeholders — system injection replaces {{CLIER_*}} pattern).

**Tech Stack:** Go, SQLite, sqlc, React/TypeScript (Vite), Cobra CLI

---

## File Structure

### Files to CREATE

```
internal/domain/resource/claudemd.go        — ClaudeMd domain entity
internal/domain/resource/claudemd_test.go    — ClaudeMd tests
internal/domain/resource/skill.go            — Skill domain entity
internal/domain/resource/skill_test.go       — Skill tests
internal/domain/resource/settings.go         — Settings domain entity
internal/domain/resource/settings_test.go    — Settings tests
internal/domain/resource/claudejson.go       — ClaudeJson domain entity
internal/domain/resource/claudejson_test.go  — ClaudeJson tests
internal/adapter/db/queries/claude_md.sql    — ClaudeMd CRUD queries
internal/adapter/db/queries/skill.sql        — Skill CRUD queries
internal/adapter/db/queries/settings.sql     — Settings CRUD queries
internal/adapter/db/queries/claude_json.sql  — ClaudeJson CRUD queries
cmd/claudemd.go                              — clier claude-md CLI subcommand
cmd/skill.go                                 — clier skill CLI subcommand
cmd/claudesettings.go                        — clier claude-settings CLI subcommand
cmd/claudejson.go                            — clier claude-json CLI subcommand
ui/src/pages/claude-mds.tsx                  — ClaudeMd list page
ui/src/pages/claude-md-detail.tsx            — ClaudeMd detail page
ui/src/pages/skills.tsx                      — Skill list page
ui/src/pages/skill-detail.tsx                — Skill detail page
ui/src/pages/settings-list.tsx               — Settings list page
ui/src/pages/settings-detail.tsx             — Settings detail page
ui/src/pages/claude-jsons.tsx                — ClaudeJson list page
ui/src/pages/claude-json-detail.tsx          — ClaudeJson detail page
```

### Files to DELETE

```
internal/domain/resource/cliprofile.go
internal/domain/resource/cliprofile_test.go
internal/domain/resource/systemprompt.go
internal/domain/resource/systemprompt_test.go
internal/adapter/db/queries/cli_profile.sql
internal/adapter/db/queries/system_prompt.sql
cmd/profile.go
cmd/prompt.go
ui/src/pages/cli-profiles.tsx
ui/src/pages/cli-profile-detail.tsx
ui/src/pages/system-prompts.tsx
ui/src/pages/system-prompt-detail.tsx
```

### Files to MODIFY

```
# Domain
internal/domain/member.go                    — Replace CliProfileID/SystemPromptIDs with Model/Args/new resource IDs
internal/domain/member_test.go               — Update tests for new Member fields
internal/domain/task.go                      — No change (MemberPlan stays as-is)

# Database
internal/adapter/db/schema.sql               — Drop cli_profiles/system_prompts/member_system_prompts, add new tables/columns
internal/adapter/db/queries/member.sql        — Update member queries for new columns/junctions
internal/adapter/db/store.go                 — Replace CliProfile/SystemPrompt methods with new resource methods, update Member methods

# App - Task build
internal/app/task/service.go                 — Update TaskStore interface (new resource getters)
internal/app/task/plan.go                    — Update buildMemberPlan to clearly show building block → execution mapping
internal/app/task/command.go                 — Remove CliProfile dependency, take model/args directly
internal/app/task/config.go                  — Rename to workspace_files.go, build from individual resources
internal/app/task/config_test.go             — Rename to workspace_files_test.go, update tests
internal/app/task/expand.go                  — Placeholders now system-internal only, user content passes through unchanged
internal/app/task/expand_test.go             — Verify user content unchanged, system placeholders expanded
internal/app/task/prompt.go                  — Update to work with ClaudeMd content instead of SystemPrompt
internal/app/task/plan_test.go               — Update for new structure
internal/app/task/command_test.go            — Update for new function signatures
internal/app/task/prompt_test.go             — Update for ClaudeMd
internal/app/task/service_test.go            — Update mock store

# App - Team import
internal/app/team/service.go                 — Update Store interface for new resources
internal/app/team/service_test.go            — Update tests

# CLI
cmd/member.go                                — Update create/update flags for new Member fields
cmd/import.go                                — Handle new envelope types, remove old ones
cmd/export.go                                — Export new resource types, remove old ones
cmd/dashboard.go                             — Collect/convert new resources, remove old ones
cmd/tutorial.go                              — Point to new tutorial format (URL stays same, data changes)

# UI
ui/src/types.ts                              — Replace CliProfileView/SystemPromptView with new View types
ui/src/api.ts                                — Replace api.cliProfiles/systemPrompts with new namespaces
ui/src/app.tsx                               — Update routes
ui/src/app-layout.tsx                        — Update NAV_ITEMS
ui/src/lib/entities.ts                       — Update Entity type, styles, icons, segments
ui/src/pages/members.tsx                     — Update columns for new member fields
ui/src/pages/member-detail.tsx               — Update detail rows for new resources
ui/src/components/team-structure/member-node.tsx — Update to show new resource refs
ui/src/components/team-structure/team-layout.ts  — Update member data shape
ui/src/hooks/use-team-structure.ts           — Update member data shape

# Tutorials
tutorials/todo-team/index.json               — New file list
tutorials/todo-team/*.json                   — Replace cli_profile/system_prompt envelopes with new types
```

---

## Task 1: New Domain Entities (ClaudeMd, Skill, Settings, ClaudeJson)

**Files:**
- Create: `internal/domain/resource/claudemd.go`
- Create: `internal/domain/resource/claudemd_test.go`
- Create: `internal/domain/resource/skill.go`
- Create: `internal/domain/resource/skill_test.go`
- Create: `internal/domain/resource/settings.go`
- Create: `internal/domain/resource/settings_test.go`
- Create: `internal/domain/resource/claudejson.go`
- Create: `internal/domain/resource/claudejson_test.go`

- [ ] **Step 1: Write failing tests for ClaudeMd**

```go
// internal/domain/resource/claudemd_test.go
package resource

import "testing"

func TestNewClaudeMd(t *testing.T) {
	md, err := NewClaudeMd("my-rules", "# Project Rules\n\nAlways use TDD.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "my-rules" {
		t.Errorf("name = %q, want %q", md.Name, "my-rules")
	}
	if md.Content != "# Project Rules\n\nAlways use TDD." {
		t.Errorf("content mismatch")
	}
	if md.ID == "" {
		t.Error("ID should be set")
	}
}

func TestNewClaudeMd_EmptyName(t *testing.T) {
	_, err := NewClaudeMd("", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeMd_EmptyContent(t *testing.T) {
	_, err := NewClaudeMd("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestClaudeMd_Update(t *testing.T) {
	md, _ := NewClaudeMd("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := md.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "new" || md.Content != "new content" {
		t.Error("update did not apply")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/resource/ -run TestNewClaudeMd -v`
Expected: FAIL — `NewClaudeMd` not defined

- [ ] **Step 3: Implement ClaudeMd**

```go
// internal/domain/resource/claudemd.go
package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeMd is a CLAUDE.md file that gets written to {workspace}/project/CLAUDE.md.
// Maps 1:1 to the Claude Code project-level CLAUDE.md.
type ClaudeMd struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeMd(name, content string) (*ClaudeMd, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude md name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude md content must not be empty")
	}

	now := time.Now()
	return &ClaudeMd{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c *ClaudeMd) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude md name must not be empty")
		}
		c.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude md content must not be empty")
		}
		c.Content = trimmed
	}
	c.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/resource/ -run TestNewClaudeMd -v`
Expected: PASS

- [ ] **Step 5: Write failing tests for Skill**

```go
// internal/domain/resource/skill_test.go
package resource

import "testing"

func TestNewSkill(t *testing.T) {
	s, err := NewSkill("code-review", "Review code for quality issues")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "code-review" {
		t.Errorf("name = %q, want %q", s.Name, "code-review")
	}
	if s.Content != "Review code for quality issues" {
		t.Errorf("content mismatch")
	}
	if s.ID == "" {
		t.Error("ID should be set")
	}
}

func TestNewSkill_EmptyName(t *testing.T) {
	_, err := NewSkill("", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewSkill_EmptyContent(t *testing.T) {
	_, err := NewSkill("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestNewSkill_InvalidName(t *testing.T) {
	for _, bad := range []string{"Has Spaces", "UPPERCASE", "special!char", "under_score", ".dotfile"} {
		_, err := NewSkill(bad, "content")
		if err == nil {
			t.Errorf("expected error for invalid name %q", bad)
		}
	}
}

func TestNewSkill_ValidNames(t *testing.T) {
	for _, good := range []string{"code-review", "tdd", "my-skill-123"} {
		_, err := NewSkill(good, "content")
		if err != nil {
			t.Errorf("unexpected error for valid name %q: %v", good, err)
		}
	}
}

func TestSkill_Update(t *testing.T) {
	s, _ := NewSkill("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := s.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "new" || s.Content != "new content" {
		t.Error("update did not apply")
	}
}
```

- [ ] **Step 6: Implement Skill**

```go
// internal/domain/resource/skill.go
package resource

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Skill is a Claude Code skill that gets written to {workspace}/.claude/skills/{name}/SKILL.md.
// Maps 1:1 to the Claude Code skill system.
// Name is used as the folder name, so it must be a valid directory name
// (lowercase, hyphens, no spaces or special chars).
type Skill struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// validSkillName checks that the name is safe as a directory name.
// Allows lowercase letters, digits, and hyphens only.
var validSkillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func NewSkill(name, content string) (*Skill, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("skill name must not be empty")
	}
	if !validSkillName.MatchString(name) {
		return nil, errors.New("skill name must be lowercase with hyphens only (e.g. code-review)")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("skill content must not be empty")
	}

	now := time.Now()
	return &Skill{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Skill) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("skill name must not be empty")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("skill content must not be empty")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 7: Run Skill tests**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/resource/ -run TestNewSkill -v`
Expected: PASS

- [ ] **Step 8: Write failing tests for Settings**

```go
// internal/domain/resource/settings_test.go
package resource

import "testing"

func TestNewSettings(t *testing.T) {
	s, err := NewSettings("skip-permissions", `{"skipDangerousModePermissionPrompt":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "skip-permissions" {
		t.Errorf("name = %q, want %q", s.Name, "skip-permissions")
	}
	if s.Content != `{"skipDangerousModePermissionPrompt":true}` {
		t.Errorf("content mismatch")
	}
}

func TestNewSettings_EmptyName(t *testing.T) {
	_, err := NewSettings("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewSettings_EmptyContent(t *testing.T) {
	_, err := NewSettings("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestSettings_Update(t *testing.T) {
	s, _ := NewSettings("old", `{"old":true}`)
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

- [ ] **Step 9: Implement Settings**

```go
// internal/domain/resource/settings.go
package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Settings is a settings.json file that gets written to CLAUDE_CONFIG_DIR/settings.json.
// Maps 1:1 to the Claude Code settings.json.
type Settings struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSettings(name, content string) (*Settings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("settings name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("settings content must not be empty")
	}

	now := time.Now()
	return &Settings{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Settings) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("settings name must not be empty")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("settings content must not be empty")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 10: Write failing tests for ClaudeJson**

```go
// internal/domain/resource/claudejson_test.go
package resource

import "testing"

func TestNewClaudeJson(t *testing.T) {
	cj, err := NewClaudeJson("onboarding-done", `{"hasCompletedOnboarding":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cj.Name != "onboarding-done" {
		t.Errorf("name = %q, want %q", cj.Name, "onboarding-done")
	}
	if cj.Content != `{"hasCompletedOnboarding":true}` {
		t.Errorf("content mismatch")
	}
}

func TestNewClaudeJson_EmptyName(t *testing.T) {
	_, err := NewClaudeJson("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeJson_EmptyContent(t *testing.T) {
	_, err := NewClaudeJson("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestClaudeJson_Update(t *testing.T) {
	cj, _ := NewClaudeJson("old", `{"old":true}`)
	newName := "new"
	newContent := `{"new":true}`
	if err := cj.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cj.Name != "new" || cj.Content != `{"new":true}` {
		t.Error("update did not apply")
	}
}
```

- [ ] **Step 11: Implement ClaudeJson**

```go
// internal/domain/resource/claudejson.go
package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeJson is a .claude.json file that gets written to CLAUDE_CONFIG_DIR/.claude.json.
// Maps 1:1 to the Claude Code .claude.json project config.
type ClaudeJson struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeJson(name, content string) (*ClaudeJson, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude json name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude json content must not be empty")
	}

	now := time.Now()
	return &ClaudeJson{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c *ClaudeJson) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude json name must not be empty")
		}
		c.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude json content must not be empty")
		}
		c.Content = trimmed
	}
	c.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 12: Run all new resource tests**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/resource/ -run "TestNew(ClaudeMd|Skill|Settings|ClaudeJson)" -v`
Expected: ALL PASS

- [ ] **Step 13: Delete old resource files**

```bash
rm internal/domain/resource/cliprofile.go
rm internal/domain/resource/cliprofile_test.go
rm internal/domain/resource/systemprompt.go
rm internal/domain/resource/systemprompt_test.go
```

- [ ] **Step 14: Commit**

```bash
git add internal/domain/resource/
git commit -m "feat: replace CliProfile/SystemPrompt with ClaudeMd, Skill, Settings, ClaudeJson

New resources map 1:1 to Claude Code features:
- ClaudeMd → project/CLAUDE.md
- Skill → .claude/skills/{name}/SKILL.md
- Settings → CLAUDE_CONFIG_DIR/settings.json
- ClaudeJson → CLAUDE_CONFIG_DIR/.claude.json

Remove CliProfile (bundle) and SystemPrompt (opaque abstraction).

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Update Member Domain Entity

**Files:**
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/member_test.go`

- [ ] **Step 1: Write failing test for new Member fields**

```go
// Replace content of internal/domain/member_test.go
package domain

import "testing"

func TestNewMember(t *testing.T) {
	m, err := NewMember("coder", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
		"claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
		[]string{"env-1"}, "repo-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "coder" {
		t.Errorf("name = %q, want %q", m.Name, "coder")
	}
	if m.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want %q", m.Model, "claude-sonnet-4-6")
	}
	if len(m.Args) != 1 || m.Args[0] != "--dangerously-skip-permissions" {
		t.Errorf("args = %v, want [--dangerously-skip-permissions]", m.Args)
	}
	if m.ClaudeMdID != "claude-md-1" {
		t.Errorf("claude_md_id = %q, want %q", m.ClaudeMdID, "claude-md-1")
	}
	if len(m.SkillIDs) != 1 || m.SkillIDs[0] != "skill-1" {
		t.Errorf("skill_ids = %v, want [skill-1]", m.SkillIDs)
	}
	if m.SettingsID != "settings-1" {
		t.Errorf("settings_id = %q, want %q", m.SettingsID, "settings-1")
	}
	if m.ClaudeJsonID != "claude-json-1" {
		t.Errorf("claude_json_id = %q, want %q", m.ClaudeJsonID, "claude-json-1")
	}
	if m.GitRepoID != "repo-1" {
		t.Errorf("git_repo_id = %q, want %q", m.GitRepoID, "repo-1")
	}
}

func TestNewMember_EmptyName(t *testing.T) {
	_, err := NewMember("", "model", nil, "", nil, "", "", nil, "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMember_EmptyModel(t *testing.T) {
	_, err := NewMember("name", "", nil, "", nil, "", "", nil, "")
	if err == nil {
		t.Error("expected error for empty model")
	}
}

func TestMember_NilSlicesDefault(t *testing.T) {
	m, err := NewMember("coder", "claude-sonnet-4-6", nil, "", nil, "", "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Args == nil {
		t.Error("Args should be empty slice, not nil")
	}
	if m.SkillIDs == nil {
		t.Error("SkillIDs should be empty slice, not nil")
	}
	if m.EnvIDs == nil {
		t.Error("EnvIDs should be empty slice, not nil")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/ -run TestNewMember -v`
Expected: FAIL — signature mismatch

- [ ] **Step 3: Implement new Member**

```go
// internal/domain/member.go — full replacement
package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain/resource"
)

type Member struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Model        string    `json:"model"`
	Args         []string  `json:"args"`
	ClaudeMdID   string    `json:"claude_md_id"`   // empty string = not set (nullable FK)
	SkillIDs     []string  `json:"skill_ids"`
	SettingsID   string    `json:"settings_id"`     // empty string = not set (nullable FK)
	ClaudeJsonID string    `json:"claude_json_id"`  // empty string = not set (nullable FK)
	EnvIDs       []string  `json:"env_ids"`
	GitRepoID    string    `json:"git_repo_id"`     // empty string = not set (nullable FK)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewMember(name, model string, args []string,
	claudeMdID string, skillIDs []string,
	settingsID, claudeJsonID string,
	envIDs []string, gitRepoID string) (*Member, error) {

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("member name must not be empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("member model must not be empty")
	}
	if args == nil {
		args = []string{}
	}
	if skillIDs == nil {
		skillIDs = []string{}
	}
	if envIDs == nil {
		envIDs = []string{}
	}

	now := time.Now()
	return &Member{
		ID:           uuid.NewString(),
		Name:         name,
		Model:        model,
		Args:         args,
		ClaudeMdID:   claudeMdID,
		SkillIDs:     skillIDs,
		SettingsID:   settingsID,
		ClaudeJsonID: claudeJsonID,
		EnvIDs:       envIDs,
		GitRepoID:    gitRepoID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (m *Member) Update(name, model *string, args *[]string,
	claudeMdID *string, skillIDs *[]string,
	settingsID, claudeJsonID *string,
	envIDs *[]string, gitRepoID *string) error {

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("member name must not be empty")
		}
		m.Name = trimmed
	}
	if model != nil {
		trimmed := strings.TrimSpace(*model)
		if trimmed == "" {
			return errors.New("member model must not be empty")
		}
		m.Model = trimmed
	}
	if args != nil {
		m.Args = *args
	}
	if claudeMdID != nil {
		m.ClaudeMdID = *claudeMdID
	}
	if skillIDs != nil {
		m.SkillIDs = *skillIDs
	}
	if settingsID != nil {
		m.SettingsID = *settingsID
	}
	if claudeJsonID != nil {
		m.ClaudeJsonID = *claudeJsonID
	}
	if envIDs != nil {
		m.EnvIDs = *envIDs
	}
	if gitRepoID != nil {
		m.GitRepoID = *gitRepoID
	}
	m.UpdatedAt = time.Now()
	return nil
}

// ResolvedMember is a Member spec with all referenced resources loaded.
// Produced by the resolve phase; consumed by the build phase to create MemberPlan.
type ResolvedMember struct {
	TeamMemberID string
	Name         string
	Model        string
	Args         []string
	ClaudeMd     *resource.ClaudeMd
	Skills       []resource.Skill
	Settings     *resource.Settings
	ClaudeJson   *resource.ClaudeJson
	Envs         []resource.Env
	Repo         *resource.GitRepo
	Relations    MemberRelations
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/domain/ -run TestNewMember -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/member.go internal/domain/member_test.go
git commit -m "feat: restructure Member with Model, Args, and new resource IDs

Member now directly holds Model and Args (CLI flags) instead of
referencing CliProfile. Resource references updated:
- ClaudeMdID (1) + SkillIDs (N) replace SystemPromptIDs
- SettingsID + ClaudeJsonID replace CliProfileID

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Database Schema and Queries

**Files:**
- Modify: `internal/adapter/db/schema.sql`
- Create: `internal/adapter/db/queries/claude_md.sql`
- Create: `internal/adapter/db/queries/skill.sql`
- Create: `internal/adapter/db/queries/settings.sql`
- Create: `internal/adapter/db/queries/claude_json.sql`
- Modify: `internal/adapter/db/queries/member.sql`
- Delete: `internal/adapter/db/queries/cli_profile.sql`
- Delete: `internal/adapter/db/queries/system_prompt.sql`

- [ ] **Step 1: Rewrite schema.sql**

Replace cli_profiles, system_prompts, member_system_prompts with new tables. Update members table to have new columns.

```sql
-- New tables (add before members table)
CREATE TABLE IF NOT EXISTS claude_mds (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS claude_jsons (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Updated members table (replaces old one)
-- claude_md_id, settings_id, claude_json_id, git_repo_id are all nullable
-- (user may not set them). SQLite NULLs satisfy FK constraints.
CREATE TABLE IF NOT EXISTS members (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    model          TEXT NOT NULL,
    args           TEXT NOT NULL DEFAULT '[]',
    claude_md_id   TEXT REFERENCES claude_mds(id) ON DELETE RESTRICT,
    settings_id    TEXT REFERENCES settings(id) ON DELETE RESTRICT,
    claude_json_id TEXT REFERENCES claude_jsons(id) ON DELETE RESTRICT,
    git_repo_id    TEXT REFERENCES git_repos(id) ON DELETE RESTRICT,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);

-- New junction table (replaces member_system_prompts)
CREATE TABLE IF NOT EXISTS member_skills (
    member_id TEXT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    skill_id  TEXT NOT NULL REFERENCES skills(id)  ON DELETE RESTRICT,
    PRIMARY KEY (member_id, skill_id)
);

-- member_envs stays the same
-- Remove: cli_profiles, system_prompts, member_system_prompts tables
```

- [ ] **Step 2: Write new SQL query files**

Each query file follows the same CRUD pattern as existing ones (see queries/env.sql for reference). Create claude_md.sql, skill.sql, settings.sql, claude_json.sql with standard GetX/ListX/CreateX/UpdateX/DeleteX queries.

- [ ] **Step 3: Update member.sql for new columns/junctions**

Replace cli_profile_id references with model, args, claude_md_id, settings_id, claude_json_id. Replace member_system_prompts with member_skills junction.

- [ ] **Step 4: Delete old query files**

```bash
rm internal/adapter/db/queries/cli_profile.sql
rm internal/adapter/db/queries/system_prompt.sql
```

- [ ] **Step 5: Regenerate sqlc**

Run: `cd /Users/jake_kakao/jakeraft/clier/internal/adapter/db && sqlc generate`
Expected: Generated files updated in generated/ directory

- [ ] **Step 6: Commit**

```bash
git add internal/adapter/db/
git commit -m "feat: update DB schema for new resource types

Drop cli_profiles, system_prompts, member_system_prompts tables.
Add claude_mds, skills, settings, claude_jsons tables.
Update members table with model, args, new resource FKs.
Add member_skills junction table.

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Store Adapter

**Files:**
- Modify: `internal/adapter/db/store.go`

- [ ] **Step 1: Remove CliProfile and SystemPrompt store methods**

Delete: `marshalCliProfileSlices`, `CreateCliProfile`, `unmarshalCliProfile`, `GetCliProfile`, `ListCliProfiles`, `UpdateCliProfile`, `DeleteCliProfile`, `CreateSystemPrompt`, `GetSystemPrompt`, `ListSystemPrompts`, `UpdateSystemPrompt`, `DeleteSystemPrompt`.

Also remove the old migration code in `NewStore` (the `columnExists` checks for cli_profiles).

- [ ] **Step 2: Add new resource store methods**

Add CRUD methods for ClaudeMd, Skill, Settings, ClaudeJson following the same pattern as Env (simple entities with no special marshaling).

- [ ] **Step 3: Update Member store methods**

Update `CreateMember`, `GetMember`, `ListMembers`, `UpdateMember`, `DeleteMember` to use new columns (model, args, claude_md_id, settings_id, claude_json_id) and new junction table (member_skills instead of member_system_prompts).

The args field needs JSON marshal/unmarshal like the old SystemArgs/CustomArgs pattern.

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build ./...`
Expected: Build succeeds (may still have errors in other packages that depend on old types — that's expected, we fix those in later tasks)

- [ ] **Step 5: Commit**

```bash
git add internal/adapter/db/store.go
git commit -m "feat: update store for new resource types

Replace CliProfile/SystemPrompt CRUD with ClaudeMd, Skill, Settings,
ClaudeJson CRUD. Update Member store to use new columns and junctions.

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Task Build Pipeline (plan, command, config, prompt)

**Files:**
- Modify: `internal/app/task/service.go`
- Modify: `internal/app/task/plan.go`
- Modify: `internal/app/task/command.go`
- Modify: `internal/app/task/config.go`
- Modify: `internal/app/task/prompt.go`
- Modify: `internal/app/task/plan_test.go`
- Modify: `internal/app/task/command_test.go`
- Modify: `internal/app/task/prompt_test.go`
- Modify: `internal/app/task/service_test.go`

This is the critical task where the build phase must clearly show building block → execution mapping with system/user separation.

- [ ] **Step 1: Update TaskStore interface in service.go**

Replace `GetCliProfile`/`GetSystemPrompt` with `GetClaudeMd`/`GetSkill`/`GetSettings`/`GetClaudeJson`.

- [ ] **Step 2: Update resolve phase in plan.go**

`resolveMember` must load from new resource types. `ResolvedMember` already has the new shape (from Task 2).

- [ ] **Step 3: Rewrite buildMemberPlan as the transparent facade**

This is the key change. `buildMemberPlan` must clearly show:
1. Each building block and where it goes
2. System-generated vs user-defined separation

```go
func buildMemberPlan(rm *ResolvedMember, nameByID map[string]string, teamName, taskID string) domain.MemberPlan {
	memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, PlaceholderTaskID, rm.TeamMemberID)

	// === CLAUDE.md ===
	systemClaudeMd := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID) // Clier system
	var userClaudeMd string                                                        // user building block
	if rm.ClaudeMd != nil {
		userClaudeMd = rm.ClaudeMd.Content
	}

	// === settings.json ===
	var userSettings string // user building block (no system injection currently)
	if rm.Settings != nil {
		userSettings = rm.Settings.Content
	}

	// === .claude.json ===
	systemClaudeJson := buildSystemClaudeJson(memberspace) // Clier system: projects path
	var userClaudeJson string                               // user building block
	if rm.ClaudeJson != nil {
		userClaudeJson = rm.ClaudeJson.Content
	}

	// === Skills ===
	userSkills := rm.Skills // user building block (no system injection)

	// === Assemble files ===
	files := buildWorkspaceFiles(memberspace, systemClaudeMd, userClaudeMd, userSettings, systemClaudeJson, userClaudeJson, userSkills)

	// === Command: user building blocks ===
	model := rm.Model
	args := rm.Args
	userEnvs := rm.Envs

	// === Command: Clier system-generated ===
	systemEnvs := buildSystemEnvs(taskID, rm.TeamMemberID)
	authEnvs := buildAuthEnvs()
	gitEnvs := buildIdentityEnvs(teamName, rm.Name)

	// === Assemble command ===
	cmd := buildCommand(model, args, systemEnvs, authEnvs, gitEnvs, userEnvs, memberspace)

	launchPath := memberspace + "/launch.sh"
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
	}
}
```

- [ ] **Step 4: Update command.go**

`buildCommand` no longer takes `resource.CliProfile`. Takes model, args, and separate env slices directly.

`buildAgentCommand` takes model and args (single slice, no systemArgs/customArgs split).

Remove `--append-system-prompt` from the command entirely.

- [ ] **Step 5: Rewrite config.go → workspace_files.go**

Rename to `workspace_files.go`. New function `buildWorkspaceFiles` creates all file entries with consistent system/user merge pattern:
- `{memberspace}/project/CLAUDE.md` — **system:** team protocol + **user:** ClaudeMd content
- `{memberspace}/.claude/settings.json` — **system:** (none currently) + **user:** Settings content
- `{memberspace}/.claude/.claude.json` — **system:** projects path auto-inject + **user:** ClaudeJson content
- `{memberspace}/.claude/skills/{name}/SKILL.md` — **user:** one per Skill (no system injection)

For .claude.json system injection: `buildSystemClaudeJson(memberspace)` generates the `projects` key with workspace path. This is merged with user content via JSON merge (system keys + user keys). No `{{CLIER_MEMBERSPACE}}` placeholder needed — system builds the path directly.

- [ ] **Step 6: Update expand.go**

expand.go stays — placeholders are still needed because Plan is stored in DB and must not contain machine-specific paths or auth tokens. What changes:
- Placeholders are now **system-internal only** (never appear in user-provided building block content)
- System injection functions (buildSystemClaudeJson, buildSystemEnvs, etc.) emit placeholders that expand resolves
- User content is pure — no `{{CLIER_*}}` in ClaudeMd, Settings, ClaudeJson, Skill, Env values
- Update expand_test.go to verify user content passes through unchanged while system-injected placeholders are expanded

- [ ] **Step 7: Update prompt.go**

Remove `joinPrompts` (no longer merging SystemPrompts). Keep `buildClierPrompt` as-is (it generates the team protocol).

- [ ] **Step 8: Update all test files**

Update plan_test.go, command_test.go, prompt_test.go, config_test.go → workspace_files_test.go, expand_test.go, service_test.go to use new types and function signatures. Each test must use the new ResolvedMember shape.

- [ ] **Step 9: Verify tests pass**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/app/task/ -v`
Expected: ALL PASS

- [ ] **Step 10: Commit**

```bash
git add internal/app/task/
git commit -m "feat: transparent build pipeline with building block facade

buildMemberPlan now clearly shows:
- Each building block and its Claude Code destination
- System-generated (team protocol, infra envs) vs user-defined separation
- No --append-system-prompt; CLAUDE.md is a real file

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Team Import/Export Service

**Files:**
- Modify: `internal/app/team/service.go`
- Modify: `internal/app/team/service_test.go`

- [ ] **Step 1: Update Store interface**

Replace CliProfile/SystemPrompt methods with new resource methods.

- [ ] **Step 2: Update ImportTeam logic**

No functional change to import logic — it still creates referenced resources if missing. Just uses new types.

- [ ] **Step 3: Update tests**

- [ ] **Step 4: Verify tests pass**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./internal/app/team/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/team/
git commit -m "refactor: update team service for new resource types

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: CLI Commands

**Files:**
- Create: `cmd/claudemd.go`
- Create: `cmd/skill.go`
- Create: `cmd/settings.go` (rename: avoid conflict with existing settings)
- Create: `cmd/claudejson.go`
- Modify: `cmd/member.go`
- Modify: `cmd/import.go`
- Modify: `cmd/export.go`
- Modify: `cmd/dashboard.go`
- Delete: `cmd/profile.go`
- Delete: `cmd/prompt.go`

- [ ] **Step 1: Create new CLI commands**

Each new command follows the exact same pattern as `cmd/env.go` (existing). CRUD: create, list, update, delete subcommands.

New commands:
- `clier claude-md create/list/update/delete`
- `clier skill create/list/update/delete`
- `clier settings create/list/update/delete` (CLI subcommand name: `claude-settings` to avoid collision with internal settings)
- `clier claude-json create/list/update/delete`

- [ ] **Step 2: Update member.go**

Replace `--profile` flag with `--model`, `--args` flags. Replace `--prompt` with `--claude-md`, `--skill` flags. Add `--settings`, `--claude-json` flags.

- [ ] **Step 3: Update import.go**

Replace `case "cli_profile"` and `case "system_prompt"` with `case "claude_md"`, `case "skill"`, `case "settings"`, `case "claude_json"`.

- [ ] **Step 4: Update export.go**

Replace probe list with new resource types.

- [ ] **Step 5: Update dashboard.go**

Replace collectDashboardData to load new resources. Replace view structs and converters. The `dashboardData` struct must match the new UI types.

- [ ] **Step 6: Delete old command files**

```bash
rm cmd/profile.go
rm cmd/prompt.go
```

- [ ] **Step 7: Verify full build**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build ./...`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add cmd/
git commit -m "feat: new CLI commands for ClaudeMd, Skill, Settings, ClaudeJson

Replace profile/prompt commands with claude-md, skill, claude-settings,
claude-json commands. Update member, import, export, dashboard.

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: UI — Types, API, Entities

**Files:**
- Modify: `ui/src/types.ts`
- Modify: `ui/src/api.ts`
- Modify: `ui/src/lib/entities.ts`

- [ ] **Step 1: Update types.ts**

Remove `CliProfileView`, `SystemPromptView`. Add `ClaudeMdView`, `SkillView`, `SettingsView`, `ClaudeJsonView`. Update `MemberView` and `DashboardData`.

```typescript
export interface DashboardData {
  teams: TeamView[];
  members: MemberView[];
  claudeMds: ClaudeMdView[];
  skills: SkillView[];
  settings: SettingsView[];
  claudeJsons: ClaudeJsonView[];
  gitRepos: GitRepoView[];
  envs: EnvView[];
  tasks: TaskView[];
}

export interface MemberView {
  id: string;
  name: string;
  model: string;
  args: string[];
  claudeMdId: string | null;
  skillIds: string[];
  settingsId: string | null;
  claudeJsonId: string | null;
  envIds: string[];
  gitRepoId: string | null;
  claudeMdName: string | null;
  skillNames: string[];
  settingsName: string | null;
  claudeJsonName: string | null;
  envNames: string[];
  gitRepoName: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ClaudeMdView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface SkillView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface SettingsView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClaudeJsonView {
  id: string;
  name: string;
  content: string;
  createdAt: string;
  updatedAt: string;
}
```

- [ ] **Step 2: Update api.ts**

Replace `api.cliProfiles` and `api.systemPrompts` with `api.claudeMds`, `api.skills`, `api.settings`, `api.claudeJsons`. Update `getStructure` to return new member fields.

- [ ] **Step 3: Update entities.ts**

```typescript
type Entity = "team" | "task" | "member" | "claude-md" | "skill" | "settings" | "claude-json" | "git-repo" | "env";
```

Update `ENTITY_STYLE`, `ENTITY_ICON`, `SEGMENT_TO_ENTITY`, and `entityFromPath` regex.

Use existing entity color tokens:
- `claude-md` → `entity-instruction` (teal, was system-prompt's color)
- `skill` → new or reuse `entity-instruction`
- `settings` → `entity-model` (yellow-orange, was cli-profile's color)
- `claude-json` → `entity-model`

Choose icons from lucide-react:
- `claude-md` → `FileText`
- `skill` → `BookOpen` (reuse from old system-prompt)
- `settings` → `Settings` (lucide icon)
- `claude-json` → `FileJson`

- [ ] **Step 4: Commit**

```bash
git add ui/src/types.ts ui/src/api.ts ui/src/lib/entities.ts
git commit -m "feat: update UI types, API, entities for new resources

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: UI — Pages and Routes

**Files:**
- Create: `ui/src/pages/claude-mds.tsx`
- Create: `ui/src/pages/claude-md-detail.tsx`
- Create: `ui/src/pages/skills.tsx`
- Create: `ui/src/pages/skill-detail.tsx`
- Create: `ui/src/pages/settings-list.tsx`
- Create: `ui/src/pages/settings-detail.tsx`
- Create: `ui/src/pages/claude-jsons.tsx`
- Create: `ui/src/pages/claude-json-detail.tsx`
- Modify: `ui/src/app.tsx`
- Modify: `ui/src/app-layout.tsx`
- Modify: `ui/src/pages/members.tsx`
- Modify: `ui/src/pages/member-detail.tsx`
- Delete: `ui/src/pages/cli-profiles.tsx`
- Delete: `ui/src/pages/cli-profile-detail.tsx`
- Delete: `ui/src/pages/system-prompts.tsx`
- Delete: `ui/src/pages/system-prompt-detail.tsx`

- [ ] **Step 1: Create list pages**

Follow exact same pattern as existing `system-prompts.tsx` (simplest example). Each list page uses `EntityListPage` with columns.

Example for claude-mds.tsx:
```tsx
import { api } from "@/api";
import type { ClaudeMdView } from "@/api";
import { EntityListPage } from "@/components/entity-list-page";
import type { EntityTableColumn } from "@/components/entity-table";

const columns: EntityTableColumn<ClaudeMdView>[] = [
  { header: "Content", cell: (c) => c.content, flex: 2 },
];

export function ClaudeMds() {
  return (
    <EntityListPage<ClaudeMdView>
      entityType="claude-md"
      apiList={api.claudeMds.list}
      columns={columns}
      empty={{ title: "No CLAUDE.md files yet", description: "CLAUDE.md project instructions" }}
      routeBase="/claude-mds"
    />
  );
}
```

Repeat same pattern for skills.tsx, settings-list.tsx, claude-jsons.tsx.

- [ ] **Step 2: Create detail pages**

Follow exact same pattern as existing `system-prompt-detail.tsx`. Each detail page uses `useDetailPage` + `DetailLayout` + `SectionCard` + `OverviewTable`.

- [ ] **Step 3: Update app.tsx routes**

Remove old routes, add new ones:
```tsx
<Route path="/claude-mds" element={<ClaudeMds />} />
<Route path="/claude-mds/:id" element={<Keyed Component={ClaudeMdDetail} />} />
<Route path="/skills" element={<Skills />} />
<Route path="/skills/:id" element={<Keyed Component={SkillDetail} />} />
<Route path="/claude-settings" element={<SettingsList />} />
<Route path="/claude-settings/:id" element={<Keyed Component={SettingsDetail} />} />
<Route path="/claude-jsons" element={<ClaudeJsons />} />
<Route path="/claude-jsons/:id" element={<Keyed Component={ClaudeJsonDetail} />} />
```

- [ ] **Step 4: Update app-layout.tsx NAV_ITEMS**

```tsx
const NAV_ITEMS = [
  { to: "/tasks", label: "Task", icon: Play },
  { to: "/teams", label: "Team", icon: Users },
  { to: "/members", label: "Member", icon: User },
  { to: "/claude-mds", label: "CLAUDE.md", icon: FileText },
  { to: "/skills", label: "Skill", icon: BookOpen },
  { to: "/claude-settings", label: "Settings", icon: Settings2 },
  { to: "/claude-jsons", label: ".claude.json", icon: FileJson },
  { to: "/git-repos", label: "Repo", icon: FolderGit2 },
  { to: "/envs", label: "Env", icon: KeyRound },
];
```

- [ ] **Step 5: Update member pages**

Update `members.tsx` columns to show model, args instead of cliProfileName.
Update `member-detail.tsx` to show new resource badges (ClaudeMd, Skills, Settings, ClaudeJson).

- [ ] **Step 6: Delete old pages**

```bash
rm ui/src/pages/cli-profiles.tsx
rm ui/src/pages/cli-profile-detail.tsx
rm ui/src/pages/system-prompts.tsx
rm ui/src/pages/system-prompt-detail.tsx
```

- [ ] **Step 7: Verify UI build**

Run: `cd /Users/jake_kakao/jakeraft/clier/ui && npm run build`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add ui/
git commit -m "feat: update UI pages and routes for new resource types

Replace CLI Profile and System Prompt pages with ClaudeMd, Skill,
Settings, ClaudeJson pages. Follow existing component patterns.
Update member pages to show new resource fields.

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: UI — Team Structure Components

**Files:**
- Modify: `ui/src/components/team-structure/member-node.tsx`
- Modify: `ui/src/components/team-structure/team-layout.ts`
- Modify: `ui/src/hooks/use-team-structure.ts`

- [ ] **Step 1: Update use-team-structure.ts**

Replace `cliProfileId`/`systemPromptIds` with `model`/`skillIds` in the structure data shape.

- [ ] **Step 2: Update team-layout.ts**

Update the member data interface to match.

- [ ] **Step 3: Update member-node.tsx**

Show model name and skill count instead of CLI profile name and system prompt count.

- [ ] **Step 4: Verify UI build**

Run: `cd /Users/jake_kakao/jakeraft/clier/ui && npm run build`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add ui/src/components/team-structure/ ui/src/hooks/
git commit -m "feat: update team structure visualization for new resources

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 11: Tutorials

**Files:**
- Modify: `tutorials/todo-team/index.json`
- Delete + Create: tutorial envelope files

- [ ] **Step 1: Create new tutorial files**

Replace `claude-sonnet.json` (cli_profile) with separate files:
- `settings-default.json` (type: settings)
- `claude-json-default.json` (type: claude_json)

Replace `prompt-*.json` (system_prompt) with:
- `claude-md-tech-lead.json` (type: claude_md) — or skill files if appropriate
- `claude-md-coder.json` (type: claude_md)
- `claude-md-reviewer.json` (type: claude_md)

Update `member-*.json` to use new fields (model, args, claude_md_id, settings_id, claude_json_id).

- [ ] **Step 2: Update index.json**

New file ordering reflecting dependency order.

- [ ] **Step 3: Delete old tutorial files**

```bash
rm tutorials/todo-team/claude-sonnet.json
rm tutorials/todo-team/claude-haiku.json
rm tutorials/todo-team/prompt-tech-lead.json
rm tutorials/todo-team/prompt-coder.json
rm tutorials/todo-team/prompt-reviewer.json
```

- [ ] **Step 4: Commit**

```bash
git add tutorials/
git commit -m "feat: update tutorial for new resource types

Tutorial now uses ClaudeMd, Settings, ClaudeJson instead of
CliProfile and SystemPrompt. Serves as the default config guide.

🤖 Generated with [Claude Code](https://claude.ai/code)
Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 12: Full Integration Test

- [ ] **Step 1: Delete old database**

```bash
rm -f ~/.clier/data.db
```

- [ ] **Step 2: Run all Go tests**

Run: `cd /Users/jake_kakao/jakeraft/clier && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: Build binary**

Run: `cd /Users/jake_kakao/jakeraft/clier && go build -o clier .`
Expected: Build succeeds

- [ ] **Step 4: Run tutorial smoke test**

Run: `./clier tutorial start` (then immediately `./clier task stop <id>`)
Expected: Tutorial imports, task starts, agents launch in tmux

- [ ] **Step 5: Run dashboard smoke test**

Run: `./clier dashboard`
Expected: Dashboard opens in browser with new resource types displayed

- [ ] **Step 6: Commit any fixes**

Fix any issues found during integration testing.
