# Server Domain Entities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** clier-server에 Workspace 엔티티(ClaudeMd, Skill, ClaudeSettings, Member, Team)와 Run 엔티티를 추가한다.

**Architecture:** 기존 GitRepo/AgentDotMd 패턴을 따라 각 엔티티를 migration → domain → store → service → handler → router 순서로 추가. Member는 빌딩블록을 FK로 참조하고, Team은 TeamMember/TeamRelation 하위 테이블을 가진다. Run은 개인 전용(Visibility/Fork/Version 없음).

**Tech Stack:** Go 1.25, Echo v4, PostgreSQL, Squirrel, sqlx

**Working Directory:** `/Users/jake_kakao/jakeraft/clier-server`

---

### Task 1: AgentDotMd → ClaudeMd 리네임

**Files:**
- Create: `migrations/000008_rename_agent_dot_mds_to_claude_mds.up.sql`
- Create: `migrations/000008_rename_agent_dot_mds_to_claude_mds.down.sql`
- Create: `internal/domain/claude_md.go`
- Create: `internal/domain/claude_md_version.go`
- Create: `internal/db/claude_md_store.go`
- Create: `internal/services/claudemd/claudemd.go`
- Create: `internal/handler/claude_md.go`
- Modify: `internal/router/router.go`
- Delete: `internal/domain/agent_dot_md.go`
- Delete: `internal/domain/agent_dot_md_version.go`
- Delete: `internal/db/agent_dot_md_store.go`
- Delete: `internal/services/agentdotmd/agentdotmd.go`
- Delete: `internal/handler/agent_dot_md.go`

- [ ] **Step 1: Migration 작성**

```sql
-- 000008_rename_agent_dot_mds_to_claude_mds.up.sql
ALTER TABLE agent_dot_mds RENAME TO claude_mds;
ALTER TABLE agent_dot_md_versions RENAME TO claude_md_versions;
ALTER TABLE claude_md_versions RENAME COLUMN agent_dot_md_id TO claude_md_id;
ALTER INDEX idx_agent_dot_mds_owner_id RENAME TO idx_claude_mds_owner_id;
ALTER INDEX idx_agent_dot_md_versions_agent_dot_md_id RENAME TO idx_claude_md_versions_claude_md_id;
```

```sql
-- 000008_rename_agent_dot_mds_to_claude_mds.down.sql
ALTER TABLE claude_mds RENAME TO agent_dot_mds;
ALTER TABLE claude_md_versions RENAME TO agent_dot_md_versions;
ALTER TABLE agent_dot_md_versions RENAME COLUMN claude_md_id TO agent_dot_md_id;
ALTER INDEX idx_claude_mds_owner_id RENAME TO idx_agent_dot_mds_owner_id;
ALTER INDEX idx_claude_md_versions_claude_md_id RENAME TO idx_agent_dot_md_versions_agent_dot_md_id;
```

- [ ] **Step 2: Domain 구조체 리네임**

`internal/domain/claude_md.go` — 기존 `agent_dot_md.go`를 복사하고 모든 `AgentDotMd` → `ClaudeMd`, `agent_dot_md` → `claude_md` 치환:

```go
package domain

import "time"

type ClaudeMd struct {
	ID            int64      `db:"id" json:"id"`
	OwnerID       int64      `db:"owner_id" json:"owner_id"`
	Name          string     `db:"name" json:"name"`
	LowerName     string     `db:"lower_name" json:"-"`
	Content       string     `db:"content" json:"content"`
	Visibility    Visibility `db:"visibility" json:"visibility"`
	IsFork        bool       `db:"is_fork" json:"is_fork"`
	ForkID        *int64     `db:"fork_id" json:"fork_id,omitempty"`
	ForkCount     int        `db:"fork_count" json:"fork_count"`
	LatestVersion *int       `db:"latest_version" json:"latest_version,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type ClaudeMdView struct {
	ClaudeMd
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
	ForkName       *string `db:"fork_name" json:"fork_name,omitempty"`
	ForkOwnerLogin *string `db:"fork_owner_login" json:"fork_owner_login,omitempty"`
}
```

`internal/domain/claude_md_version.go`:

```go
package domain

import (
	"encoding/json"
	"time"
)

type ClaudeMdVersion struct {
	ID         int64           `db:"id" json:"id"`
	ClaudeMdID int64           `db:"claude_md_id" json:"claude_md_id"`
	Version    int             `db:"version" json:"version"`
	Content    json.RawMessage `db:"content" json:"content"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
}

type ClaudeMdVersionView struct {
	ClaudeMdVersion
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
}

type ClaudeMdSnapshot struct {
	Content string `json:"content"`
}
```

- [ ] **Step 3: Store 리네임**

기존 `internal/db/agent_dot_md_store.go`를 복사하여 `internal/db/claude_md_store.go` 생성. 모든 `AgentDotMd` → `ClaudeMd`, `agent_dot_md` → `claude_md`, 테이블명 `agent_dot_mds` → `claude_mds`, `agent_dot_md_versions` → `claude_md_versions` 치환.

- [ ] **Step 4: Service 리네임**

기존 `internal/services/agentdotmd/` 디렉토리를 `internal/services/claudemd/`로 복사. 패키지명 `agentdotmd` → `claudemd`, 모든 `AgentDotMd` → `ClaudeMd` 치환.

- [ ] **Step 5: Handler 리네임**

기존 `internal/handler/agent_dot_md.go`를 복사하여 `internal/handler/claude_md.go` 생성. 모든 `AgentDotMd` → `ClaudeMd` 치환.

- [ ] **Step 6: Router 업데이트**

`internal/router/router.go`에서 `agent-dot-mds` → `claude-mds` 경로 변경:

```go
// Before
pub.GET("/agent-dot-mds", agentDotMdHandler.ListPublic)
pub.GET("/orgs/:owner/agent-dot-mds", agentDotMdHandler.List)
// ...

// After
pub.GET("/claude-mds", claudeMdHandler.ListPublic)
pub.GET("/orgs/:owner/claude-mds", claudeMdHandler.List)
// ...
```

- [ ] **Step 7: 이전 파일 삭제**

```bash
rm internal/domain/agent_dot_md.go internal/domain/agent_dot_md_version.go
rm internal/db/agent_dot_md_store.go
rm -rf internal/services/agentdotmd/
rm internal/handler/agent_dot_md.go
```

- [ ] **Step 8: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 9: 커밋**

```bash
git add -A && git commit -m "refactor: AgentDotMd → ClaudeMd 리네임

migration, domain, store, service, handler, router 전체 리네임.
API 경로: /agent-dot-mds → /claude-mds

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Skill 엔티티 추가

**Files:**
- Create: `migrations/000009_create_skills.up.sql`
- Create: `migrations/000009_create_skills.down.sql`
- Create: `internal/domain/skill.go`
- Create: `internal/domain/skill_version.go`
- Create: `internal/db/skill_store.go`
- Create: `internal/services/skill/skill.go`
- Create: `internal/handler/skill.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Migration 작성**

```sql
-- 000009_create_skills.up.sql
CREATE TABLE skills (
    id             BIGSERIAL PRIMARY KEY,
    owner_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    lower_name     TEXT NOT NULL,
    content        TEXT NOT NULL,
    visibility     SMALLINT NOT NULL DEFAULT 0,
    is_fork        BOOLEAN NOT NULL DEFAULT false,
    fork_id        BIGINT REFERENCES skills(id) ON DELETE SET NULL,
    fork_count     INT NOT NULL DEFAULT 0,
    latest_version INT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, lower_name)
);

CREATE INDEX idx_skills_owner_id ON skills(owner_id);

CREATE TABLE skill_versions (
    id         BIGSERIAL PRIMARY KEY,
    skill_id   BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version    INT NOT NULL,
    content    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(skill_id, version)
);

CREATE INDEX idx_skill_versions_skill_id ON skill_versions(skill_id);
```

```sql
-- 000009_create_skills.down.sql
DROP TABLE IF EXISTS skill_versions;
DROP TABLE IF EXISTS skills;
```

- [ ] **Step 2: Domain 구조체**

`internal/domain/skill.go`:

```go
package domain

import "time"

type Skill struct {
	ID            int64      `db:"id" json:"id"`
	OwnerID       int64      `db:"owner_id" json:"owner_id"`
	Name          string     `db:"name" json:"name"`
	LowerName     string     `db:"lower_name" json:"-"`
	Content       string     `db:"content" json:"content"`
	Visibility    Visibility `db:"visibility" json:"visibility"`
	IsFork        bool       `db:"is_fork" json:"is_fork"`
	ForkID        *int64     `db:"fork_id" json:"fork_id,omitempty"`
	ForkCount     int        `db:"fork_count" json:"fork_count"`
	LatestVersion *int       `db:"latest_version" json:"latest_version,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type SkillView struct {
	Skill
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
	ForkName       *string `db:"fork_name" json:"fork_name,omitempty"`
	ForkOwnerLogin *string `db:"fork_owner_login" json:"fork_owner_login,omitempty"`
}
```

`internal/domain/skill_version.go`:

```go
package domain

import (
	"encoding/json"
	"time"
)

type SkillVersion struct {
	ID        int64           `db:"id" json:"id"`
	SkillID   int64           `db:"skill_id" json:"skill_id"`
	Version   int             `db:"version" json:"version"`
	Content   json.RawMessage `db:"content" json:"content"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type SkillVersionView struct {
	SkillVersion
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
}

type SkillSnapshot struct {
	Content string `json:"content"`
}
```

- [ ] **Step 3: Store — ClaudeMd store를 복제하여 Skill용으로 변환**

`internal/db/skill_store.go` — `claude_md_store.go`를 복사하고:
- `ClaudeMd` → `Skill`, `claude_md` → `skill`
- 테이블명: `claude_mds` → `skills`, `claude_md_versions` → `skill_versions`
- FK 컬럼: `claude_md_id` → `skill_id`

전체 구조는 동일: Store struct, BeginTx, Create, Update, Delete, IncrementForkCount, UpdateLatestVersion, View queries, Version queries, Tx wrapper.

- [ ] **Step 4: Service — ClaudeMd service를 복제하여 Skill용으로 변환**

`internal/services/skill/skill.go` — `claudemd/claudemd.go`를 복사하고:
- 패키지명: `claudemd` → `skill`
- 모든 `ClaudeMd` → `Skill`
- Snapshot: `ClaudeMdSnapshot{Content: content}` → `SkillSnapshot{Content: content}`

- [ ] **Step 5: Handler — ClaudeMd handler를 복제하여 Skill용으로 변환**

`internal/handler/skill.go` — `claude_md.go`를 복사하고:
- 모든 `ClaudeMd` → `Skill`
- Service 타입 변경

- [ ] **Step 6: Router에 Skill 경로 추가**

```go
// Public (OptionalAuth)
pub.GET("/skills", skillHandler.ListPublic)
pub.GET("/orgs/:owner/skills", skillHandler.List)
pub.GET("/orgs/:owner/skills/:name", skillHandler.Get)
pub.GET("/orgs/:owner/skills/:name/versions", skillHandler.ListVersions)
pub.GET("/orgs/:owner/skills/:name/versions/:version", skillHandler.GetVersion)

// Private (RequireAuth)
priv.POST("/orgs/:owner/skills", skillHandler.Create)
priv.PUT("/orgs/:owner/skills/:name", skillHandler.Update)
priv.DELETE("/orgs/:owner/skills/:name", skillHandler.Delete)
priv.POST("/orgs/:owner/skills/:name/fork", skillHandler.Fork)
```

- [ ] **Step 7: main.go에 Skill 의존성 주입**

`cmd/server/main.go`에서 SkillStore, SkillService, SkillHandler 생성 및 router에 전달.

- [ ] **Step 8: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: Skill 엔티티 추가

migration, domain, store, service, handler, router.
API: /orgs/:owner/skills (CRUD + fork + versions)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: ClaudeSettings 엔티티 추가

**Files:** Task 2와 동일 패턴. `skill` → `claude_settings`, 테이블명 `skills` → `claude_settings`.

- [ ] **Step 1: Migration** (`000010_create_claude_settings.up.sql`)

```sql
CREATE TABLE claude_settings (
    id             BIGSERIAL PRIMARY KEY,
    owner_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    lower_name     TEXT NOT NULL,
    content        TEXT NOT NULL,
    visibility     SMALLINT NOT NULL DEFAULT 0,
    is_fork        BOOLEAN NOT NULL DEFAULT false,
    fork_id        BIGINT REFERENCES claude_settings(id) ON DELETE SET NULL,
    fork_count     INT NOT NULL DEFAULT 0,
    latest_version INT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, lower_name)
);

CREATE INDEX idx_claude_settings_owner_id ON claude_settings(owner_id);

CREATE TABLE claude_settings_versions (
    id                 BIGSERIAL PRIMARY KEY,
    claude_settings_id BIGINT NOT NULL REFERENCES claude_settings(id) ON DELETE CASCADE,
    version            INT NOT NULL,
    content            JSONB NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(claude_settings_id, version)
);

CREATE INDEX idx_claude_settings_versions_claude_settings_id ON claude_settings_versions(claude_settings_id);
```

- [ ] **Step 2-7: Domain, Store, Service, Handler, Router, DI** — Skill과 동일 패턴으로 `ClaudeSettings` 생성. Content 필드는 valid JSON (settings.json 형식).

- [ ] **Step 8: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: ClaudeSettings 엔티티 추가

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Member 엔티티 추가

**Files:**
- Create: `migrations/000011_create_members.up.sql`
- Create: `migrations/000011_create_members.down.sql`
- Create: `internal/domain/member.go`
- Create: `internal/domain/member_version.go`
- Create: `internal/db/member_store.go`
- Create: `internal/services/member/member.go`
- Create: `internal/handler/member.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Migration**

```sql
-- 000011_create_members.up.sql
CREATE TABLE members (
    id                 BIGSERIAL PRIMARY KEY,
    owner_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    lower_name         TEXT NOT NULL,
    command            TEXT NOT NULL,
    git_repo_url       TEXT NOT NULL DEFAULT '',
    claude_md_id       BIGINT REFERENCES claude_mds(id) ON DELETE SET NULL,
    claude_settings_id BIGINT REFERENCES claude_settings(id) ON DELETE SET NULL,
    visibility         SMALLINT NOT NULL DEFAULT 0,
    is_fork            BOOLEAN NOT NULL DEFAULT false,
    fork_id            BIGINT REFERENCES members(id) ON DELETE SET NULL,
    fork_count         INT NOT NULL DEFAULT 0,
    latest_version     INT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, lower_name)
);

CREATE INDEX idx_members_owner_id ON members(owner_id);

CREATE TABLE member_skills (
    member_id BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    skill_id  BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (member_id, skill_id)
);

CREATE TABLE member_versions (
    id         BIGSERIAL PRIMARY KEY,
    member_id  BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    version    INT NOT NULL,
    content    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(member_id, version)
);

CREATE INDEX idx_member_versions_member_id ON member_versions(member_id);
```

- [ ] **Step 2: Domain**

`internal/domain/member.go`:

```go
package domain

import "time"

type Member struct {
	ID               int64      `db:"id" json:"id"`
	OwnerID          int64      `db:"owner_id" json:"owner_id"`
	Name             string     `db:"name" json:"name"`
	LowerName        string     `db:"lower_name" json:"-"`
	Command          string     `db:"command" json:"command"`
	GitRepoURL       string     `db:"git_repo_url" json:"git_repo_url"`
	ClaudeMdID       *int64     `db:"claude_md_id" json:"claude_md_id,omitempty"`
	ClaudeSettingsID *int64     `db:"claude_settings_id" json:"claude_settings_id,omitempty"`
	Visibility       Visibility `db:"visibility" json:"visibility"`
	IsFork           bool       `db:"is_fork" json:"is_fork"`
	ForkID           *int64     `db:"fork_id" json:"fork_id,omitempty"`
	ForkCount        int        `db:"fork_count" json:"fork_count"`
	LatestVersion    *int       `db:"latest_version" json:"latest_version,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

type MemberView struct {
	Member
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
	ForkName       *string `db:"fork_name" json:"fork_name,omitempty"`
	ForkOwnerLogin *string `db:"fork_owner_login" json:"fork_owner_login,omitempty"`
	SkillIDs       []int64 `json:"skill_ids"`
}
```

`internal/domain/member_version.go`:

```go
package domain

import (
	"encoding/json"
	"time"
)

type MemberVersion struct {
	ID        int64           `db:"id" json:"id"`
	MemberID  int64           `db:"member_id" json:"member_id"`
	Version   int             `db:"version" json:"version"`
	Content   json.RawMessage `db:"content" json:"content"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type MemberVersionView struct {
	MemberVersion
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
}

type MemberSnapshot struct {
	Command          string  `json:"command"`
	GitRepoURL       string  `json:"git_repo_url"`
	ClaudeMdID       *int64  `json:"claude_md_id,omitempty"`
	SkillIDs         []int64 `json:"skill_ids,omitempty"`
	ClaudeSettingsID *int64  `json:"claude_settings_id,omitempty"`
}
```

- [ ] **Step 3: Store — Member 특화 로직**

Member store는 빌딩블록 패턴 + member_skills 조인 테이블 관리가 추가됨:

```go
// Create에서 member_skills 함께 삽입
func (s *MemberStore) Create(ctx context.Context, m *domain.Member, skillIDs []int64) error {
	// INSERT members
	// INSERT member_skills for each skillID
}

// GetSkillIDs — member의 skill ID 목록 조회
func (s *MemberStore) GetSkillIDs(ctx context.Context, memberID int64) ([]int64, error) {
	query := "SELECT skill_id FROM member_skills WHERE member_id = $1"
	// ...
}

// ReplaceSkillIDs — member_skills 교체
func (s *MemberStore) ReplaceSkillIDs(ctx context.Context, memberID int64, skillIDs []int64) error {
	// DELETE FROM member_skills WHERE member_id = $1
	// INSERT member_skills for each skillID
}
```

- [ ] **Step 4: Service — Member CRUD + Fork**

Member service는 snapshot에 skillIDs 포함:

```go
func (s *Service) snapshot(m *domain.Member, skillIDs []int64) (json.RawMessage, error) {
	snap := domain.MemberSnapshot{
		Command:          m.Command,
		GitRepoURL:       m.GitRepoURL,
		ClaudeMdID:       m.ClaudeMdID,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: m.ClaudeSettingsID,
	}
	return json.Marshal(snap)
}
```

- [ ] **Step 5: Handler, Router, DI** — 빌딩블록 패턴과 동일. API 경로: `/orgs/:owner/members`.

- [ ] **Step 6: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: Member 엔티티 추가

빌딩블록 FK 참조 (claude_md_id, claude_settings_id) +
member_skills 조인 테이블. Command 필드 포함.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: Team 엔티티 추가

**Files:**
- Create: `migrations/000012_create_teams.up.sql`
- Create: `internal/domain/team.go`
- Create: `internal/domain/team_version.go`
- Create: `internal/db/team_store.go`
- Create: `internal/services/team/team.go`
- Create: `internal/handler/team.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Migration**

```sql
-- 000012_create_teams.up.sql
CREATE TABLE teams (
    id                  BIGSERIAL PRIMARY KEY,
    owner_id            BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    lower_name          TEXT NOT NULL,
    root_team_member_id BIGINT,
    visibility          SMALLINT NOT NULL DEFAULT 0,
    is_fork             BOOLEAN NOT NULL DEFAULT false,
    fork_id             BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    fork_count          INT NOT NULL DEFAULT 0,
    latest_version      INT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(owner_id, lower_name)
);

CREATE INDEX idx_teams_owner_id ON teams(owner_id);

CREATE TABLE team_members (
    id        BIGSERIAL PRIMARY KEY,
    team_id   BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    member_id BIGINT NOT NULL REFERENCES members(id),
    name      TEXT NOT NULL
);

CREATE TABLE team_relations (
    team_id             BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    from_team_member_id BIGINT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    to_team_member_id   BIGINT NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, from_team_member_id, to_team_member_id)
);

CREATE TABLE team_versions (
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    version    INT NOT NULL,
    content    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(team_id, version)
);

CREATE INDEX idx_team_versions_team_id ON team_versions(team_id);
```

- [ ] **Step 2: Domain**

`internal/domain/team.go`:

```go
package domain

import "time"

type Team struct {
	ID               int64      `db:"id" json:"id"`
	OwnerID          int64      `db:"owner_id" json:"owner_id"`
	Name             string     `db:"name" json:"name"`
	LowerName        string     `db:"lower_name" json:"-"`
	RootTeamMemberID *int64     `db:"root_team_member_id" json:"root_team_member_id,omitempty"`
	Visibility       Visibility `db:"visibility" json:"visibility"`
	IsFork           bool       `db:"is_fork" json:"is_fork"`
	ForkID           *int64     `db:"fork_id" json:"fork_id,omitempty"`
	ForkCount        int        `db:"fork_count" json:"fork_count"`
	LatestVersion    *int       `db:"latest_version" json:"latest_version,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

type TeamView struct {
	Team
	OwnerLogin     string         `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string        `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
	ForkName       *string        `db:"fork_name" json:"fork_name,omitempty"`
	ForkOwnerLogin *string        `db:"fork_owner_login" json:"fork_owner_login,omitempty"`
	TeamMembers    []TeamMember   `json:"team_members"`
	Relations      []TeamRelation `json:"relations"`
}

type TeamMember struct {
	ID       int64  `db:"id" json:"id"`
	TeamID   int64  `db:"team_id" json:"team_id"`
	MemberID int64  `db:"member_id" json:"member_id"`
	Name     string `db:"name" json:"name"`
}

type TeamRelation struct {
	TeamID           int64 `db:"team_id" json:"team_id"`
	FromTeamMemberID int64 `db:"from_team_member_id" json:"from_team_member_id"`
	ToTeamMemberID   int64 `db:"to_team_member_id" json:"to_team_member_id"`
}
```

`internal/domain/team_version.go`:

```go
package domain

import (
	"encoding/json"
	"time"
)

type TeamVersion struct {
	ID        int64           `db:"id" json:"id"`
	TeamID    int64           `db:"team_id" json:"team_id"`
	Version   int             `db:"version" json:"version"`
	Content   json.RawMessage `db:"content" json:"content"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type TeamVersionView struct {
	TeamVersion
	OwnerLogin     string  `db:"owner_login" json:"owner_login"`
	OwnerAvatarURL *string `db:"owner_avatar_url" json:"owner_avatar_url,omitempty"`
}

type TeamSnapshot struct {
	RootTeamMemberID *int64               `json:"root_team_member_id,omitempty"`
	TeamMembers      []TeamMemberSnapshot `json:"team_members"`
	Relations        []RelationSnapshot   `json:"relations"`
}

type TeamMemberSnapshot struct {
	ID       int64  `json:"id"`
	MemberID int64  `json:"member_id"`
	Name     string `json:"name"`
}

type RelationSnapshot struct {
	FromTeamMemberID int64 `json:"from_team_member_id"`
	ToTeamMemberID   int64 `json:"to_team_member_id"`
}
```

- [ ] **Step 3: Store** — team_members, team_relations 하위 테이블 관리 포함. Create 시 team + team_members + team_relations 트랜잭션 처리. GetView 시 team_members, team_relations JOIN.

- [ ] **Step 4: Service** — snapshot에 TeamMembers/Relations 포함. Fork 시 team_members/relations 복사.

- [ ] **Step 5: Handler, Router, DI**

```go
// Router
pub.GET("/orgs/:owner/teams", teamHandler.List)
pub.GET("/orgs/:owner/teams/:name", teamHandler.Get)
priv.POST("/orgs/:owner/teams", teamHandler.Create)
priv.PUT("/orgs/:owner/teams/:name", teamHandler.Update)
priv.DELETE("/orgs/:owner/teams/:name", teamHandler.Delete)
priv.POST("/orgs/:owner/teams/:name/fork", teamHandler.Fork)
```

- [ ] **Step 6: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: Team 엔티티 추가

team_members + team_relations 하위 테이블.
TeamSnapshot에 전체 구조 스냅샷 저장.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: Run 엔티티 추가 (개인 전용)

**Files:**
- Create: `migrations/000013_create_runs.up.sql`
- Create: `internal/domain/run.go`
- Create: `internal/db/run_store.go`
- Create: `internal/services/run/run.go`
- Create: `internal/handler/run.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Migration**

```sql
-- 000013_create_runs.up.sql
CREATE TABLE runs (
    id         TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    team_id    BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    member_id  BIGINT REFERENCES members(id) ON DELETE SET NULL,
    status     TEXT NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    stopped_at TIMESTAMPTZ
);

CREATE INDEX idx_runs_user_id ON runs(user_id);

CREATE TABLE run_messages (
    id                  TEXT PRIMARY KEY,
    run_id              TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    from_team_member_id TEXT NOT NULL,
    to_team_member_id   TEXT NOT NULL,
    content             TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_run_messages_run_id ON run_messages(run_id);

CREATE TABLE run_notes (
    id             TEXT PRIMARY KEY,
    run_id         TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    team_member_id TEXT NOT NULL,
    content        TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_run_notes_run_id ON run_notes(run_id);
```

- [ ] **Step 2: Domain — 공유 가능 리소스 패턴과 다름**

`internal/domain/run.go`:

```go
package domain

import "time"

type RunStatus string

const (
	RunRunning RunStatus = "running"
	RunStopped RunStatus = "stopped"
)

type Run struct {
	ID        string     `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"user_id"`
	Name      string     `db:"name" json:"name"`
	TeamID    *int64     `db:"team_id" json:"team_id,omitempty"`
	MemberID  *int64     `db:"member_id" json:"member_id,omitempty"`
	Status    RunStatus  `db:"status" json:"status"`
	Messages  []Message  `json:"messages"`
	Notes     []Note     `json:"notes"`
	StartedAt time.Time  `db:"started_at" json:"started_at"`
	StoppedAt *time.Time `db:"stopped_at" json:"stopped_at,omitempty"`
}

type Message struct {
	ID               string    `db:"id" json:"id"`
	FromTeamMemberID string    `db:"from_team_member_id" json:"from_team_member_id"`
	ToTeamMemberID   string    `db:"to_team_member_id" json:"to_team_member_id"`
	Content          string    `db:"content" json:"content"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type Note struct {
	ID           string    `db:"id" json:"id"`
	TeamMemberID string    `db:"team_member_id" json:"team_member_id"`
	Content      string    `db:"content" json:"content"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
```

- [ ] **Step 3: Store** — Run CRUD + Messages/Notes 서브 리소스. Visibility/Fork/Version 없음.

```go
type RunStore struct { /* ... */ }

func (s *RunStore) Create(ctx context.Context, r *domain.Run) error { /* INSERT runs */ }
func (s *RunStore) Get(ctx context.Context, id string) (*domain.Run, error) { /* SELECT + messages + notes */ }
func (s *RunStore) ListByUser(ctx context.Context, userID int64) ([]domain.Run, error) { /* SELECT runs WHERE user_id */ }
func (s *RunStore) UpdateStatus(ctx context.Context, id string, status domain.RunStatus, stoppedAt *time.Time) error { /* UPDATE */ }
func (s *RunStore) AddMessage(ctx context.Context, runID string, m *domain.Message) error { /* INSERT run_messages */ }
func (s *RunStore) AddNote(ctx context.Context, runID string, n *domain.Note) error { /* INSERT run_notes */ }
```

- [ ] **Step 4: Service** — Create, Get, List, Stop, AddMessage, AddNote.

- [ ] **Step 5: Handler, Router**

```go
// All RequireAuth (개인 전용)
priv.POST("/runs", runHandler.Create)
priv.GET("/runs", runHandler.List)
priv.GET("/runs/:id", runHandler.Get)
priv.PATCH("/runs/:id", runHandler.UpdateStatus)
priv.POST("/runs/:id/messages", runHandler.AddMessage)
priv.POST("/runs/:id/notes", runHandler.AddNote)
```

- [ ] **Step 6: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: Run 엔티티 추가

개인 전용 (Visibility/Fork/Version 없음).
Messages, Notes 하위 리소스.
API: /runs (CRUD + messages + notes)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 7: GitRepo 엔티티 삭제

설계 문서에 따라 GitRepo는 Member.GitRepoURL 필드로 대체됨.

- [ ] **Step 1: Migration**

```sql
-- 000014_drop_git_repos.up.sql
DROP TABLE IF EXISTS git_repo_versions;
DROP TABLE IF EXISTS git_repos;
```

- [ ] **Step 2: 관련 코드 삭제**

```bash
rm internal/domain/git_repo.go internal/domain/git_repo_version.go
rm internal/db/git_repo_store.go
rm -rf internal/services/gitrepo/
rm internal/handler/git_repo.go
```

- [ ] **Step 3: Router에서 git repo 경로 제거, main.go에서 DI 제거**

- [ ] **Step 4: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "refactor: GitRepo 엔티티 삭제

Member.GitRepoURL 필드로 대체됨.

Co-Authored-By: Claude <noreply@anthropic.com>"
```
