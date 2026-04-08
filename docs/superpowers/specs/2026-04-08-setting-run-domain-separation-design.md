# Setting / Run Domain Separation Design

## Background

clier의 도메인 엔티티가 마켓플레이스 공유(Setting)와 로컬 실행(Run)의 관심사를 혼재하고 있다.
`clier-server`를 통한 마켓플레이스를 구축하려면, 공유 가능한 스펙과 로컬 런타임을 명확히 분리해야 한다.

### 현재 문제

- `Member`에 공유 가능한 스펙(Name, Model, BuildingBlock FK)과 런타임(Args, GitRepoURL)이 혼재
- `AgentDotMd`라는 추상화가 실제 파일(`CLAUDE.md`)과 괴리
- `ClaudeJson` 빌딩블록이 존재하나, `.claude.json`은 프로젝트 레벨이 없어 사실상 불필요
- `AgentType` 필드가 존재하나, `command`에서 바이너리를 감지하면 불필요
- 시스템이 사용자 콘텐츠에 몰래 머지(CLAUDE.md 앞에 protocol 삽입, .claude.json deep merge)
- clier와 clier-server의 엔티티 구조가 완전히 다름 (UUID vs int64, 서버 필드 누락)

### 설계 원칙

1. **서버 스키마 기준**: clier-server의 엔티티 패턴을 기준으로 clier 엔티티를 재설계
2. **다운로드 = Run 워크스페이스**: 다운로드 결과물과 Task 실행 워크스페이스가 동일한 레이아웃
3. **시스템 주입 투명화**: 숨겨진 머지 제거, 모든 주입을 Plan에 명시

## Design

### Two-Domain Architecture

```
Setting Domain (Marketplace)          Run Domain (Local Orchestration)
─────────────────────────────         ──────────────────────────────────
파일 세팅, 다운로드, 공유              실행 오케스트레이션, 프로토콜 주입
서버와 동일한 스키마                    로컬 전용

BuildingBlock (3종)                   Task
Member (단독 실행 가능 스펙)             ├─ MemberBinding (런타임 override)
Team (멤버 조합 + 관계)                 └─ MemberPlan (투명한 실행 명세)
```

### Setting Domain — 서버 스키마 기준

#### 서버 엔티티 공통 패턴

clier-server의 모든 리소스 엔티티는 동일한 패턴을 따른다.
clier도 이 패턴을 그대로 사용한다.

```go
// 서버 공통 필드 (모든 리소스 엔티티가 포함)
ID            int64       // BIGSERIAL PRIMARY KEY
OwnerID       int64       // FK → users
Name          string      // 표시명
LowerName     string      // 검색/유니크용 (UNIQUE with OwnerID)
Visibility    Visibility  // 0=Public, 1=Private
IsFork        bool
ForkID        *int64      // FK → self (ON DELETE SET NULL)
ForkCount     int
LatestVersion *int
CreatedAt     time.Time
UpdatedAt     time.Time
```

```go
// 버전 엔티티 공통 패턴
ID         int64
ResourceID int64            // FK → 리소스 (ON DELETE CASCADE)
Version    int              // UNIQUE with ResourceID
Content    json.RawMessage  // JSONB 스냅샷
CreatedAt  time.Time
```

#### Building Blocks (3종)

`ClaudeJson` 삭제. `.claude.json`은 Claude Code의 user-level 런타임 상태 파일이며
프로젝트 레벨이 존재하지 않는다 (공식 문서 확인). 사용자가 필요한 설정은 전부 `settings.json`으로 커버 가능.

| Building Block | 생성 파일 | 담당 |
|---|---|---|
| **ClaudeMd** | `project/CLAUDE.md` | 에이전트 지시사항 |
| **Skill** | `.claude/skills/{name}/SKILL.md` | 에이전트 스킬 |
| **ClaudeSettings** | `.claude/settings.json` | model, permissions, hooks, env |

##### ClaudeMd (AgentDotMd에서 리네임)

```go
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

type ClaudeMdVersion struct {
    ID         int64           `db:"id" json:"id"`
    ClaudeMdID int64           `db:"claude_md_id" json:"claude_md_id"`
    Version    int             `db:"version" json:"version"`
    Content    json.RawMessage `db:"content" json:"content"`
    CreatedAt  time.Time       `db:"created_at" json:"created_at"`
}

type ClaudeMdSnapshot struct {
    Content string `json:"content"`
}
```

##### Skill

```go
type Skill struct {
    ID            int64      `db:"id" json:"id"`
    OwnerID       int64      `db:"owner_id" json:"owner_id"`
    Name          string     `db:"name" json:"name"`          // lowercase + hyphens
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

type SkillVersion struct {
    ID        int64           `db:"id" json:"id"`
    SkillID   int64           `db:"skill_id" json:"skill_id"`
    Version   int             `db:"version" json:"version"`
    Content   json.RawMessage `db:"content" json:"content"`
    CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type SkillSnapshot struct {
    Content string `json:"content"`
}
```

##### ClaudeSettings

```go
type ClaudeSettings struct {
    ID            int64      `db:"id" json:"id"`
    OwnerID       int64      `db:"owner_id" json:"owner_id"`
    Name          string     `db:"name" json:"name"`
    LowerName     string     `db:"lower_name" json:"-"`
    Content       string     `db:"content" json:"content"`    // valid JSON (settings.json)
    Visibility    Visibility `db:"visibility" json:"visibility"`
    IsFork        bool       `db:"is_fork" json:"is_fork"`
    ForkID        *int64     `db:"fork_id" json:"fork_id,omitempty"`
    ForkCount     int        `db:"fork_count" json:"fork_count"`
    LatestVersion *int       `db:"latest_version" json:"latest_version,omitempty"`
    CreatedAt     time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type ClaudeSettingsVersion struct {
    ID               int64           `db:"id" json:"id"`
    ClaudeSettingsID int64           `db:"claude_settings_id" json:"claude_settings_id"`
    Version          int             `db:"version" json:"version"`
    Content          json.RawMessage `db:"content" json:"content"`
    CreatedAt        time.Time       `db:"created_at" json:"created_at"`
}

type ClaudeSettingsSnapshot struct {
    Content string `json:"content"`
}
```

#### Member

단독 실행 가능한 에이전트 스펙. 빌딩 블록을 FK로 참조.

```go
type Member struct {
    ID               int64      `db:"id" json:"id"`
    OwnerID          int64      `db:"owner_id" json:"owner_id"`
    Name             string     `db:"name" json:"name"`
    LowerName        string     `db:"lower_name" json:"-"`
    Command          string     `db:"command" json:"command"`       // "claude --dangerously-skip-permissions"
    GitRepoURL       string     `db:"git_repo_url" json:"git_repo_url"`
    ClaudeMdID       *int64     `db:"claude_md_id" json:"claude_md_id,omitempty"`
    ClaudeSettingsID *int64     `db:"claude_settings_id" json:"claude_settings_id,omitempty"`
    // SkillIDs → member_skills 조인 테이블
    Visibility       Visibility `db:"visibility" json:"visibility"`
    IsFork           bool       `db:"is_fork" json:"is_fork"`
    ForkID           *int64     `db:"fork_id" json:"fork_id,omitempty"`
    ForkCount        int        `db:"fork_count" json:"fork_count"`
    LatestVersion    *int       `db:"latest_version" json:"latest_version,omitempty"`
    CreatedAt        time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

type MemberVersion struct {
    ID        int64           `db:"id" json:"id"`
    MemberID  int64           `db:"member_id" json:"member_id"`
    Version   int             `db:"version" json:"version"`
    Content   json.RawMessage `db:"content" json:"content"`
    CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type MemberSnapshot struct {
    Command          string  `json:"command"`
    GitRepoURL       string  `json:"git_repo_url"`
    ClaudeMdID       *int64  `json:"claude_md_id,omitempty"`
    SkillIDs         []int64 `json:"skill_ids,omitempty"`
    ClaudeSettingsID *int64  `json:"claude_settings_id,omitempty"`
}
```

변경점:
- `AgentType` 삭제: `Command`의 첫 단어(바이너리명)에서 감지
- `Model` 삭제: `ClaudeSettings`의 `settings.json` → `"model"` 필드
- `Args` 삭제: `Command`에 통합
- `ClaudeJsonID` 삭제: `.claude.json` 빌딩블록 제거
- ID 체계: UUID(string) → int64 (서버와 통일)
- 서버 공통 필드 추가: OwnerID, Visibility, Fork 관련, LatestVersion

`Command` 필드:
- 바이너리 + CLI flags 포함 (예: `"claude --dangerously-skip-permissions"`)
- Run Domain에서 바이너리 감지에 사용 (인증 토큰 주입, 프로토콜 파일명 결정)
- settings.json의 model/permissions와 겹칠 수 있으며, CLI가 최종 우선순위

바이너리 감지:

```go
func detectRuntime(command string) Runtime {
    binary := strings.Fields(command)[0]
    switch binary {
    case "claude":
        return ClaudeRuntime   // 인증: CLAUDE_CODE_OAUTH_TOKEN, 프로토콜: CLAUDE.md
    case "codex":
        return CodexRuntime    // 인증: OPENAI_API_KEY, 프로토콜: AGENTS.md
    default:
        return GenericRuntime
    }
}
```

#### Team

멤버 조합과 관계를 정의하는 스펙.

```go
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
    // TeamMembers, Relations → 별도 테이블
}

type TeamMember struct {
    ID       int64  `db:"id" json:"id"`
    TeamID   int64  `db:"team_id" json:"team_id"`
    MemberID int64  `db:"member_id" json:"member_id"`   // FK → Member
    Name     string `db:"name" json:"name"`              // 팀 내 역할명
}

type TeamRelation struct {
    TeamID           int64 `db:"team_id" json:"team_id"`
    FromTeamMemberID int64 `db:"from_team_member_id" json:"from_team_member_id"`
    ToTeamMemberID   int64 `db:"to_team_member_id" json:"to_team_member_id"`
}

type TeamVersion struct {
    ID        int64           `db:"id" json:"id"`
    TeamID    int64           `db:"team_id" json:"team_id"`
    Version   int             `db:"version" json:"version"`
    Content   json.RawMessage `db:"content" json:"content"`
    CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type TeamSnapshot struct {
    RootTeamMemberID *int64              `json:"root_team_member_id,omitempty"`
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

#### Fork + Ref 공유 모델

Terraform 모듈 레지스트리 벤치마킹.

```
Team (FORK - 복사해서 소유, 구조 수정 가능)
├── TeamMembers[] → Member (REF - 서버 원본 FK 참조)
│                      ├── ClaudeMdID (REF - FK)
│                      ├── SkillIDs (REF - 조인 테이블)
│                      └── ClaudeSettingsID (REF - FK)
└── Relations[]
```

- **Fork**: 서버에 내 계정 복사본 생성. 수정 가능, 독립적.
  - Member, Team 레벨에서 fork
- **Ref**: FK로 빌딩 블록 원본 참조. 수정이 필요하면 해당 빌딩 블록을 fork.
- 현재 서버에 fork 인프라 구현 완료 (`is_fork`, `fork_id`, `fork_count`, versions)

#### DB 스키마 (Setting Domain)

서버(PostgreSQL)와 클라이언트(SQLite) 모두 동일한 테이블 구조.

```sql
-- Building Blocks
CREATE TABLE claude_mds (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id       INTEGER NOT NULL,
    name           TEXT NOT NULL,
    lower_name     TEXT NOT NULL,
    content        TEXT NOT NULL,
    visibility     INTEGER NOT NULL DEFAULT 0,
    is_fork        INTEGER NOT NULL DEFAULT 0,
    fork_id        INTEGER REFERENCES claude_mds(id) ON DELETE SET NULL,
    fork_count     INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(owner_id, lower_name)
);

CREATE TABLE skills (
    -- 동일 패턴
);

CREATE TABLE claude_settings (
    -- 동일 패턴, content는 valid JSON
);

-- Member
CREATE TABLE members (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id         INTEGER NOT NULL,
    name             TEXT NOT NULL,
    lower_name       TEXT NOT NULL,
    command          TEXT NOT NULL,
    git_repo_url     TEXT NOT NULL DEFAULT '',
    claude_md_id     INTEGER REFERENCES claude_mds(id),
    claude_settings_id INTEGER REFERENCES claude_settings(id),
    visibility       INTEGER NOT NULL DEFAULT 0,
    is_fork          INTEGER NOT NULL DEFAULT 0,
    fork_id          INTEGER REFERENCES members(id) ON DELETE SET NULL,
    fork_count       INTEGER NOT NULL DEFAULT 0,
    latest_version   INTEGER,
    created_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at       DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(owner_id, lower_name)
);

CREATE TABLE member_skills (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    skill_id  INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (member_id, skill_id)
);

-- Team
CREATE TABLE teams (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id            INTEGER NOT NULL,
    name                TEXT NOT NULL,
    lower_name          TEXT NOT NULL,
    root_team_member_id INTEGER,
    visibility          INTEGER NOT NULL DEFAULT 0,
    is_fork             INTEGER NOT NULL DEFAULT 0,
    fork_id             INTEGER REFERENCES teams(id) ON DELETE SET NULL,
    fork_count          INTEGER NOT NULL DEFAULT 0,
    latest_version      INTEGER,
    created_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(owner_id, lower_name)
);

CREATE TABLE team_members (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    team_id   INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id),
    name      TEXT NOT NULL
);

CREATE TABLE team_relations (
    team_id             INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    from_team_member_id INTEGER NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    to_team_member_id   INTEGER NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, from_team_member_id, to_team_member_id)
);

-- Versions (각 엔티티별)
CREATE TABLE claude_md_versions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    claude_md_id INTEGER NOT NULL REFERENCES claude_mds(id) ON DELETE CASCADE,
    version      INTEGER NOT NULL,
    content      TEXT NOT NULL,  -- JSON snapshot
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(claude_md_id, version)
);
-- skill_versions, claude_settings_versions, member_versions, team_versions 동일 패턴
```

### Unified Workspace Layout

다운로드 결과물과 Task 실행 워크스페이스가 동일한 레이아웃을 사용한다.

**Member 단독 (다운로드 및 실행 공통):**

```
{memberspace}/
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/
│       └── {name}/SKILL.md        ← Skill
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
```

**Team (다운로드 및 실행 공통, 각 멤버별):**

```
{memberspace}/
├── CLAUDE.md                      ← Team Protocol (Run 시에만 생성)
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/
│       └── {name}/SKILL.md        ← Skill
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
```

다운로드 시:
- Building Block 파일들이 위 레이아웃으로 설치됨
- Team의 경우 부모 CLAUDE.md는 비어있거나 없음 (프로토콜은 Run 시 생성)
- 사용자가 project/ 안에서 git clone, 에이전트 직접 실행 가능

Run 시:
- 다운로드와 동일한 레이아웃 사용 (재생성 아님)
- Team 실행의 경우 부모 CLAUDE.md에 Team Protocol 생성
- Env vars export 후 command 실행

### Run Domain — 로컬 전용

#### Task

Member 단독 실행 또는 Team 실행을 지원하는 오케스트레이션 단위.
Run Domain 엔티티는 서버에 올라가지 않으므로 기존 ID 체계(string UUID) 유지.

```go
type Task struct {
    ID        string
    Name      string
    TeamID    *int64              // Team 실행 시 (MemberID와 배타적)
    MemberID  *int64              // Member 단독 실행 시 (TeamID와 배타적)
    Bindings  []MemberBinding     // 런타임 override
    Status    TaskStatus          // "running" | "stopped"
    Plan      []MemberPlan        // 투명한 실행 명세
    CreatedAt time.Time
    StoppedAt *time.Time
}
```

#### MemberBinding

런타임 override. 스펙의 기본값을 변경할 때만 사용.

```go
type MemberBinding struct {
    TeamMemberID int64    // Team 실행 시 slot 지정 (solo면 0)
    GitRepoURL   string   // repo override (빈값 = Member 스펙 사용)
}
```

#### MemberPlan

실행의 완전한 명세. 숨겨진 주입 없음. Plan에 없으면 주입 안 됨.

```go
type MemberPlan struct {
    TeamMemberID int64
    MemberName   string
    Env          map[string]string  // 모든 환경변수 명시
    Protocol     string             // Team Protocol 전문 (solo면 빈값)
    Terminal     TerminalPlan
    Workspace    WorkspacePlan
}
```

#### 환경변수 (Env)

```go
env := map[string]string{
    // Clier 식별자
    "CLIER_TASK_ID":   taskID,
    "CLIER_TEAM_ID":   teamID,
    "CLIER_MEMBER_ID": memberID,

    // 에이전트 런타임 (command에서 감지)
    // Claude: CLAUDE_CONFIG_DIR, CLAUDE_CODE_OAUTH_TOKEN
    // Codex: OPENAI_API_KEY
    // → detectRuntime(command) 결과에 따라 결정

    // Git Identity
    "GIT_AUTHOR_NAME":      teamName + "/" + memberName,
    "GIT_AUTHOR_EMAIL":     "noreply@clier.com",
    "GIT_COMMITTER_NAME":   teamName + "/" + memberName,
    "GIT_COMMITTER_EMAIL":  "noreply@clier.com",
}
```

#### Team Protocol

Team 실행 시에만 생성. 부모 디렉토리 CLAUDE.md로 배치하여 Claude Code가 자동 로드.
사용자의 ClaudeMd(`project/CLAUDE.md`)와 머지 없이 분리.

```
Claude Code 로딩 순서 (공식 문서 확인):
1. project/CLAUDE.md (유저 ClaudeMd) — 로드
2. 부모 디렉토리로 올라감 → {memberspace}/CLAUDE.md (Team Protocol) — 로드
3. 둘 다 concatenate됨 (override 아님)
```

### 시스템 주입 정리

| 주입 | 도메인 | 방식 | 투명성 |
|---|---|---|---|
| Env vars (9개) | Run | export | Plan.Env에 명시 |
| Team Protocol | Run | 부모 디렉토리 CLAUDE.md | Plan.Protocol에 명시 |
| ~~.claude.json 온보딩~~ | ~~삭제~~ | - | - |
| ~~CLAUDE.md 머지~~ | ~~삭제~~ | - | - |
| ~~.claude.json 머지~~ | ~~삭제~~ | - | - |

Setting Domain에는 시스템 주입이 없다.
Run Domain의 주입은 전부 MemberPlan에 명시적으로 포함된다.

## 현재 대비 변경 요약

### 엔티티 변경

| 항목 | 변경 |
|---|---|
| 전체 Setting 엔티티 ID | string(UUID) → int64 (서버와 통일) |
| 전체 Setting 엔티티 | 서버 공통 필드 추가 (OwnerID, Visibility, Fork, Version) |
| `AgentDotMd` | → `ClaudeMd` 리네임 |
| `ClaudeJson` | 삭제 |
| `Member.AgentType` | 삭제 (`Command`에서 감지) |
| `Member.Model` | 삭제 (`ClaudeSettings` → `settings.json`의 `"model"`) |
| `Member.Args` | 삭제 (`Command`에 통합) |
| `Member.Command` | 신규 (바이너리 + CLI flags) |
| `Task.TeamID/MemberID` | string → *int64 (Setting 엔티티 참조) |
| `Task.Bindings` | 신규 (런타임 override) |
| `MemberPlan.Env` | 신규 (환경변수 명시) |
| `MemberPlan.Protocol` | 신규 (Team Protocol 명시) |
| Workspace | 다운로드와 Run 동일 레이아웃 |

### 파일 변경 (예상)

| 파일 | 변경 |
|---|---|
| `internal/domain/resource/agent_dot_md.go` | → `claude_md.go` 리네임, 서버 스키마 적용 |
| `internal/domain/resource/claudejson.go` | 삭제 |
| `internal/domain/resource/skill.go` | 서버 스키마 적용 |
| `internal/domain/resource/claude_settings.go` | 서버 스키마 적용 |
| `internal/domain/member.go` | 전면 재설계 (서버 스키마, Command 추가) |
| `internal/domain/team.go` | 서버 스키마 적용 |
| `internal/domain/task.go` | MemberID, Bindings 추가, MemberPlan 확장 |
| `internal/app/task/plan.go` | 빌드 로직 수정 (머지 제거, 통합 워크스페이스) |
| `internal/app/task/prompt.go` | Protocol을 별도 파일로 출력 (부모 디렉토리) |
| `internal/app/task/workspace_files.go` | 머지 로직 제거, 통합 레이아웃 적용 |
| `internal/app/task/command.go` | detectRuntime 기반으로 리팩토링 |
| `internal/adapter/runtime/` | AgentRuntime 인터페이스 단순화 |
| `internal/adapter/db/schema.sql` | 전면 재설계 (서버 스키마 기준) |
| `internal/adapter/db/` | 모든 store 파일 수정 |

### 서버 추가 필요 (clier-server, 별도 작업)

| 항목 | 내용 |
|---|---|
| `AgentDotMd` 리네임 | → `ClaudeMd` (테이블, 코드 전체) |
| `Skill` 추가 | 빌딩블록 (동일 패턴) |
| `ClaudeSettings` 추가 | 빌딩블록 (동일 패턴) |
| `Member` 추가 | 스펙 (FK로 빌딩블록 참조) |
| `Team` 추가 | 스펙 (FK로 멤버 참조 + 관계 테이블) |

### 사용 흐름

```bash
# Setting: 마켓에서 다운로드
clier download member jake/react-reviewer@v3
clier download team jake/dev-squad@v1

# Setting: fork 후 커스터마이징
clier fork member jake/react-reviewer
clier member update myname/react-reviewer --command "claude --max-turns 100"

# Run: 단독 실행 (다운로드된 워크스페이스 그대로 사용)
clier task create --member myname/react-reviewer
> repo? → https://github.com/my/project
clier task start --id xxx

# Run: 팀 실행
clier task create --team myname/dev-squad
> leader repo? → https://github.com/my/project
clier task start --id xxx

# Run: 재실행 (bindings 재사용)
clier task create --from task-abc123
```

## Scope

### In scope

- Setting / Run 도메인 분리
- 서버 스키마 기준으로 모든 Setting 엔티티 재설계 (int64 ID, 공통 필드)
- Building Block 정비 (ClaudeJson 삭제, AgentDotMd → ClaudeMd)
- Member 엔티티 리팩토링 (AgentType/Model/Args 삭제, Command 추가)
- Task 엔티티 확장 (MemberID, Bindings, Plan 투명화)
- 통합 Workspace 레이아웃 (다운로드 = Run)
- 머지 로직 제거
- DB 스키마 마이그레이션

### Out of scope

- UI 변경 (서버에서 구현 예정)
- 서버 엔티티 추가 (별도 작업)
- download / fork CLI 커맨드 구현 (별도 작업)
- 마켓플레이스 검색/필터링 기능 (별도 작업)
