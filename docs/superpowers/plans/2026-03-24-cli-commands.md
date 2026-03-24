# CLI Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** clier의 모든 리소스를 CRUD하고 sprint/message를 실행하는 CLI 커맨드를 추가한다.

**Architecture:** cmd 레이어에서 cobra 커맨드를 정의하고, 각 RunE에서 Store를 생성한 뒤 domain/service를 호출하여 결과를 JSON으로 출력한다. 모든 출력은 `printJSON` 헬퍼를 통해 stdout에 raw JSON으로 나간다.

**Tech Stack:** Go 1.25, Cobra CLI, SQLite (modernc.org/sqlite), sqlc

---

### Task 1: Domain JSON Tags

**Files:**
- Modify: `internal/domain/team.go`
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/cliprofile.go`
- Modify: `internal/domain/systemprompt.go`
- Modify: `internal/domain/environment.go`
- Modify: `internal/domain/gitrepo.go`
- Modify: `internal/domain/sprint.go`
- Modify: `internal/domain/message.go`
- Modify: `internal/domain/snapshot.go`

- [ ] **Step 1: Add json tags to all domain structs**

모든 exported struct 필드에 `json:"snake_case"` 태그를 추가한다.

Team:
```go
type Team struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	RootMemberID string      `json:"root_member_id"`
	MemberIDs    []string    `json:"member_ids"`
	Relations    []Relation  `json:"relations"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Relation struct {
	From string       `json:"from"`
	To   string       `json:"to"`
	Type RelationType `json:"type"`
}

type MemberRelations struct {
	Leaders []string `json:"leaders"`
	Workers []string `json:"workers"`
	Peers   []string `json:"peers"`
}
```

Member:
```go
type Member struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	CliProfileID    string    `json:"cli_profile_id"`
	SystemPromptIDs []string  `json:"system_prompt_ids"`
	EnvironmentIDs  []string  `json:"environment_ids"`
	GitRepoID       string    `json:"git_repo_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

CliProfile:
```go
type CliProfile struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	Binary     CliBinary `json:"binary"`
	SystemArgs []string  `json:"system_args"`
	CustomArgs []string  `json:"custom_args"`
	DotConfig  DotConfig `json:"dot_config"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
```

SystemPrompt:
```go
type SystemPrompt struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prompt    string    `json:"prompt"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

Environment:
```go
type Environment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

GitRepo:
```go
type GitRepo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

Sprint:
```go
type Sprint struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	TeamSnapshot TeamSnapshot `json:"team_snapshot"`
	State        SprintState  `json:"state"`
	Error        string       `json:"error"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
```

Message:
```go
type Message struct {
	ID           string    `json:"id"`
	SprintID     string    `json:"sprint_id"`
	FromMemberID string    `json:"from_member_id"`
	ToMemberID   string    `json:"to_member_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}
```

Snapshots:
```go
type TeamSnapshot struct {
	TeamName     string           `json:"team_name"`
	RootMemberID string           `json:"root_member_id"`
	Members      []MemberSnapshot `json:"members"`
}

type MemberSnapshot struct {
	MemberID       string                `json:"member_id"`
	MemberName     string                `json:"member_name"`
	Binary         CliBinary             `json:"binary"`
	Model          string                `json:"model"`
	CliProfileName string                `json:"cli_profile_name"`
	SystemArgs     []string              `json:"system_args"`
	CustomArgs     []string              `json:"custom_args"`
	DotConfig      DotConfig             `json:"dot_config"`
	SystemPrompts  []PromptSnapshot      `json:"system_prompts"`
	Environments   []EnvironmentSnapshot `json:"environments"`
	GitRepo        *GitRepoSnapshot      `json:"git_repo"`
	Relations      MemberRelations       `json:"relations"`
}

type PromptSnapshot struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type EnvironmentSnapshot struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GitRepoSnapshot struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
```

- [ ] **Step 2: Run all tests**

Run: `go test ./internal/...`
Expected: All PASS (json tags don't break existing code)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/
git commit -m "feat(domain): add json tags to all structs"
```

---

### Task 2: cmd 기반 (output.go + root.go 수정)

**Files:**
- Create: `cmd/output.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create output.go**

```go
package cmd

import (
	"encoding/json"
	"os"
)

func printJSON(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
```

- [ ] **Step 2: Add newStore helper to root.go**

```go
import (
	"path/filepath"
	"github.com/jakeraft/clier/internal/adapter/db"
)

func newStore() (*db.Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	dbPath := filepath.Join(home, configDirName, "clier.db")
	return db.NewStore(dbPath)
}
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add cmd/output.go cmd/root.go
git commit -m "feat(cmd): add printJSON helper and newStore"
```

---

### Task 3: profile CRUD

**Files:**
- Create: `cmd/profile.go`

- [ ] **Step 1: Implement profile commands**

```go
package cmd

import (
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newProfileCmd())
}

func newProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage CLI profiles",
	}
	cmd.AddCommand(newProfileCreateCmd())
	cmd.AddCommand(newProfileListCmd())
	cmd.AddCommand(newProfileUpdateCmd())
	cmd.AddCommand(newProfileDeleteCmd())
	return cmd
}

func newProfileCreateCmd() *cobra.Command {
	var name, binary, model string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a CLI profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil { return err }
			defer store.Close()

			p, err := domain.NewCliProfile(name, domain.CliBinary(binary), model)
			if err != nil { return err }
			if err := store.CreateCliProfile(cmd.Context(), p); err != nil { return err }
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	cmd.Flags().StringVar(&binary, "binary", "", "CLI binary (claude|codex)")
	cmd.Flags().StringVar(&model, "model", "", "Model name")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("binary")
	_ = cmd.MarkFlagRequired("model")
	return cmd
}

func newProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all CLI profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil { return err }
			defer store.Close()

			profiles, err := store.ListCliProfiles(cmd.Context())
			if err != nil { return err }
			return printJSON(profiles)
		},
	}
}

func newProfileUpdateCmd() *cobra.Command {
	var name, model string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a CLI profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil { return err }
			defer store.Close()

			p, err := store.GetCliProfile(cmd.Context(), args[0])
			if err != nil { return err }

			var namePtr, modelPtr *string
			if cmd.Flags().Changed("name") { namePtr = &name }
			if cmd.Flags().Changed("model") { modelPtr = &model }

			if err := p.Update(namePtr, modelPtr); err != nil { return err }
			if err := store.UpdateCliProfile(cmd.Context(), &p); err != nil { return err }
			return printJSON(p)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Profile name")
	cmd.Flags().StringVar(&model, "model", "", "Model name")
	return cmd
}

func newProfileDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a CLI profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newStore()
			if err != nil { return err }
			defer store.Close()

			if err := store.DeleteCliProfile(cmd.Context(), args[0]); err != nil { return err }
			return printJSON(map[string]string{"deleted": args[0]})
		},
	}
}
```

- [ ] **Step 2: Build check**

Run: `go build ./...`
Expected: Success

- [ ] **Step 3: Manual smoke test**

Run: `go run . profile create --name test --binary claude --model claude-sonnet-4-6`
Expected: JSON output with created profile

- [ ] **Step 4: Commit**

```bash
git add cmd/profile.go
git commit -m "feat(cmd): add profile CRUD commands"
```

---

### Task 4: prompt CRUD

**Files:**
- Create: `cmd/prompt.go`

동일 패턴. domain.NewSystemPrompt → store.CreateSystemPrompt → printJSON.

- [ ] **Step 1: Implement prompt commands** (create/list/update/delete, profile.go 패턴 그대로)
- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/prompt.go
git commit -m "feat(cmd): add prompt CRUD commands"
```

---

### Task 5: env CRUD

**Files:**
- Create: `cmd/env.go`

동일 패턴. create는 `--name`, `--key`, `--value` 필수.

- [ ] **Step 1: Implement env commands**
- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/env.go
git commit -m "feat(cmd): add env CRUD commands"
```

---

### Task 6: repo CRUD

**Files:**
- Create: `cmd/repo.go`

동일 패턴. create는 `--name`, `--url` 필수.

- [ ] **Step 1: Implement repo commands**
- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/repo.go
git commit -m "feat(cmd): add repo CRUD commands"
```

---

### Task 7: member CRUD

**Files:**
- Create: `cmd/member.go`

create는 `--name`, `--profile` 필수. `--prompts`, `--envs`는 comma-separated ID 목록. `--repo`는 optional.

- [ ] **Step 1: Implement member commands**

create에서 `--prompts` 파싱:
```go
var promptsStr string
cmd.Flags().StringVar(&promptsStr, "prompts", "", "Comma-separated system prompt IDs")

// RunE 내:
var promptIDs []string
if promptsStr != "" {
    promptIDs = strings.Split(promptsStr, ",")
}
```

- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/member.go
git commit -m "feat(cmd): add member CRUD commands"
```

---

### Task 8: team CRUD + member/relation 서브커맨드

**Files:**
- Create: `cmd/team.go`

team은 가장 복잡: CRUD 4개 + member 3개 + relation 3개 = 10개 커맨드.

- [ ] **Step 1: Implement team CRUD** (create/list/update/delete)

create: `--name`, `--root-member` 필수.
```go
team, err := domain.NewTeam(name, rootMember)
if err != nil { return err }
if err := store.CreateTeam(cmd.Context(), team); err != nil { return err }
return printJSON(team)
```

- [ ] **Step 2: Implement team member subcommands** (add/remove/list)

```go
// team member add <team-id> <member-id>
team, err := store.GetTeam(ctx, args[0])
if err != nil { return err }
if err := team.AddMember(args[1]); err != nil { return err }
if err := store.AddTeamMember(ctx, team.ID, args[1]); err != nil { return err }
return printJSON(team)
```

list는 team.MemberIDs를 출력.

- [ ] **Step 3: Implement team relation subcommands** (add/remove/list)

```go
// team relation add <team-id> --from <id> --to <id> --type <leader|peer>
team, err := store.GetTeam(ctx, args[0])
if err != nil { return err }
r := domain.Relation{From: from, To: to, Type: domain.RelationType(relType)}
if err := team.AddRelation(r); err != nil { return err }
if err := store.AddTeamRelation(ctx, team.ID, r); err != nil { return err }
return printJSON(team)
```

- [ ] **Step 4: Build check**
- [ ] **Step 5: Commit**

```bash
git add cmd/team.go
git commit -m "feat(cmd): add team CRUD, member, and relation commands"
```

---

### Task 9: sprint start/stop/list

**Files:**
- Create: `cmd/sprint.go`

sprint start는 Service를 조립해야 한다 (Store + Terminal + Workspace).

- [ ] **Step 1: Implement sprint commands**

```go
// sprint start --team <id>
func newSprintStartCmd() *cobra.Command {
    var teamID string
    cmd := &cobra.Command{
        Use:   "start",
        Short: "Start a sprint",
        RunE: func(cmd *cobra.Command, args []string) error {
            store, err := newStore()
            if err != nil { return err }
            defer store.Close()

            svc := sprint.New(store, terminal.NewCmuxTerminal(), workspace.NewWorkspace())
            sp, err := svc.Start(cmd.Context(), teamID)
            if err != nil { return err }
            return printJSON(sp)
        },
    }
    cmd.Flags().StringVar(&teamID, "team", "", "Team ID")
    _ = cmd.MarkFlagRequired("team")
    return cmd
}
```

stop: `clier sprint stop <id>`
list: store.ListSprints → printJSON

- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/sprint.go
git commit -m "feat(cmd): add sprint start/stop/list commands"
```

---

### Task 10: message send

**Files:**
- Create: `cmd/message.go`

환경변수에서 CLIER_SPRINT_ID, CLIER_MEMBER_ID를 읽는다.

- [ ] **Step 1: Implement message send**

```go
func newMessageSendCmd() *cobra.Command {
    var toMemberID string
    cmd := &cobra.Command{
        Use:   "send <content>",
        Short: "Send a message to a teammate",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            sprintID := os.Getenv("CLIER_SPRINT_ID")
            fromMemberID := os.Getenv("CLIER_MEMBER_ID")
            if sprintID == "" || fromMemberID == "" {
                return fmt.Errorf("CLIER_SPRINT_ID and CLIER_MEMBER_ID must be set")
            }

            store, err := newStore()
            if err != nil { return err }
            defer store.Close()

            svc := sprint.New(store, terminal.NewCmuxTerminal(), nil)
            if err := svc.DeliverMessage(cmd.Context(), sprintID, fromMemberID, toMemberID, args[0]); err != nil {
                return err
            }
            return printJSON(map[string]string{
                "status": "delivered",
                "from":   fromMemberID,
                "to":     toMemberID,
            })
        },
    }
    cmd.Flags().StringVar(&toMemberID, "to", "", "Recipient member ID")
    _ = cmd.MarkFlagRequired("to")
    return cmd
}
```

- [ ] **Step 2: Build check**
- [ ] **Step 3: Commit**

```bash
git add cmd/message.go
git commit -m "feat(cmd): add message send command"
```

---

### Task 11: 전체 검증

- [ ] **Step 1: go build && go vet**

Run: `go build ./... && go vet ./...`

- [ ] **Step 2: go test ./internal/...**

Run: `go test ./internal/...`
Expected: All PASS

- [ ] **Step 3: --help 일관성 확인**

Run: `go run . --help`
확인: 모든 서브커맨드가 일관된 Short 메시지를 가지는지

- [ ] **Step 4: Smoke test**

```bash
go run . profile create --name test --binary claude --model claude-sonnet-4-6
go run . prompt create --name greeting --prompt "Hello"
go run . env create --name api --key API_KEY --value secret
go run . repo create --name myrepo --url https://github.com/example/repo
```

- [ ] **Step 5: Final commit (if any fixes)**
