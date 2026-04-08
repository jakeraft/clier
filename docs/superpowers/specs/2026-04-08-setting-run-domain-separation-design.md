# Workspace / Run Domain Separation Design

## Background

clier의 도메인 엔티티가 마켓플레이스 공유(Workspace)와 로컬 실행(Run)의 관심사를 혼재하고 있다.
`clier-server`를 통한 마켓플레이스를 구축하려면, 공유 가능한 스펙과 로컬 런타임을 명확히 분리해야 한다.

### 현재 문제

- `Member`에 공유 가능한 스펙(Name, Model, BuildingBlock FK)과 런타임(Args, GitRepoURL)이 혼재
- `AgentDotMd`라는 추상화가 실제 파일(`CLAUDE.md`)과 괴리
- `ClaudeJson` 빌딩블록이 존재하나, `.claude.json`은 프로젝트 레벨이 없어 사실상 불필요
- `AgentType` 필드가 존재하나, `command`에서 바이너리를 감지하면 불필요
- 시스템이 사용자 콘텐츠에 몰래 머지(CLAUDE.md 앞에 protocol 삽입, .claude.json deep merge)

### 설계 원칙

1. **서버 = source of truth**: Workspace 엔티티는 서버에만 저장. CLI는 API 클라이언트로서 서버와 통신하며 로컬 DB에 Workspace 엔티티를 저장하지 않는다.
2. **Workspace = Run 동일 레이아웃**: `clier workspace` 결과물과 `clier task start` 실행 환경이 동일한 디렉토리 구조.
3. **시스템 주입 투명화**: 숨겨진 머지 제거. Team Protocol도 workspace 생성 시 포함.

## Design

### Two-Domain Architecture

```
Workspace Domain (서버 + 파일)        Run Domain (로컬 실행)
──────────────────────────────       ──────────────────────────
서버에서 스펙 CRUD                     Task 오케스트레이션
워크스페이스 파일 생성                   env vars export
Team Protocol 포함                    프로세스 실행

서버: BuildingBlock, Member, Team     로컬 SQLite: Task만
CLI: API 호출 + 파일 쓰기             CLI: Plan 실행
로컬 DB 없음                          
```

### Workspace Domain

#### 서버가 source of truth

Workspace 엔티티(BuildingBlock, Member, Team)는 서버(PostgreSQL)에만 저장된다.
CLI는 로컬 DB에 이들을 저장하지 않는다.

```
clier member create    → POST /api/v1/orgs/:owner/members (서버)
clier member list      → GET  /api/v1/orgs/:owner/members (서버)
clier workspace member → GET  /api/v1/... → 워크스페이스 파일 생성 (로컬)
clier task create      → 서버에서 스펙 fetch → Plan 빌드 → SQLite 저장 (로컬)
clier task start       → SQLite에서 Plan 읽기 → 실행 (서버 불필요)
```

CLI의 Go struct는 서버 API 응답 파싱 + workspace 파일 생성에 사용.

#### 서버 엔티티 공통 패턴

clier-server의 모든 리소스 엔티티는 동일한 패턴을 따른다.

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
    ID            int64      `json:"id"`
    OwnerID       int64      `json:"owner_id"`
    Name          string     `json:"name"`
    Content       string     `json:"content"`
    Visibility    Visibility `json:"visibility"`
    IsFork        bool       `json:"is_fork"`
    ForkID        *int64     `json:"fork_id,omitempty"`
    ForkCount     int        `json:"fork_count"`
    LatestVersion *int       `json:"latest_version,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}
```

Skill, ClaudeSettings도 동일 패턴. Content 필드에 각각 스킬 마크다운, settings.json 내용 저장.

#### Member

단독 실행 가능한 에이전트 스펙. 빌딩 블록을 FK로 참조.

```go
type Member struct {
    ID               int64      `json:"id"`
    OwnerID          int64      `json:"owner_id"`
    Name             string     `json:"name"`
    Command          string     `json:"command"`        // "claude --dangerously-skip-permissions"
    GitRepoURL       string     `json:"git_repo_url"`
    ClaudeMdID       *int64     `json:"claude_md_id,omitempty"`
    ClaudeSettingsID *int64     `json:"claude_settings_id,omitempty"`
    // SkillIDs → member_skills 조인 테이블 (API 응답에 포함)
    Visibility       Visibility `json:"visibility"`
    IsFork           bool       `json:"is_fork"`
    ForkID           *int64     `json:"fork_id,omitempty"`
    ForkCount        int        `json:"fork_count"`
    LatestVersion    *int       `json:"latest_version,omitempty"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}
```

변경점:
- `AgentType` 삭제: `Command`의 첫 단어(바이너리명)에서 감지
- `Model` 삭제: `ClaudeSettings`의 `settings.json` → `"model"` 필드
- `Args` 삭제: `Command`에 통합
- `ClaudeJsonID` 삭제: `.claude.json` 빌딩블록 제거

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
    ID               int64      `json:"id"`
    OwnerID          int64      `json:"owner_id"`
    Name             string     `json:"name"`
    RootTeamMemberID *int64     `json:"root_team_member_id,omitempty"`
    TeamMembers      []TeamMember  `json:"team_members"`
    Relations        []TeamRelation `json:"relations"`
    Visibility       Visibility `json:"visibility"`
    IsFork           bool       `json:"is_fork"`
    ForkID           *int64     `json:"fork_id,omitempty"`
    ForkCount        int        `json:"fork_count"`
    LatestVersion    *int       `json:"latest_version,omitempty"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

type TeamMember struct {
    ID       int64  `json:"id"`
    TeamID   int64  `json:"team_id"`
    MemberID int64  `json:"member_id"`   // FK → Member
    Name     string `json:"name"`        // 팀 내 역할명
}

type TeamRelation struct {
    TeamID           int64 `json:"team_id"`
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

### Unified Workspace Layout

`clier workspace` 결과물과 `clier task start` 실행 환경이 동일한 디렉토리 구조.
Team Protocol도 workspace 생성 시 포함된다.

**Member 단독:**

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

**Team (각 멤버별):**

```
{memberspace}/
├── CLAUDE.md                      ← Team Protocol (workspace 생성 시 포함)
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/
│       └── {name}/SKILL.md        ← Skill
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
```

Team Protocol은 부모 디렉토리 CLAUDE.md로 배치.
Claude Code가 cwd에서 부모 디렉토리로 올라가며 자동 로드 (공식 문서 확인).
사용자의 ClaudeMd(`project/CLAUDE.md`)와 머지 없이 분리.

**workspace 생성 시:**
- 서버에서 스펙 fetch → 빌딩블록 내용 resolve → 위 레이아웃으로 파일 생성
- Team의 경우 Team Protocol도 이 시점에 생성
- 사용자가 project/ 안에서 git clone 후 에이전트 직접 실행 가능

**Run 시:**
- 워크스페이스 파일 변경 없음
- Env vars export + command 실행만

### Run Domain — 로컬 전용

#### Task

Member 단독 실행 또는 Team 실행을 지원하는 오케스트레이션 단위.
Run Domain 엔티티는 서버에 올라가지 않으므로 기존 ID 체계(string UUID) 유지.
로컬 SQLite에 저장.

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
    Terminal     TerminalPlan
    Workspace    WorkspacePlan
}
```

#### 환경변수 (Env)

Run 시 유일한 시스템 주입. 파일 변경 없이 export만.

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

### 시스템 주입 정리

| 주입 | 도메인 | 시점 | 방식 |
|---|---|---|---|
| Team Protocol | Workspace | workspace 생성 시 | 부모 디렉토리 CLAUDE.md 파일 |
| Env vars (9개) | Run | task 실행 시 | export (파일 변경 없음) |
| ~~.claude.json 온보딩~~ | ~~삭제~~ | - | - |
| ~~CLAUDE.md 머지~~ | ~~삭제~~ | - | - |
| ~~.claude.json 머지~~ | ~~삭제~~ | - | - |

## 현재 대비 변경 요약

### 아키텍처 변경

| 항목 | 현재 | 변경 |
|---|---|---|
| Workspace 엔티티 저장 | 로컬 SQLite | 서버 (API 호출) |
| CLI 로컬 DB | 모든 엔티티 | Task만 |
| 도메인명 | - | Workspace Domain / Run Domain |
| Team Protocol 생성 시점 | Run 시 | Workspace 생성 시 |

### 엔티티 변경

| 항목 | 변경 |
|---|---|
| `AgentDotMd` | → `ClaudeMd` 리네임 |
| `ClaudeJson` | 삭제 |
| `Member.AgentType` | 삭제 (`Command`에서 감지) |
| `Member.Model` | 삭제 (`ClaudeSettings` → `settings.json`의 `"model"`) |
| `Member.Args` | 삭제 (`Command`에 통합) |
| `Member.Command` | 신규 (바이너리 + CLI flags) |
| `Task.MemberID` | 신규 (Member 단독 실행 지원) |
| `Task.Bindings` | 신규 (런타임 override) |
| `MemberPlan.Env` | 신규 (환경변수 명시) |
| `MemberPlan.Protocol` | 삭제 (workspace 파일에 이미 포함) |
| Workspace | workspace 생성과 Run 동일 레이아웃 |

### 파일 변경 (예상)

| 파일 | 변경 |
|---|---|
| `internal/domain/resource/agent_dot_md.go` | → `claude_md.go` 리네임 |
| `internal/domain/resource/claudejson.go` | 삭제 |
| `internal/domain/resource/skill.go` | 서버 스키마 적용 (db 태그 제거, json만) |
| `internal/domain/resource/claude_settings.go` | 서버 스키마 적용 |
| `internal/domain/member.go` | 전면 재설계 (서버 스키마, Command 추가) |
| `internal/domain/team.go` | 서버 스키마 적용 |
| `internal/domain/task.go` | MemberID, Bindings 추가 |
| `internal/app/task/plan.go` | 빌드 로직 수정 (서버 API fetch, 머지 제거) |
| `internal/app/task/prompt.go` | workspace 생성 시 Protocol 파일 생성 |
| `internal/app/task/workspace_files.go` | 머지 로직 제거, 통합 레이아웃 적용 |
| `internal/app/task/command.go` | detectRuntime 기반으로 리팩토링 |
| `internal/adapter/runtime/` | AgentRuntime 인터페이스 단순화 |
| `internal/adapter/db/schema.sql` | Workspace 테이블 전부 삭제, Task만 남김 |
| `internal/adapter/db/` | Workspace store 파일 삭제, Task store만 유지 |

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
# Workspace: 서버에서 스펙 관리
clier member create --name react-reviewer --command "claude" ...   # 서버에 생성
clier member list                                                   # 서버에서 조회

# Workspace: 작업공간 생성
clier workspace member jake/react-reviewer     # 서버에서 fetch → 파일 생성
clier workspace team jake/dev-squad            # 멤버별 파일 + Protocol 생성

# Workspace: fork 후 커스터마이징
clier fork member jake/react-reviewer          # 서버에 복사본 생성
clier member update myname/react-reviewer ...  # 서버에서 수정

# Run: 단독 실행 (워크스페이스 그대로 사용)
clier task create --member myname/react-reviewer
> repo? → https://github.com/my/project
clier task start --id xxx                      # env export + command 실행

# Run: 팀 실행
clier task create --team myname/dev-squad
clier task start --id xxx

# Run: 재실행 (bindings 재사용)
clier task create --from task-abc123
```

## Scope

### In scope

- Workspace / Run 도메인 분리
- Workspace 엔티티를 서버 전용으로 전환 (로컬 DB에서 제거)
- Building Block 정비 (ClaudeJson 삭제, AgentDotMd → ClaudeMd)
- Member 엔티티 리팩토링 (AgentType/Model/Args 삭제, Command 추가)
- Task 엔티티 확장 (MemberID, Bindings)
- 통합 Workspace 레이아웃 (workspace 생성 = Run 환경)
- Team Protocol을 workspace 생성 시 포함
- 머지 로직 전부 제거
- 로컬 DB를 Task 전용으로 축소

### Out of scope

- UI (서버에서 구현)
- 서버 엔티티 추가 (별도 작업)
- workspace / fork CLI 커맨드 구현 (별도 작업)
- 마켓플레이스 검색/필터링 (별도 작업)
