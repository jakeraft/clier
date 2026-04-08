# Workspace / Run Domain Separation Design

Refs: jakeraft/clier#24

## Background

clier를 client-server 아키텍처로 분리한다 (#24).
서버가 모든 엔티티와 상태를 소유하고, CLI는 DB 없는 경량 런타임 도구가 된다.
이 과정에서 도메인 엔티티를 Workspace(스펙)와 Run(실행)으로 정비한다.

### 현재 문제

- CLI가 SQLite DB를 직접 소유하며 모든 엔티티를 관리
- `Member`에 공유 가능한 스펙(Name, Model, BuildingBlock FK)과 런타임(Args)이 혼재
- `AgentDotMd`라는 추상화가 실제 파일(`CLAUDE.md`)과 괴리
- `ClaudeJson` 빌딩블록이 존재하나, `.claude.json`은 프로젝트 레벨이 없어 사실상 불필요
- `AgentType` 필드가 존재하나, `command`에서 바이너리를 감지하면 불필요
- 시스템이 사용자 콘텐츠에 몰래 머지(CLAUDE.md 앞에 protocol 삽입, .claude.json deep merge)
- MemberPlan에 tmux 세션/윈도우 구조가 보이지 않음
- `task`, `run`, `session` 용어 혼재

### 설계 원칙

1. **서버 = DB**: 모든 엔티티(Workspace + Run)를 서버가 소유. CLI에 로컬 DB 없음.
2. **CLI = 경량 런타임 도구**: 서버를 DB로 사용하되, 로컬 dependency가 있는 런타임 환경 로직(tmux, workspace 파일, 프로세스 실행)을 담당.
3. **로컬 파일 기반 실행**: workspace 파일을 로컬에 생성하고, run은 로컬 파일 기준으로 실행. 실행 계획은 `.clier/{RUN_ID}.json`에 저장.
4. **시스템 주입 투명화**: 숨겨진 머지 제거. Team Protocol도 workspace 생성 시 포함.

## Design

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                    clier-server                      │
│                                                     │
│  역할: 모든 엔티티의 DB + REST API + UI              │
│                                                     │
│  DB (PostgreSQL)                                    │
│  ├─ Workspace (공유 가능): ClaudeMd, Skill,          │
│  │   ClaudeSettings, Member, Team                   │
│  └─ Run (개인 전용): Run (+ Messages, Notes)         │
│                                                     │
│  REST API                                           │
│  ├─ /api/v1/orgs/:owner/claude-mds/...              │
│  ├─ /api/v1/orgs/:owner/members/...                 │
│  ├─ /api/v1/orgs/:owner/teams/...                   │
│  ├─ /api/v1/runs/...                                │
│  └─ (+ fork, version)                               │
│                                                     │
│  SPA (UI)                                           │
└────────────────────┬────────────────────────────────┘
                     │ REST API
┌────────────────────▼────────────────────────────────┐
│                    clier CLI                         │
│                                                     │
│  역할: 서버를 DB로 사용하는 런타임 도구               │
│                                                     │
│  HTTP Client — 서버 API 호출 (CRUD)                  │
│  Workspace Writer — 로컬 파일 생성                   │
│  Runner — RunPlan 생성 + tmux 실행                   │
│                                                     │
│  로컬 DB 없음. 로컬에 남는 것:                        │
│  workspace 파일 + .clier/{RUN_ID}.json               │
└─────────────────────────────────────────────────────┘
```

### CLI의 역할

| CLI가 하는 것 | CLI가 안 하는 것 |
|---|---|
| 서버 API 호출 (CRUD) | DB 저장/조회 |
| workspace 파일 생성 | 엔티티 상태 관리 |
| RunPlan 생성 + tmux 실행 | 비즈니스 로직 |
| | 버전/fork 관리 |
| env vars export + command 실행 | UI 서빙 |

### 엔티티 분류: 공유 가능 vs 개인 전용

서버 엔티티는 소유권/공유 여부에 따라 두 가지 패턴으로 나뉜다.

| 엔티티 | 마켓플레이스 공유 | Visibility | Fork | Version |
|---|---|---|---|---|
| ClaudeMd | Yes | Yes | Yes | Yes |
| Skill | Yes | Yes | Yes | Yes |
| ClaudeSettings | Yes | Yes | Yes | Yes |
| Member | Yes | Yes | Yes | Yes |
| Team | Yes | Yes | Yes | Yes |
| **Run** | **No — 항상 개인 소유** | **No** | **No** | **No** |

### Workspace Domain — 서버 소유, 공유 가능

#### 서버 엔티티 공통 패턴 (공유 가능 리소스)

마켓플레이스에서 공유되는 모든 리소스 엔티티는 동일한 패턴을 따른다.
Run은 이 패턴을 따르지 않는다 (개인 전용, 별도 스키마).

```go
// 공유 가능 리소스 공통 필드
ID            int64       // BIGSERIAL PRIMARY KEY
OwnerID       int64       // FK → users
Name          string
LowerName     string      // UNIQUE with OwnerID
Visibility    Visibility  // 0=Public, 1=Private
IsFork        bool
ForkID        *int64
ForkCount     int
LatestVersion *int
CreatedAt     time.Time
UpdatedAt     time.Time
```

```go
// 버전 엔티티 공통 패턴
ID         int64
ResourceID int64
Version    int
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
- settings.json의 model/permissions와 겹칠 수 있으며, CLI가 최종 우선순위

바이너리 감지 (CLI에서 runtime 결정에 사용):

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
    ID               int64          `json:"id"`
    OwnerID          int64          `json:"owner_id"`
    Name             string         `json:"name"`
    RootTeamMemberID *int64         `json:"root_team_member_id,omitempty"`
    TeamMembers      []TeamMember   `json:"team_members"`
    Relations        []TeamRelation `json:"relations"`
    Visibility       Visibility     `json:"visibility"`
    IsFork           bool           `json:"is_fork"`
    ForkID           *int64         `json:"fork_id,omitempty"`
    ForkCount        int            `json:"fork_count"`
    LatestVersion    *int           `json:"latest_version,omitempty"`
    CreatedAt        time.Time      `json:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at"`
}

type TeamMember struct {
    ID       int64  `json:"id"`
    TeamID   int64  `json:"team_id"`
    MemberID int64  `json:"member_id"`
    Name     string `json:"name"`
}

type TeamRelation struct {
    TeamID           int64 `json:"team_id"`
    FromTeamMemberID int64 `json:"from_team_member_id"`
    ToTeamMemberID   int64 `json:"to_team_member_id"`
}
```

#### Fork + Ref 공유 모델

```
Team (FORK - 복사해서 소유, 구조 수정 가능)
├── TeamMembers[] → Member (REF - FK 참조)
│                      ├── ClaudeMdID (REF)
│                      ├── SkillIDs (REF)
│                      └── ClaudeSettingsID (REF)
└── Relations[]
```

### Run Domain — 서버 소유, 개인 전용, CLI가 실행

#### Run (서버, 개인 전용 — 실행 기록)

Run은 서버가 소유하지만 **항상 개인 소유**이다.
마켓플레이스에 공유되지 않으며, Visibility/Fork/Version이 없다.
Run은 실행 기록(상태, 메시지, 노트)만 담당. 실행 계획은 `.clier/{RUN_ID}.json`에 저장.

```go
type Run struct {
    ID        string      `json:"id"`         // UUID
    UserID    int64       `json:"user_id"`    // 소유자 (항상 본인)
    Name      string      `json:"name"`
    TeamID    *int64      `json:"team_id,omitempty"`
    MemberID  *int64      `json:"member_id,omitempty"`
    Status    RunStatus   `json:"status"`     // "running" | "stopped"
    Messages  []Message   `json:"messages"`   // 에이전트 간 통신 기록
    Notes     []Note      `json:"notes"`      // 진행 상황 기록
    StartedAt time.Time   `json:"started_at"`
    StoppedAt *time.Time  `json:"stopped_at,omitempty"`
    // Visibility 없음, Fork 없음, Version 없음
}

type RunStatus string

const (
    RunRunning RunStatus = "running"
    RunStopped RunStatus = "stopped"
)

// Message — 에이전트 간 통신 기록 (Run 하위)
type Message struct {
    ID               string    `json:"id"`
    FromTeamMemberID string    `json:"from_team_member_id"`
    ToTeamMemberID   string    `json:"to_team_member_id"`
    Content          string    `json:"content"`
    CreatedAt        time.Time `json:"created_at"`
}

// Note — 에이전트 진행 상황 기록 (Run 하위)
type Note struct {
    ID           string    `json:"id"`
    TeamMemberID string    `json:"team_member_id"`
    Content      string    `json:"content"`
    CreatedAt    time.Time `json:"created_at"`
}
```

#### `.clier/{RUN_ID}.json` — 실행 계획 (로컬 파일)

run 시 CLI가 로컬에 생성하는 실행 계획. 사용자가 열어보면 tmux 세션 구조, 각 멤버에 전송되는 command, env vars가 전부 보인다.

**Member 단독 실행 예시:**

```json
{
  "run_id": "abc123",
  "member": "jakeraft/tutorial",
  "created_at": "2026-04-08T15:30:00Z",
  "terminal": {
    "session": "tutorial-abc12345",
    "members": [
      {
        "team_member_id": 42,
        "name": "tutorial",
        "window": 0,
        "cwd": "/path/to/jakeraft/tutorial/project",
        "command": "export CLAUDE_CONFIG_DIR='/path/.claude' && export CLIER_RUN_ID='abc123' && export CLIER_MEMBER_ID='42' && export CLAUDE_CODE_OAUTH_TOKEN='...' && export GIT_AUTHOR_NAME='tutorial' && export GIT_AUTHOR_EMAIL='noreply@clier.com' && export GIT_COMMITTER_NAME='tutorial' && export GIT_COMMITTER_EMAIL='noreply@clier.com' && cd '/path/to/project' && claude '--dangerously-skip-permissions' --model 'opus-4'"
      }
    ]
  }
}
```

**Team 실행 예시:**

```json
{
  "run_id": "def456",
  "team": "jakeraft/dev-squad",
  "created_at": "2026-04-08T15:30:00Z",
  "terminal": {
    "session": "dev-squad-def45678",
    "members": [
      {
        "team_member_id": 1,
        "name": "leader",
        "window": 0,
        "cwd": "/path/to/jakeraft/dev-squad/leader/project",
        "command": "export CLIER_RUN_ID='def456' && export CLIER_TEAM_ID='10' && export CLIER_MEMBER_ID='1' && export CLAUDE_CONFIG_DIR='/path/leader/.claude' && export CLAUDE_CODE_OAUTH_TOKEN='...' && export GIT_AUTHOR_NAME='dev-squad/leader' && export GIT_AUTHOR_EMAIL='noreply@clier.com' && export GIT_COMMITTER_NAME='dev-squad/leader' && export GIT_COMMITTER_EMAIL='noreply@clier.com' && cd '/path/leader/project' && claude --model 'opus-4'"
      },
      {
        "team_member_id": 2,
        "name": "worker-a",
        "window": 1,
        "cwd": "/path/to/jakeraft/dev-squad/worker-a/project",
        "command": "export CLIER_RUN_ID='def456' && export CLIER_TEAM_ID='10' && export CLIER_MEMBER_ID='2' && export CLAUDE_CONFIG_DIR='/path/worker-a/.claude' && export CLAUDE_CODE_OAUTH_TOKEN='...' && export GIT_AUTHOR_NAME='dev-squad/worker-a' && export GIT_AUTHOR_EMAIL='noreply@clier.com' && export GIT_COMMITTER_NAME='dev-squad/worker-a' && export GIT_COMMITTER_EMAIL='noreply@clier.com' && cd '/path/worker-a/project' && codex --full-auto"
      }
    ]
  }
}
```

사용자가 `.clier/` 디렉토리를 열면:
- 매 실행마다 `{RUN_ID}.json` 파일 생성
- tmux 세션명, 윈도우 번호, 각 멤버에 전송되는 전체 command 확인 가능
- 실행 이력이 파일로 남음

#### RunPlan이 단일 진실 공급원 (Single Source of Truth)

RunPlan에 모든 실행 정보가 포함되므로, 에이전트에게 필요한 env var은 2개뿐:

| Env Var | 용도 |
|---|---|
| `CLIER_RUN_PLAN` | RunPlan 파일 경로 (나머지 전부 여기서 조회) |
| `CLIER_MEMBER_ID` | 자신의 멤버 식별 (RunPlan에 여러 멤버가 있으므로) |

나머지(run_id, team_id, session, window, 다른 env vars)는 전부 RunPlan 파일 안에 있다.
별도 env var 목록 불필요.

#### `tell`과 `note`의 흐름

```
clier run tell --to worker-a <<'EOF'
  이 파일 리뷰해줘
EOF

1. CLIER_RUN_PLAN, CLIER_MEMBER_ID 읽기 (env)
2. RunPlan 파일 읽기 → run_id, session, members 확인
3. "worker-a" → MemberTerminal 찾기 → session + window
4. tmux send-keys → 해당 window에 메시지 전달
5. POST /api/v1/runs/{run_id}/messages → 서버에 기록
```

```
clier run note <<'EOF'
  리뷰 완료
EOF

1. CLIER_RUN_PLAN, CLIER_MEMBER_ID 읽기 (env)
2. RunPlan 파일 읽기 → run_id 확인
3. POST /api/v1/runs/{run_id}/notes → 서버에 기록
```

#### RunPlan 사용 매트릭스

| CLI 명령어 | RunPlan | 서버 API |
|---|---|---|
| `clier member run` | **생성** | POST /runs |
| `clier run attach` | **읽기** (session 찾기) | - |
| `clier run tell` | **읽기** (대상 window → send-keys) | POST /runs/{id}/messages |
| `clier run note` | **읽기** (run_id 확인) | POST /runs/{id}/notes |
| `clier run stop` | **읽기** (session → kill) | PATCH /runs/{id} |
| `clier run list` | - | GET /runs |
| `clier run logs` | - | GET /runs/{id} |

### Unified Workspace Layout

`clier member workspace`와 `clier member run`이 동일한 디렉토리 구조를 사용한다.
Team Protocol도 workspace 생성 시 포함된다. Run 시 `.clier/{RUN_ID}.json`만 추가.

**Member 단독:**

```
jakeraft/tutorial/
├── .clier/
│   └── abc123.json                ← 실행 계획 (run 시 생성)
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/
│       └── {name}/SKILL.md        ← Skill
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
```

**Team:**

```
jakeraft/dev-squad/
├── .clier/
│   └── def456.json                ← 실행 계획 (run 시 생성)
├── leader/
│   ├── CLAUDE.md                  ← Team Protocol (workspace 생성 시 포함)
│   ├── .claude/
│   │   ├── settings.json
│   │   └── skills/...
│   └── project/
│       ├── CLAUDE.md              ← ClaudeMd
│       └── (git repo)
├── worker-a/
│   ├── CLAUDE.md                  ← Team Protocol
│   ├── .claude/...
│   └── project/...
└── worker-b/
    └── ...
```

Team Protocol은 각 멤버의 부모 디렉토리 CLAUDE.md로 배치.
Claude Code가 cwd(`project/`)에서 부모로 올라가며 자동 로드 (공식 문서 확인).

### CLI 명령어 구조

```bash
# Workspace: 서버에서 스펙 관리
clier member create --name react-reviewer --command "claude" ...
clier member list
clier team create --name dev-squad ...
clier team list

# Workspace: 작업공간만 생성 (download only)
clier member workspace jakeraft/tutorial
clier team workspace jakeraft/dev-squad

# Run: workspace 생성(멱등) + 실행
clier member run jakeraft/tutorial
clier team run jakeraft/dev-squad
# 1. workspace 없으면 생성 (멱등)
# 2. 서버에 Run 생성 (POST /api/v1/runs → run ID 발급)
# 3. .clier/{RUN_ID}.json 생성 (tmux 구조, command, env vars 포함)
# 4. tmux 세션 생성 + 각 window에 command send-keys
# 5. 종료 시 서버 Run 상태 업데이트 (PATCH)

# Run: 실행 추적/관리 (서버 API)
clier run list
clier run stop --id xxx
clier run logs --id xxx
clier run attach --id xxx
clier run tell --to member-name <<'EOF'    # message 전송
  message content
EOF
clier run note <<'EOF'                     # note 기록
  progress note
EOF

# Fork
clier fork member jake/react-reviewer
clier member update myname/react-reviewer ...
```

### 시스템 주입 정리

| 주입 | 시점 | 방식 | 확인 방법 |
|---|---|---|---|
| Team Protocol | workspace 생성 시 | 부모 디렉토리 CLAUDE.md 파일 | 파일 직접 확인 |
| RunPlan | run 시 | `.clier/{RUN_ID}.json` (tmux 구조, command, env 전부 포함) | `.clier/` 디렉토리 확인 |
| `CLIER_RUN_PLAN` | run 시 | env var (RunPlan 파일 경로) | RunPlan의 command에 포함 |
| `CLIER_MEMBER_ID` | run 시 | env var (자신의 멤버 ID) | RunPlan의 command에 포함 |
| ~~.claude.json~~ | ~~삭제~~ | - | - |
| ~~CLAUDE.md 머지~~ | ~~삭제~~ | - | - |
| ~~.claude.json 머지~~ | ~~삭제~~ | - | - |
| ~~별도 env var 목록~~ | ~~삭제~~ | RunPlan.command에 전부 포함 | - |

## 현재 대비 변경 요약

### 아키텍처 변경

| 항목 | 현재 | 변경 |
|---|---|---|
| 상태 저장 | CLI 로컬 SQLite (모든 엔티티) | 서버 PostgreSQL (모든 엔티티) |
| CLI 역할 | DB + 비즈니스 로직 + 터미널 + UI | 서버를 DB로 사용하는 런타임 도구 |
| CLI 로컬 파일 | SQLite DB | workspace 파일 + `.clier/{RUN_ID}.json` |
| 실행 계획 | MemberPlan (Task 하위, env/tmux 안 보임) | `.clier/{RUN_ID}.json` (tmux 세션, command, env 전부 보임) |
| 실행 기록 | Task에 Plan + Messages + Notes 혼재 | Run = 기록만 (Messages, Notes), 실행 계획 = 로컬 파일 |
| Team Protocol | Run 시 CLAUDE.md에 머지 | Workspace 생성 시 별도 파일 |
| CLI 명령어 | `task create` + `task start` | `member run` / `team run` |
| 용어 | Task (혼재) | Run (통일) |

### 엔티티 변경

| 항목 | 변경 |
|---|---|
| `Task` | → `Run` 리네임 |
| `TaskStatus` | → `RunStatus` |
| `Message.TaskID` | → `Message.RunID` |
| `Note.TaskID` | → `Note.RunID` |
| `AgentDotMd` | → `ClaudeMd` 리네임 |
| `ClaudeJson` | 삭제 |
| `Member.AgentType` | 삭제 (`Command`에서 감지) |
| `Member.Model` | 삭제 (`ClaudeSettings` → `settings.json`의 `"model"`) |
| `Member.Args` | 삭제 (`Command`에 통합) |
| `Member.Command` | 신규 (바이너리 + CLI flags) |
| `Run.MemberID` | 신규 (Member 단독 실행 지원) |
| `Run.Plan` | 삭제 (`.clier/{RUN_ID}.json`으로 이동) |
| `Run.Messages` | 유지 (Run 하위 필드) |
| `Run.Notes` | 유지 (Run 하위 필드) |
| `CLIER_TASK_ID` env | → `CLIER_RUN_ID` |

### CLI 파일 변경 (예상)

| 파일 | 변경 |
|---|---|
| `internal/adapter/db/` | **전체 삭제** |
| `internal/domain/` | 서버 스키마 Go struct (API 파싱용) |
| `internal/domain/task.go` | → `run.go` 리네임, Task → Run |
| `internal/adapter/api/` | **신규** — HTTP 클라이언트 |
| `internal/app/workspace/` | **신규** — workspace 파일 생성 |
| `internal/app/run/` | **신규** — Runner (RunPlan 생성 + tmux 실행, 기존 terminal/ 통합) |
| `cmd/task*.go` | → `cmd/run*.go` 리네임 |

### 서버 추가 필요 (clier-server, 별도 작업)

| 항목 | 내용 |
|---|---|
| `AgentDotMd` 리네임 | → `ClaudeMd` |
| `Skill` 추가 | 빌딩블록 (동일 패턴) |
| `ClaudeSettings` 추가 | 빌딩블록 (동일 패턴) |
| `Member` 추가 | 스펙 (FK로 빌딩블록 참조) |
| `Team` 추가 | 스펙 (FK로 멤버 참조 + 관계 테이블) |
| `Run` 추가 | 개인 전용 (UserID, Messages/Notes 하위 필드) |

## Scope

### In scope

- Workspace / Run 도메인 분리
- Task → Run 용어 통일
- CLI에서 로컬 DB 완전 제거
- Building Block 정비 (ClaudeJson 삭제, AgentDotMd → ClaudeMd)
- Member 엔티티 리팩토링 (AgentType/Model/Args 삭제, Command 추가)
- Run을 서버 소유 + 개인 전용 + 실행 기록 전용으로 전환
- `.clier/{RUN_ID}.json` 실행 계획 파일
- HTTP 클라이언트 어댑터 추가
- 통합 Workspace 레이아웃
- Team Protocol을 workspace 생성 시 포함
- CLI 명령어: `member run` / `team run` / `member workspace` / `team workspace`
- CLI 명령어: `run list` / `run stop` / `run logs` / `run attach` / `run tell` / `run note`
- 머지 로직 전부 제거
- `CLIER_TASK_ID` → `CLIER_RUN_ID`

### Out of scope

- UI (서버에서 구현)
- 서버 엔티티/API 구현 (별도 작업)
- 마켓플레이스 검색/필터링 (별도 작업)
- SQLite → 서버 마이그레이션 경로 (별도 작업)
- Dashboard 커맨드 전환 (서버 UI로 대체 예정)
- Import/Export 커맨드 재설계 (서버 API 기반으로 별도 작업)

### 삭제 대상 (기존 코드)

| 대상 | 이유 |
|---|---|
| `internal/adapter/db/` 전체 | 로컬 DB 제거 |
| `internal/domain/resource/claudejson.go` | ClaudeJson 삭제 |
| `terminal_refs` 테이블 | `.clier/{RUN_ID}.json`이 tmux 세션/윈도우 정보를 로컬에 저장하므로 대체됨 |
| `RefStore` 인터페이스 | terminal_refs 삭제에 따라 불필요 |
| `workspace_files.go`의 머지 로직 | CLAUDE.md 머지, .claude.json deep merge 제거 |
| `MemberPlan` (Task 하위) | `.clier/{RUN_ID}.json`으로 이동 |
