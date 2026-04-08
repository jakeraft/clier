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

## Design

### Two-Domain Architecture

```
Setting Domain (Marketplace)          Run Domain (Local Orchestration)
─────────────────────────────         ──────────────────────────────────
파일 세팅, 다운로드, 공유              실행 오케스트레이션, 프로토콜 주입

BuildingBlock (3종)                   Task
Member (단독 실행 가능 스펙)             ├─ MemberBinding (런타임 override)
Team (멤버 조합 + 관계)                 └─ MemberPlan (투명한 실행 명세)
```

**Setting = "이 에이전트가 뭔지" (파일로 표현)**
**Run = "어디서 어떻게 돌릴지" (오케스트레이션)**

### Setting Domain

#### Building Blocks (3종)

`ClaudeJson` 삭제. `.claude.json`은 Claude Code의 user-level 런타임 상태 파일이며
프로젝트 레벨이 존재하지 않는다 (공식 문서 확인). 사용자가 필요한 설정은 전부 `settings.json`으로 커버 가능.

| Building Block | 생성 파일 | 담당 |
|---|---|---|
| **ClaudeMd** | `CLAUDE.md` | 에이전트 지시사항 |
| **Skill** | `.claude/skills/{name}/SKILL.md` | 에이전트 스킬 |
| **ClaudeSettings** | `.claude/settings.json` | model, permissions, hooks, env |

각 Building Block은 서버에서 개별 공유 가능하며, 버전 관리와 fork를 지원한다.

```go
type ClaudeMd struct {
    ID        string
    Name      string
    Content   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Skill struct {
    ID        string
    Name      string    // lowercase + hyphens (^[a-z0-9]+(-[a-z0-9]+)*$)
    Content   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type ClaudeSettings struct {
    ID        string
    Name      string
    Content   string    // valid JSON
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

서버에서 참조 시:

```go
type ResourceRef struct {
    Owner   string  // "jake"
    Name    string  // "react-prompt"
    Version int     // 3
}
```

#### Member

단독 실행 가능한 에이전트 스펙. 다운로드하면 바로 사용할 수 있는 완전한 패키지.

```go
type Member struct {
    ID                string
    Name              string          // "react-code-reviewer"
    Command           string          // "claude --dangerously-skip-permissions"
    GitRepoURL        string          // "https://github.com/jake/react-app"
    ClaudeMdID        string          // FK → ClaudeMd (로컬)
    SkillIDs          []string        // FK → Skill[] (로컬)
    ClaudeSettingsID  string          // FK → ClaudeSettings (로컬)
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

변경점:
- `AgentType` 삭제: `Command`의 첫 단어(바이너리명)에서 감지
- `Model` 삭제: `ClaudeSettings`의 `settings.json` → `"model"` 필드로 이동
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

서버에서 참조 시 (마켓플레이스):

```go
type MemberServer struct {
    // ... 서버 공통 필드 (owner, visibility, fork_info, versions)
    Name              string
    Command           string
    GitRepoURL        string
    ClaudeMdRef       ResourceRef     // owner/name@version
    SkillRefs         []ResourceRef
    ClaudeSettingsRef ResourceRef
}
```

#### Team

멤버 조합과 관계를 정의하는 스펙.

```go
type Team struct {
    ID               string
    Name             string
    RootTeamMemberID string
    TeamMembers      []TeamMember
    Relations        []Relation
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type TeamMember struct {
    ID       string
    MemberID string   // FK → Member (로컬) 또는 ResourceRef (서버)
    Name     string   // 팀 내 역할명
}

type Relation struct {
    From string   // TeamMember ID (리더)
    To   string   // TeamMember ID (워커)
}
```

#### Fork + Ref 공유 모델

Terraform 모듈 레지스트리 벤치마킹.

```
Team (FORK - 복사해서 소유, 구조 수정 가능)
├── TeamMembers[] → Member (REF - 서버 원본 참조)
│                      ├── ClaudeMdRef (REF)
│                      ├── SkillRefs[] (REF)
│                      └── ClaudeSettingsRef (REF)
└── Relations[]
```

- **Fork**: 서버에 내 계정 복사본 생성. 수정 가능, 독립적.
  - Member, Team 레벨에서 fork
- **Ref**: 서버 원본을 `owner/name@version`으로 참조.
  - Building Blocks는 ref. 수정이 필요하면 그때 fork.
- 현재 서버에 fork 인프라 구현 완료 (`is_fork`, `fork_id`, `fork_count`, versions)

#### 다운로드 결과물

`clier download member`:

```
{target-dir}/
├── CLAUDE.md                      ← ClaudeMd
├── .claude/
│   ├── settings.json              ← ClaudeSettings (model, permissions 포함)
│   └── skills/
│       └── {name}/SKILL.md        ← Skill
└── (사용자가 git clone, 사용자가 직접 에이전트 실행)
```

`clier download team`:

```
{target-dir}/
├── leader/
│   ├── CLAUDE.md
│   └── .claude/...
├── worker-a/
│   ├── CLAUDE.md
│   └── .claude/...
└── worker-b/
    ├── CLAUDE.md
    └── .claude/...
```

시스템 주입 없음. 전부 사용자 파일. 사용자가 자기 공간을 완전히 컨트롤.

### Run Domain

#### Task

Member 단독 실행 또는 Team 실행을 지원하는 오케스트레이션 단위.

```go
type Task struct {
    ID        string
    Name      string
    TeamID    string              // Team 실행 시 (MemberID와 배타적)
    MemberID  string              // Member 단독 실행 시 (TeamID와 배타적)
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
    TeamMemberID string   // Team 실행 시 slot 지정
    GitRepoURL   string   // repo override (빈값 = Member 스펙 사용)
}
```

#### MemberPlan

실행의 완전한 명세. 숨겨진 주입 없음. Plan에 없으면 주입 안 됨.

```go
type MemberPlan struct {
    TeamMemberID string
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

Team 실행 시에만 주입. 부모 디렉토리 CLAUDE.md로 배치하여 Claude Code가 자동 로드.
사용자의 ClaudeMd(`project/CLAUDE.md`)와 머지 없이 분리.

```
Claude Code 로딩 순서 (공식 문서 확인):
1. project/CLAUDE.md (유저 ClaudeMd) — 로드
2. 부모 디렉토리로 올라감 → {memberspace}/CLAUDE.md (Team Protocol) — 로드
3. 둘 다 concatenate됨 (override 아님)
```

#### Run 시 Workspace Layout

**Member 단독 실행:**

```
{memberspace}/
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/{name}/SKILL.md     ← Skills
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
```

**Team 실행 (각 멤버별):**

```
{memberspace}/
├── CLAUDE.md                      ← Team Protocol (시스템, 유일한 파일 주입)
├── .claude/
│   ├── settings.json              ← ClaudeSettings
│   └── skills/{name}/SKILL.md     ← Skills
└── project/                       ← cwd
    ├── CLAUDE.md                  ← ClaudeMd
    └── (git repo)
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
| `AgentDotMd` | → `ClaudeMd` 리네임 |
| `ClaudeJson` | 삭제 |
| `Member.AgentType` | 삭제 (`Command`에서 감지) |
| `Member.Model` | 삭제 (`ClaudeSettings` → `settings.json`의 `"model"`) |
| `Member.Args` | 삭제 (`Command`에 통합) |
| `Member.Command` | 신규 (바이너리 + CLI flags) |
| `Task.MemberID` | 신규 (Member 단독 실행 지원) |
| `Task.Bindings` | 신규 (런타임 override) |
| `MemberPlan.Env` | 신규 (환경변수 명시) |
| `MemberPlan.Protocol` | 신규 (Team Protocol 명시) |

### 파일 변경 (예상)

| 파일 | 변경 |
|---|---|
| `internal/domain/resource/agent_dot_md.go` | → `claude_md.go` 리네임 |
| `internal/domain/resource/claudejson.go` | 삭제 |
| `internal/domain/member.go` | AgentType, Model, Args 제거, Command 추가 |
| `internal/domain/task.go` | MemberID, Bindings 추가, MemberPlan에 Env/Protocol 추가 |
| `internal/app/task/plan.go` | 빌드 로직 수정 (머지 제거, Env/Protocol 명시적 생성) |
| `internal/app/task/prompt.go` | Protocol을 별도 파일로 출력 (부모 디렉토리) |
| `internal/app/task/workspace_files.go` | 머지 로직 제거, 파일 스코핑 적용 |
| `internal/app/task/command.go` | detectRuntime 기반으로 리팩토링 |
| `internal/adapter/runtime/` | AgentRuntime 인터페이스 단순화 |
| `internal/adapter/db/schema.sql` | members 테이블 변경, claudejson 관련 삭제 |
| `internal/adapter/db/member.go` | 쿼리 수정 |
| `internal/adapter/db/claudejson.go` | 삭제 |

### 서버 추가 필요 (clier-server)

| 항목 | 내용 |
|---|---|
| `AgentDotMd` 리네임 | → `ClaudeMd` |
| `Skill` 추가 | 빌딩블록 (버전 관리 + fork) |
| `ClaudeSettings` 추가 | 빌딩블록 (버전 관리 + fork) |
| `Member` 추가 | 스펙 (ResourceRef로 빌딩블록 참조, 버전 관리 + fork) |
| `Team` 추가 | 스펙 (ResourceRef로 멤버 참조, 버전 관리 + fork) |

### 사용 흐름

```bash
# Setting: 마켓에서 다운로드
clier download member jake/react-reviewer@v3
clier download team jake/dev-squad@v1

# Setting: fork 후 커스터마이징
clier fork member jake/react-reviewer
clier member update myname/react-reviewer --command "claude --max-turns 100"

# Run: 단독 실행
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
- Building Block 정비 (ClaudeJson 삭제, AgentDotMd → ClaudeMd)
- Member 엔티티 리팩토링 (AgentType/Model/Args 삭제, Command 추가)
- Task 엔티티 확장 (MemberID, Bindings, Plan 투명화)
- Workspace 레이아웃 변경 (머지 제거, 스코프 분리)
- DB 스키마 마이그레이션

### Out of scope

- UI 변경 (서버에서 구현 예정)
- 서버 엔티티 추가 (별도 작업)
- download / fork CLI 커맨드 구현 (별도 작업)
- 마켓플레이스 검색/필터링 기능 (별도 작업)
