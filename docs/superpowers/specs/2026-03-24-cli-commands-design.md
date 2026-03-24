# CLI Commands Design

## Overview

clier의 모든 리소스를 관리하는 CLI 커맨드를 추가한다. 모든 출력은 JSON (stdout), 에러는 stderr.

## Command Tree

```
clier
├── claude login/check              (기존)
├── codex login/check               (기존)
├── profile create/list/update/delete
├── prompt create/list/update/delete
├── env create/list/update/delete
├── repo create/list/update/delete
├── member create/list/update/delete
├── team
│   ├── create/list/update/delete
│   ├── member add/remove/list
│   └── relation add/remove/list
├── sprint start/stop/list
└── message send
```

## Conventions

### Output

- 모든 커맨드는 JSON을 stdout에 출력한다.
- 에러 메시지만 stderr에 출력한다.
- `output.go`에 공통 헬퍼를 정의한다.

```go
func printJSON(v any) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
```

### JSON Key Convention

- Go 코드: CamelCase (`RootMemberID`)
- JSON 출력: snake_case (`root_member_id`)
- 도메인 엔티티에 `json:"snake_case"` 태그를 추가한다.

### Use/Short Pattern

동사를 통일한다:

| 동사 | Short 패턴 | Args |
|------|-----------|------|
| `create` | `Create a <resource>` | `--name` 등 required 플래그 |
| `list` | `List all <resources>` | 없음 |
| `update` | `Update a <resource>` | `<id>` positional + optional 플래그 |
| `delete` | `Delete a <resource>` | `<id>` positional |

### Flag Convention

- kebab-case: `--root-member`, `--cli-profile`
- ID 참조는 positional argument: `clier team delete <id>`
- 필수 플래그는 cobra `MarkFlagRequired`로 강제

### Error Handling

- `RunE`를 사용, cobra가 에러를 stderr에 출력한다.
- Store/domain 에러는 wrapping 없이 그대로 반환한다.

## Commands Detail

### profile

```bash
clier profile create --name <name> --binary <claude|codex> --model <model>
clier profile list
clier profile update <id> [--name <name>] [--model <model>]
clier profile delete <id>
```

### prompt

```bash
clier prompt create --name <name> --prompt <text>
clier prompt list
clier prompt update <id> [--name <name>] [--prompt <text>]
clier prompt delete <id>
```

### env

```bash
clier env create --name <name> --key <key> --value <value>
clier env list
clier env update <id> [--name <name>] [--key <key>] [--value <value>]
clier env delete <id>
```

### repo

```bash
clier repo create --name <name> --url <url>
clier repo list
clier repo update <id> [--name <name>] [--url <url>]
clier repo delete <id>
```

### member

```bash
clier member create --name <name> --profile <id> [--prompts <id,...>] [--envs <id,...>] [--repo <id>]
clier member list
clier member update <id> [--name <name>] [--profile <id>] [--prompts <id,...>] [--envs <id,...>] [--repo <id>]
clier member delete <id>
```

### team

```bash
clier team create --name <name> --root-member <id>
clier team list
clier team update <id> [--name <name>] [--root-member <id>]
clier team delete <id>

clier team member add <team-id> <member-id>
clier team member remove <team-id> <member-id>
clier team member list <team-id>

clier team relation add <team-id> --from <id> --to <id> --type <leader|peer>
clier team relation remove <team-id> --from <id> --to <id> --type <leader|peer>
clier team relation list <team-id>
```

### sprint

```bash
clier sprint start --team <id>
clier sprint stop <id>
clier sprint list
```

### message

```bash
clier message send --to <member-id> "<content>"
```

`message send`는 환경변수 `CLIER_SPRINT_ID`, `CLIER_MEMBER_ID`를 읽어 sender와 sprint를 식별한다.

## File Structure

```
cmd/
├── root.go        (기존)
├── agent.go       (기존)
├── output.go      (JSON 출력 헬퍼)
├── profile.go
├── prompt.go
├── env.go
├── repo.go
├── member.go
├── team.go        (team CRUD + member/relation 서브커맨드)
├── sprint.go
└── message.go
```

### 의존성 조립

각 커맨드 파일의 `RunE`에서 Store를 생성하고 도메인/서비스를 호출한다.

```go
RunE: func(cmd *cobra.Command, args []string) error {
    store, err := newStore()  // root.go에 정의
    if err != nil { return err }
    defer store.Close()

    // domain/service 호출
    // printJSON(result)
}
```

Sprint 커맨드는 추가로 Terminal, Workspace 어댑터를 조립한다.

## Domain JSON Tags

모든 도메인 엔티티에 `json:"snake_case"` 태그를 추가한다.

대상: Team, Member, CliProfile, SystemPrompt, Environment, GitRepo, Sprint, Message, TeamSnapshot, MemberSnapshot, PromptSnapshot, EnvironmentSnapshot, GitRepoSnapshot, MemberRelations, Relation
