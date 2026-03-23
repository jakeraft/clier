# Hexagonal Architecture Refactoring

## 목적

flat하게 나열된 패키지 구조를 헥사고날 아키텍처(포트 & 어댑터)로 재배치하여, 각 레이어의 역할과 의존 방향을 명확히 한다.

## 디렉토리 구조

```
cmd/                              ← driving adapter (cobra, 의존성 조립)

internal/
  domain/                         ← 엔티티 + 비즈니스 규칙 (순수, 외부 의존 없음)
  app/
    sprint/                       ← sprint 유즈케이스 (port 정의 + 오케스트레이션)
  adapter/
    db/                           ← driven adapter (generated → domain 변환)
    terminal/                     ← driven adapter (cmux CLI)
    settings/                     ← driven adapter (로컬 설정/인증)
```

## 의존 방향

```
cmd → app/sprint, adapter/*, domain
app/sprint → domain (+ 자체 port interface)
adapter/* → domain, generated
domain → 없음 (순수)
```

- `app/sprint`는 adapter를 모른다. 자체 정의한 port(interface)만 안다.
- adapter는 port를 implicit하게 satisfy한다.
- cmd에서 adapter를 생성하여 app에 주입한다.

## 레이어별 역할

### domain

순수 엔티티와 비즈니스 규칙. 외부 의존성 없음.

- `team.go` — Team struct, GetMemberRelations, AddRelation 등
- `sprint.go` — Sprint struct, state machine
- `snapshot.go` — TeamSnapshot, MemberSnapshot (Sprint이 참조하는 데이터 구조)
- `member.go`, `cliprofile.go`, `environment.go`, `systemprompt.go`, `gitrepo.go`, `message.go`

변경 없음. 위치 이동 없음.

### app/sprint

sprint 유즈케이스. port(interface) 정의 + 오케스트레이션.

| 파일 | 역할 |
|------|------|
| `service.go` | Service struct, port(Store/Terminal interface), New, Start, Stop, buildSnapshot |
| `message.go` | DeliverMessage, surfaces 파일 관리 |
| `command.go` | BuildCommand, BuildEnv, WriteConfigs |
| `protocol.go` | BuildProtocol, ComposePrompt |

변경 사항:
- `internal/sprint/` → `internal/app/sprint/` 이동
- `Engine` → `Service` 리네임
- `NewEngine` → `New` 리네임
- import 경로 변경

### adapter/db

thin adapter. sqlc generated 타입을 domain 타입으로 변환.

| 파일 | 역할 |
|------|------|
| `store.go` | Store struct, NewStore, Close, 개별 adapter 메서드 |
| `schema.sql` | DB 스키마 |
| `generated/` | sqlc 생성 코드 (건드리지 않음) |

각 메서드는 `generated.X → domain.X` 변환만 수행. 비즈니스 로직 없음.

변경 사항:
- `internal/db/` → `internal/adapter/db/` 이동
- import 경로 변경

### adapter/terminal

cmux CLI를 호출하는 driven adapter.

변경 사항:
- `internal/terminal/` → `internal/adapter/terminal/` 이동
- import 경로 변경

### adapter/settings

로컬 설정 및 인증 관리.

변경 사항:
- `internal/settings/` → `internal/adapter/settings/` 이동
- import 경로 변경

### cmd

driving adapter. cobra 명령 + 의존성 조립.

```go
// cmd/sprint.go
store := db.NewStore(dbPath)
term := terminal.NewCmuxTerminal()
svc := sprint.New(store, term, settings)
svc.Start(ctx, teamID)
```

단순 CRUD(environment, member 등)는 유즈케이스 패키지 없이 `domain` + `adapter/db` 직접 호출.

## 변경 범위

| 항목 | 변경 유형 |
|------|-----------|
| 디렉토리 구조 | 재배치 |
| Engine → Service | 리네임 |
| NewEngine → New | 리네임 |
| import 경로 | 전체 업데이트 |
| domain 로직 | 변경 없음 |
| terminal 코드 | 변경 없음 |
| settings 코드 | 변경 없음 |
| sqlc generated | 변경 없음 |
| db/store.go adapter 메서드 | 변경 없음 |

## 변경하지 않는 것

- domain 엔티티/비즈니스 규칙
- adapter 내부 구현 로직
- 테스트 로직 (import 경로만 변경)
- sqlc 설정 및 생성 코드
