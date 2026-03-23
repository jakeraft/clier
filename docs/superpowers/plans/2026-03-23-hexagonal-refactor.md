# Hexagonal Architecture Refactoring Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** flat 패키지 구조를 헥사고날 아키텍처(domain / app / adapter)로 재배치

**Architecture:** 디렉토리 이동 + import 경로 변경 + Engine→Service 리네임. 로직 변경 없음. 각 태스크는 이동 → 빌드 확인 → 테스트 확인 → 커밋 순서.

**Tech Stack:** Go 1.25, sqlc, Cobra

**Spec:** `docs/superpowers/specs/2026-03-23-hexagonal-refactor-design.md`

---

### Task 1: adapter/db 이동

`internal/db/` → `internal/adapter/db/`

**Files:**
- Move: `internal/db/*` → `internal/adapter/db/*`
- Move: `internal/db/generated/*` → `internal/adapter/db/generated/*`
- Modify: `internal/adapter/db/store.go` (import 경로)
- Modify: `internal/adapter/db/sqlc.yaml` (경로 확인)
- Modify: `internal/sprint/sprint.go` (import 경로)
- Modify: `internal/sprint/message.go` (import 경로 — 아직 generated 참조 있으면)

- [ ] **Step 1: 디렉토리 생성 및 파일 이동**

```bash
mkdir -p internal/adapter/db
mv internal/db/* internal/adapter/db/
rmdir internal/db
```

- [ ] **Step 2: store.go import 경로 수정**

`internal/adapter/db/store.go`에서:
```
"github.com/jakeraft/clier/internal/db/generated"
→ "github.com/jakeraft/clier/internal/adapter/db/generated"
```

- [ ] **Step 3: sprint 패키지의 db import 경로 수정**

`internal/sprint/sprint.go`, `internal/sprint/message.go` 등에서 db import가 있으면 경로 수정.
현재 sprint는 db를 직접 import하지 않으므로 확인만.

- [ ] **Step 4: sqlc.yaml 경로 확인**

`internal/adapter/db/sqlc.yaml`의 상대 경로(queries, schema, out)는 sqlc.yaml 기준이므로 변경 불필요 확인.

- [ ] **Step 5: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 6: 테스트 확인**

```bash
go test ./...
```

- [ ] **Step 7: 커밋**

```bash
git add -A
git commit -m "refactor: move db to adapter/db"
```

---

### Task 2: adapter/terminal 이동

`internal/terminal/` → `internal/adapter/terminal/`

**Files:**
- Move: `internal/terminal/*` → `internal/adapter/terminal/*`
- Modify: `internal/sprint/sprint.go` (import 경로)
- Modify: `internal/sprint/command.go` (import 경로 — terminal.ShellQuote)

- [ ] **Step 1: 디렉토리 생성 및 파일 이동**

```bash
mkdir -p internal/adapter/terminal
mv internal/terminal/* internal/adapter/terminal/
rmdir internal/terminal
```

- [ ] **Step 2: sprint 패키지 import 경로 수정**

`internal/sprint/sprint.go`, `internal/sprint/command.go`에서:
```
"github.com/jakeraft/clier/internal/terminal"
→ "github.com/jakeraft/clier/internal/adapter/terminal"
```

- [ ] **Step 3: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 4: 테스트 확인**

```bash
go test ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A
git commit -m "refactor: move terminal to adapter/terminal"
```

---

### Task 3: adapter/settings 이동

`internal/settings/` → `internal/adapter/settings/`

**Files:**
- Move: `internal/settings/*` → `internal/adapter/settings/*`
- Modify: `internal/sprint/sprint.go` (import 경로)
- Modify: `cmd/agent.go` (import 경로)
- Modify: `cmd/git.go` (import 경로)

- [ ] **Step 1: 디렉토리 생성 및 파일 이동**

```bash
mkdir -p internal/adapter/settings
mv internal/settings/* internal/adapter/settings/
rmdir internal/settings
```

- [ ] **Step 2: 모든 import 경로 수정**

`internal/sprint/sprint.go`, `cmd/agent.go`, `cmd/git.go`에서:
```
"github.com/jakeraft/clier/internal/settings"
→ "github.com/jakeraft/clier/internal/adapter/settings"
```

- [ ] **Step 3: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 4: 테스트 확인**

```bash
go test ./...
```

- [ ] **Step 5: 커밋**

```bash
git add -A
git commit -m "refactor: move settings to adapter/settings"
```

---

### Task 4: app/sprint 이동 + Engine→Service 리네임

`internal/sprint/` → `internal/app/sprint/`

**Files:**
- Move: `internal/sprint/*` → `internal/app/sprint/*`
- Modify: `internal/app/sprint/sprint.go` (Engine→Service, NewEngine→New, import 경로)
- Modify: `internal/app/sprint/command.go` (import 경로)
- Modify: `internal/app/sprint/message.go` (import 경로)
- Modify: `internal/app/sprint/protocol.go` (import 경로)
- Modify: `internal/app/sprint/*_test.go` (import 경로)
- Modify: `cmd/` (sprint import 경로, Engine→Service 호출 변경)

- [ ] **Step 1: 디렉토리 생성 및 파일 이동**

```bash
mkdir -p internal/app/sprint
mv internal/sprint/* internal/app/sprint/
rmdir internal/sprint
```

- [ ] **Step 2: sprint 패키지 내 import 경로 수정**

모든 `internal/app/sprint/*.go` 파일에서:
```
"github.com/jakeraft/clier/internal/terminal"
→ "github.com/jakeraft/clier/internal/adapter/terminal"

"github.com/jakeraft/clier/internal/settings"
→ "github.com/jakeraft/clier/internal/adapter/settings"
```

(domain import는 변경 없음)

- [ ] **Step 3: Engine → Service 리네임**

`internal/app/sprint/sprint.go`에서:
```go
// Engine → Service
type Service struct { ... }
func New(store Store, term Terminal, s *settings.Settings) *Service { ... }
```

- [ ] **Step 4: cmd에서 sprint import 경로 및 호출 수정**

cmd에서 sprint를 사용하는 파일이 있으면:
```
"github.com/jakeraft/clier/internal/sprint"
→ "github.com/jakeraft/clier/internal/app/sprint"
```

`sprint.NewEngine(...)` → `sprint.New(...)`

- [ ] **Step 5: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 6: 테스트 확인**

```bash
go test ./...
```

- [ ] **Step 7: 커밋**

```bash
git add -A
git commit -m "refactor: move sprint to app/sprint, rename Engine to Service"
```

---

### Task 5: CLAUDE.md 프로젝트 구조 업데이트

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Project Structure 섹션 업데이트**

```markdown
## Project Structure

gh (GitHub CLI) style + hexagonal architecture.

\```
main.go               ← entry point (cmd.Execute() only)
cmd/                   ← driving adapter (cobra commands, 의존성 조립)
internal/
  domain/              ← entities + business rules (no external deps)
  app/
    sprint/            ← sprint use case (port definitions + orchestration)
  adapter/
    db/                ← driven adapter (sqlc generated → domain conversion)
    terminal/          ← driven adapter (cmux CLI)
    settings/          ← driven adapter (local config/credentials)
\```
```

- [ ] **Step 2: 빌드 확인**

```bash
go build ./...
```

- [ ] **Step 3: 커밋**

```bash
git add CLAUDE.md
git commit -m "docs: update project structure in CLAUDE.md"
```
