# CLI Domain Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** clier CLI를 DB 없는 경량 런타임 도구로 전환. 도메인 엔티티 정비, 로컬 DB 제거, HTTP 클라이언트 추가, Workspace/Run 구조 재설계.

**Architecture:** 3-phase 전환 — Phase 1: 도메인 정비 (엔티티 리네임/삭제/필드 변경), Phase 2: 인프라 전환 (DB 제거 + HTTP 클라이언트), Phase 3: Workspace/Run 재설계.

**Tech Stack:** Go 1.25, Cobra, tmux

**Working Directory:** `/Users/jake_kakao/jakeraft/clier`

**Design Spec:** `docs/superpowers/specs/2026-04-08-setting-run-domain-separation-design.md`

---

## Phase 1: 도메인 엔티티 정비

### Task 1: AgentDotMd → ClaudeMd 리네임

**Files:**
- Create: `internal/domain/resource/claude_md.go`
- Delete: `internal/domain/resource/agent_dot_md.go`
- Modify: `internal/domain/member.go` (AgentDotMdID → ClaudeMdID)
- Modify: all files referencing AgentDotMd

- [ ] **Step 1: 새 파일 생성**

`internal/domain/resource/claude_md.go`:

```go
package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
		n := strings.TrimSpace(*name)
		if n == "" {
			return errors.New("claude md name must not be empty")
		}
		c.Name = n
	}
	if content != nil {
		ct := strings.TrimSpace(*content)
		if ct == "" {
			return errors.New("claude md content must not be empty")
		}
		c.Content = ct
	}
	c.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 2: member.go에서 AgentDotMdID → ClaudeMdID 변경**

```go
// Before
AgentDotMdID string `json:"agent_dot_md_id"`

// After
ClaudeMdID string `json:"claude_md_id"`
```

- [ ] **Step 3: 전체 코드베이스에서 AgentDotMd → ClaudeMd 치환**

```bash
# 참조 확인
grep -r "AgentDotMd\|agent_dot_md" --include="*.go" -l
# 각 파일에서 치환
```

영향 받는 파일: `member.go` (ResolvedMember), `plan.go` (resolveMember), `workspace_files.go`, DB store, adapter 등.

- [ ] **Step 4: 이전 파일 삭제 및 빌드 확인**

```bash
rm internal/domain/resource/agent_dot_md.go
go build ./...
go test ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A && git commit -m "refactor: AgentDotMd → ClaudeMd 리네임

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: ClaudeJson 삭제

**Files:**
- Delete: `internal/domain/resource/claudejson.go`
- Modify: `internal/domain/member.go` (ClaudeJsonID 필드 제거)
- Modify: `internal/adapter/db/schema.sql` (claude_jsons 테이블, members FK 제거)
- Modify: all files referencing ClaudeJson

- [ ] **Step 1: Member에서 ClaudeJsonID 제거**

`internal/domain/member.go`:
```go
// 삭제할 필드
ClaudeJsonID string `json:"claude_json_id"`
```

`NewMember()`, `Update()` 함수에서도 claudeJsonID 파라미터 제거.

- [ ] **Step 2: ResolvedMember에서 ClaudeJson 제거**

```go
// 삭제할 필드
ClaudeJson *resource.ClaudeJson
```

- [ ] **Step 3: 전체 참조 제거**

```bash
grep -r "ClaudeJson\|claude_json\|claudejson" --include="*.go" -l
```

영향: `plan.go` (resolveMember), `workspace_files.go` (buildWorkspaceFiles의 mergeJSON), `schema.sql`, DB store, service.

- [ ] **Step 4: 파일 삭제 및 빌드 확인**

```bash
rm internal/domain/resource/claudejson.go
go build ./...
go test ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A && git commit -m "refactor: ClaudeJson 빌딩블록 삭제

.claude.json은 Claude Code의 user-level 런타임 상태 파일이며
프로젝트 레벨이 존재하지 않음. settings.json으로 대체.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Member 필드 정비

**Files:**
- Modify: `internal/domain/member.go`

- [ ] **Step 1: AgentType, Model, Args 삭제 + Command 추가**

```go
// Before
type Member struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	AgentType        string    `json:"agent_type"`
	Model            string    `json:"model"`
	Args             []string  `json:"args"`
	ClaudeMdID       string    `json:"claude_md_id"`
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`
	GitRepoURL       string    `json:"git_repo_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// After
type Member struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Command          string    `json:"command"`
	GitRepoURL       string    `json:"git_repo_url"`
	ClaudeMdID       string    `json:"claude_md_id"`
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: NewMember, Update 시그니처 변경**

`NewMember(name, command, gitRepoURL, claudeMdID string, skillIDs []string, claudeSettingsID string)`

- [ ] **Step 3: ResolvedMember에서 AgentType, Model, Args 제거**

```go
// Before
type ResolvedMember struct {
	TeamMemberID string
	Name         string
	AgentType    string
	Model        string
	Args         []string
	// ...
}

// After
type ResolvedMember struct {
	TeamMemberID string
	Name         string
	Command      string
	// ...
}
```

- [ ] **Step 4: 전체 참조 업데이트**

```bash
grep -r "AgentType\|\.Model\b\|\.Args\b" --include="*.go" -l
```

영향: `plan.go`, `command.go`, `prompt.go`, DB store, service, CLI commands.

- [ ] **Step 5: 빌드 확인 및 커밋**

```bash
go build ./...
go test ./...
git add -A && git commit -m "refactor: Member에서 AgentType/Model/Args 삭제, Command 추가

- AgentType: Command 첫 단어에서 바이너리 감지
- Model: ClaudeSettings(settings.json)의 model 필드로 이동
- Args: Command에 통합

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Task → Run 리네임

**Files:**
- Rename: `internal/domain/task.go` → `internal/domain/run.go`
- Modify: all files referencing Task/TaskStatus/TaskRunning/TaskStopped
- Rename: `cmd/task.go` → `cmd/run.go`

- [ ] **Step 1: domain/task.go → domain/run.go**

```go
// Before
type TaskStatus string
const (
	TaskRunning TaskStatus = "running"
	TaskStopped TaskStatus = "stopped"
)
type Task struct { ... }

// After
type RunStatus string
const (
	RunRunning RunStatus = "running"
	RunStopped RunStatus = "stopped"
)
type Run struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	TeamID    string     `json:"team_id,omitempty"`
	MemberID  string     `json:"member_id,omitempty"`
	Status    RunStatus  `json:"status"`
	Messages  []Message  `json:"messages"`
	Notes     []Note     `json:"notes"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
}

type Message struct {
	ID               string    `json:"id"`
	FromTeamMemberID string    `json:"from_team_member_id"`
	ToTeamMemberID   string    `json:"to_team_member_id"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

type Note struct {
	ID           string    `json:"id"`
	TeamMemberID string    `json:"team_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}
```

- [ ] **Step 2: MemberPlan에서 Plan 분리 — Run 엔티티에서 Plan 제거**

Run은 실행 기록(Messages, Notes, Status)만 보유. Plan은 `.clier/{RUN_ID}.json`으로 이동될 예정이므로 MemberPlan은 별도 도메인 타입으로 유지하되 Run의 하위 필드에서 제거.

- [ ] **Step 3: 전체 코드에서 Task → Run 치환**

```bash
grep -r "Task\|task" --include="*.go" -l | head -30
# Task → Run, TaskStatus → RunStatus, TaskRunning → RunRunning 등
```

영향: domain, app/task/ (→ app/run/), adapter/db, adapter/terminal, cmd/task.go (→ cmd/run.go).

- [ ] **Step 4: cmd/task.go → cmd/run.go 리네임 + 명령어 변경**

```go
// Before
var taskCmd = &cobra.Command{Use: "task", ...}
newTaskStartCmd(), newTaskStopCmd(), newTaskTellCmd(), newTaskNoteCmd()

// After
var runCmd = &cobra.Command{Use: "run", ...}
newRunStopCmd(), newRunTellCmd(), newRunNoteCmd(), newRunListCmd()
// + member run, team run은 Phase 3에서 추가
```

- [ ] **Step 5: CLIER_TASK_ID → CLIER_RUN_PLAN + CLIER_MEMBER_ID로 변경**

`command.go`:
```go
// Before
"CLIER_TASK_ID=" + taskID,
"CLIER_MEMBER_ID=" + memberID,

// After — Phase 3에서 RunPlan으로 전환 시 적용
```

- [ ] **Step 6: root.go에서 agent 모드 필터링 변경**

```go
// Before
if os.Getenv("CLIER_MEMBER_ID") != "" {
	filterAgentCommands()
}

// After
if os.Getenv("CLIER_AGENT") == "true" {
	filterAgentCommands()
}
```

`filterAgentCommands()`에서 "task" → "run" 변경.

- [ ] **Step 7: 빌드 확인 및 커밋**

```bash
go build ./...
go test ./...
git add -A && git commit -m "refactor: Task → Run 리네임

도메인, 커맨드, 환경변수 전체 통일.
CLIER_AGENT env var로 agent 모드 명시.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Phase 2: 숨겨진 주입 제거

### Task 5: Placeholder 시스템 삭제

**Files:**
- Delete: `internal/app/task/expand.go`
- Delete: `internal/app/task/expand_test.go`
- Modify: `internal/app/task/plan.go` (placeholder 상수 제거)
- Modify: `internal/app/task/service.go` (expand phase 제거)

- [ ] **Step 1: plan.go에서 placeholder 상수 제거**

```go
// 삭제
const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderTaskID      = "{{CLIER_TASK_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
)
```

- [ ] **Step 2: buildMemberPlan에서 concrete 경로 사용**

placeholder 대신 실제 경로를 직접 사용하도록 변경. base, taskID 등을 파라미터로 받아 직접 조합.

- [ ] **Step 3: service.go에서 expand loop 제거**

```go
// 삭제 (service.go Start() 내부)
for i := range plans {
	plans[i] = expandPlaceholders(plans[i], base, home, taskID, authToken)
}
```

- [ ] **Step 4: expand.go, expand_test.go 삭제**

```bash
rm internal/app/task/expand.go internal/app/task/expand_test.go
go build ./...
go test ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A && git commit -m "refactor: {{CLIER_*}} placeholder 시스템 삭제

RunPlan이 concrete values를 직접 저장하므로
2-phase 빌드(build → expand) 불필요.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: Auth 주입 삭제

**Files:**
- Modify: `internal/app/task/command.go` (auth env 제거)
- Modify: `internal/app/task/service.go` (AuthChecker 제거)
- Modify: `internal/adapter/runtime/claude.go` (AuthEnvs 제거)

- [ ] **Step 1: command.go에서 auth env 제거**

```go
// Before (buildEnv)
env = append(env, rt.AuthEnvs(authPlaceholder)...)

// After — auth 관련 라인 삭제
```

- [ ] **Step 2: service.go에서 AuthChecker 인터페이스 제거**

```go
// 삭제
type AuthChecker interface {
	ReadToken() (string, error)
}
```

Service struct에서 auth 필드 제거, Start()에서 ReadToken() 호출 제거.

- [ ] **Step 3: AgentRuntime 인터페이스에서 AuthEnvs 제거**

```go
// Before (runtime.go)
AuthEnvs(token string) []string

// After — 메서드 삭제
```

`internal/adapter/runtime/claude.go`에서 AuthEnvs 구현 제거.

- [ ] **Step 4: 빌드 확인 및 커밋**

```bash
go build ./...
go test ./...
git add -A && git commit -m "refactor: auth 주입 삭제

사용자가 workspace에서 직접 인증.
CLAUDE_CODE_OAUTH_TOKEN 주입 제거.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 7: CLAUDE.md 머지 + .claude.json 머지 로직 삭제

**Files:**
- Modify: `internal/app/task/workspace_files.go` (mergeJSON 삭제, 파일 스코핑)

- [ ] **Step 1: mergeJSON, mergeJSONObjects 함수 삭제**

`workspace_files.go`에서 lines 56-114 삭제.

- [ ] **Step 2: buildWorkspaceFiles에서 system+user CLAUDE.md 머지 제거**

```go
// Before
content := systemClaudeMd + "\n\n---\n\n" + userClaudeMd

// After — user ClaudeMd만 project/CLAUDE.md에 저장
// system protocol은 부모 디렉토리 CLAUDE.md로 분리 (workspace 생성 시)
```

- [ ] **Step 3: .claude.json 머지 제거**

system config(onboarding skip)과 user ClaudeJson의 deep merge 로직 삭제. `.claude.json` 자체를 생성하지 않음.

- [ ] **Step 4: 빌드 확인 및 커밋**

```bash
go build ./...
go test ./...
git add -A && git commit -m "refactor: CLAUDE.md 머지 + .claude.json 머지 로직 삭제

- CLAUDE.md: 부모 디렉토리 스코핑으로 분리 (Claude Code walk-up)
- .claude.json: 사용자가 workspace에서 직접 관리

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Phase 3: 인프라 전환 + Workspace/Run 재설계

### Task 8: HTTP 클라이언트 어댑터 추가

**Files:**
- Create: `internal/adapter/api/client.go`
- Create: `internal/adapter/api/member.go`
- Create: `internal/adapter/api/team.go`
- Create: `internal/adapter/api/run.go`

- [ ] **Step 1: HTTP 클라이언트 기반 구조**

`internal/adapter/api/client.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		token:      token,
	}
}

func (c *Client) do(method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(b))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

func (c *Client) get(path string, result any) error    { return c.do("GET", path, nil, result) }
func (c *Client) post(path string, body, result any) error { return c.do("POST", path, body, result) }
func (c *Client) put(path string, body, result any) error  { return c.do("PUT", path, body, result) }
func (c *Client) patch(path string, body, result any) error { return c.do("PATCH", path, body, result) }
func (c *Client) delete(path string) error             { return c.do("DELETE", path, nil, nil) }
```

- [ ] **Step 2: Member API 클라이언트**

`internal/adapter/api/member.go`:

```go
package api

import "github.com/jakeraft/clier/internal/domain"

func (c *Client) GetMember(owner, name string) (*domain.Member, error) {
	var m domain.Member
	err := c.get(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, name), &m)
	return &m, err
}

func (c *Client) ListMembers(owner string) ([]domain.Member, error) {
	var members []domain.Member
	err := c.get(fmt.Sprintf("/api/v1/orgs/%s/members", owner), &members)
	return members, err
}

// ... Create, Update, Delete, Fork
```

- [ ] **Step 3: Team, Run API 클라이언트도 동일 패턴으로 생성**

- [ ] **Step 4: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: HTTP 클라이언트 어댑터 추가

서버 API를 호출하는 경량 클라이언트.
Member, Team, Run CRUD.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 9: 로컬 DB 삭제

**Files:**
- Delete: `internal/adapter/db/` 전체 디렉토리
- Modify: `cmd/root.go` (DB 초기화 제거)
- Modify: `cmd/*.go` (store 의존성 → API client로 전환)

- [ ] **Step 1: cmd에서 DB 초기화 제거, API client 초기화 추가**

```go
// Before (root.go)
store, err := db.NewStore(dbPath)

// After
apiClient := api.NewClient(serverURL, authToken)
```

서버 URL은 환경변수 `CLIER_SERVER_URL` 또는 config에서 읽기.

- [ ] **Step 2: 모든 커맨드에서 store → apiClient 전환**

각 커맨드 파일에서 `store.GetMember()` → `apiClient.GetMember()` 등으로 변경.

- [ ] **Step 3: adapter/db/ 디렉토리 삭제**

```bash
rm -rf internal/adapter/db/
go build ./...
```

- [ ] **Step 4: go.mod에서 sqlite 의존성 제거**

```bash
go mod tidy
```

- [ ] **Step 5: 커밋**

```bash
git add -A && git commit -m "refactor: 로컬 SQLite DB 삭제, HTTP 클라이언트로 전환

서버가 모든 엔티티의 source of truth.
CLI는 DB 없는 경량 런타임 도구.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 10: Workspace Writer

**Files:**
- Create: `internal/app/workspace/writer.go`
- Create: `internal/app/workspace/protocol.go`

- [ ] **Step 1: Workspace Writer**

`internal/app/workspace/writer.go`:

```go
package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
)

type Writer struct {
	client *api.Client
}

func NewWriter(client *api.Client) *Writer {
	return &Writer{client: client}
}

// PrepareMember creates workspace for a single member.
// Layout:
//   {base}/project/CLAUDE.md
//   {base}/project/.claude/settings.json
//   {base}/project/.claude/skills/{name}/SKILL.md
func (w *Writer) PrepareMember(base string, owner, memberName string) error {
	// 1. Fetch member spec from server
	// 2. Fetch referenced building blocks (ClaudeMd, Skills, ClaudeSettings)
	// 3. Write files to workspace
	projectDir := filepath.Join(base, "project")
	// ...
}

// PrepareTeam creates workspace for all team members.
// Each member gets its own subdirectory with protocol CLAUDE.md at parent level.
func (w *Writer) PrepareTeam(base string, owner, teamName string) error {
	// 1. Fetch team spec from server
	// 2. For each team member: PrepareMember + protocol CLAUDE.md
	// ...
}
```

- [ ] **Step 2: Team Protocol 생성 (workspace 시점)**

`internal/app/workspace/protocol.go`:

```go
package workspace

// BuildProtocol generates the team protocol CLAUDE.md content.
// Written to {memberDir}/CLAUDE.md (parent of project/).
func BuildProtocol(teamName, memberName string, leaders, workers []string) string {
	// 기존 prompt.go의 buildClierPrompt 로직을 여기로 이동
	// ...
}
```

- [ ] **Step 3: 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: Workspace Writer 추가

서버에서 스펙 fetch → 로컬 워크스페이스 파일 생성.
Team Protocol도 workspace 생성 시 포함.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 11: Runner (RunPlan + tmux 통합)

**Files:**
- Create: `internal/app/run/runner.go`
- Create: `internal/app/run/plan.go`
- Create: `internal/app/run/detect.go`
- Delete: `internal/app/task/` 디렉토리 (service.go, plan.go 등)
- Modify: `internal/adapter/terminal/tmux.go` (Runner에 통합)

- [ ] **Step 1: RunPlan 도메인 타입**

`internal/app/run/plan.go`:

```go
package run

type RunPlan struct {
	Session string           `json:"session"`
	Members []MemberTerminal `json:"members"`
}

type MemberTerminal struct {
	Name    string `json:"name"`
	Window  int    `json:"window"`
	Cwd     string `json:"cwd"`
	Command string `json:"command"`
}
```

- [ ] **Step 2: Runtime 감지**

`internal/app/run/detect.go`:

```go
package run

import "strings"

type Runtime int

const (
	ClaudeRuntime Runtime = iota
	CodexRuntime
	GenericRuntime
)

func DetectRuntime(command string) Runtime {
	binary := strings.Fields(command)[0]
	switch binary {
	case "claude":
		return ClaudeRuntime
	case "codex":
		return CodexRuntime
	default:
		return GenericRuntime
	}
}

// ConfigDirEnv returns the config dir env var for workspace isolation.
func ConfigDirEnv(rt Runtime, configDir string) string {
	switch rt {
	case ClaudeRuntime:
		return "CLAUDE_CONFIG_DIR=" + configDir
	default:
		return ""
	}
}
```

- [ ] **Step 3: Runner — RunPlan 생성 + tmux 실행 통합**

`internal/app/run/runner.go`:

```go
package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/adapter/api"
)

type Runner struct {
	client *api.Client
	tmux   TmuxAdapter
}

func NewRunner(client *api.Client, tmux TmuxAdapter) *Runner {
	return &Runner{client: client, tmux: tmux}
}

// Run creates a Run on server, builds RunPlan, saves to .clier/, and executes via tmux.
func (r *Runner) Run(workspaceBase, owner, name string, isTeam bool) error {
	// 1. POST /api/v1/runs → get run ID
	// 2. Build RunPlan (session name, members, commands, env vars)
	// 3. Save .clier/{RUN_ID}.json
	// 4. Create tmux session + send-keys to each window
	return nil
}

// BuildRunPlan constructs the execution plan from workspace files.
func (r *Runner) BuildRunPlan(runID, workspaceBase string, members []MemberInfo) *RunPlan {
	sessionName := runName(runID)
	plan := &RunPlan{
		Session: sessionName,
	}
	for i, m := range members {
		planPath := filepath.Join(workspaceBase, ".clier", runID+".json")
		cmd := buildCommand(m, runID, planPath)
		plan.Members = append(plan.Members, MemberTerminal{
			Name:    m.Name,
			Window:  i,
			Cwd:     m.Cwd,
			Command: cmd,
		})
	}
	return plan
}

// SavePlan writes RunPlan to .clier/{RUN_ID}.json
func SavePlan(workspaceBase, runID string, plan *RunPlan) error {
	dir := filepath.Join(workspaceBase, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, runID+".json"))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}
```

- [ ] **Step 4: 기존 app/task/ 디렉토리 삭제**

```bash
rm -rf internal/app/task/
go build ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A && git commit -m "feat: Runner 추가 (RunPlan 생성 + tmux 실행 통합)

- RunPlan: session, members[{name, window, cwd, command}]
- .clier/{RUN_ID}.json으로 저장
- 기존 app/task/ 삭제, terminal adapter 통합

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 12: CLI 명령어 재구성

**Files:**
- Create: `cmd/member.go` (member workspace, member run)
- Create: `cmd/team.go` (team workspace, team run)
- Modify: `cmd/run.go` (run list, stop, attach, tell, note, logs)
- Modify: `cmd/root.go`

- [ ] **Step 1: member 명령어**

`cmd/member.go`:

```go
package cmd

import "github.com/spf13/cobra"

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage members",
}

func newMemberWorkspaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace <owner/name>",
		Short: "Create workspace for a member (download only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse owner/name
			// Call workspace.Writer.PrepareMember()
			return nil
		},
	}
}

func newMemberRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <owner/name>",
		Short: "Create workspace (idempotent) and run member",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Workspace (idempotent)
			// 2. Runner.Run()
			return nil
		},
	}
}

func init() {
	memberCmd.AddCommand(newMemberWorkspaceCmd())
	memberCmd.AddCommand(newMemberRunCmd())
	rootCmd.AddCommand(memberCmd)
}
```

- [ ] **Step 2: team 명령어** — member와 동일 패턴.

- [ ] **Step 3: run 명령어 (관리용)**

```go
// cmd/run.go
var runCmd = &cobra.Command{Use: "run", Short: "Manage runs"}

func newRunListCmd()   // GET /api/v1/runs
func newRunStopCmd()   // PATCH /api/v1/runs/:id + tmux kill
func newRunAttachCmd() // .clier/{RUN_ID}.json 읽기 → tmux attach
func newRunTellCmd()   // .clier/{RUN_ID}.json 읽기 → tmux send-keys + POST /runs/:id/messages
func newRunNoteCmd()   // POST /runs/:id/notes
func newRunLogsCmd()   // GET /runs/:id (messages + notes)
```

- [ ] **Step 4: root.go에서 agent 모드 필터링 업데이트**

```go
if os.Getenv("CLIER_AGENT") == "true" {
	filterAgentCommands() // run tell, run note만 노출
}
```

- [ ] **Step 5: 빌드 확인 및 커밋**

```bash
go build ./...
git add -A && git commit -m "feat: CLI 명령어 재구성

- member workspace / member run
- team workspace / team run
- run list / stop / attach / tell / note / logs
- CLIER_AGENT env var로 agent 모드 활성화

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 13: 삭제 대상 정리

**Files:**
- Delete: `internal/adapter/runtime/` (AgentRuntime 인터페이스 단순화 → detect.go로 대체)
- Delete: `internal/adapter/terminal/` (Runner에 통합)
- Verify: 모든 placeholder, auth, merge 코드 제거 완료

- [ ] **Step 1: 남은 파일 정리**

```bash
# 사용되지 않는 파일 확인
grep -r "runtime\." --include="*.go" -l
grep -r "terminal\." --include="*.go" -l
```

- [ ] **Step 2: 불필요한 파일/디렉토리 삭제**

```bash
rm -rf internal/adapter/runtime/
# terminal은 Runner에 통합되었으므로 필요 시 삭제
go build ./...
go test ./...
```

- [ ] **Step 3: go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 4: 최종 커밋**

```bash
git add -A && git commit -m "chore: 불필요한 파일 정리

AgentRuntime 인터페이스 → detect.go로 대체.
terminal adapter → Runner에 통합.
sqlite 의존성 제거.

Co-Authored-By: Claude <noreply@anthropic.com>"
```
