# clier-server Plan 1: Foundation (프로젝트 셋업 + DB + 인증)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** GitHub OAuth로 로그인하고, API 토큰을 발급받아 인증된 요청을 보낼 수 있는 Go 서버를 만든다.

**Architecture:** Echo 프레임워크 기반 모놀리스 서버. sqlx + squirrel로 PostgreSQL 접근. Go-idiomatic 플랫 패키지 구조 (Gitea/Grafana 패턴). 인증은 `auth` 패키지에 인터페이스로 추상화하여 교체 가능.

**Tech Stack:** Go 1.25, Echo v4, sqlx, squirrel, golang-migrate, PostgreSQL, GitHub OAuth (RFC 8628 Device Flow)

**Spec:** `docs/superpowers/specs/2026-04-06-clier-server-design.md`

---

## File Structure

```
clier-server/
├── cmd/server/
│   └── main.go                    # 엔트리포인트, 의존성 조립
├── internal/
│   ├── domain/
│   │   └── user.go                # User, Org, AccessToken 엔티티
│   ├── db/
│   │   └── db.go                  # sqlx 초기화, DB 연결
│   ├── auth/
│   │   ├── auth.go                # Authenticator 인터페이스 + 타입
│   │   └── github/
│   │       └── github.go          # GitHub OAuth 구현체
│   ├── services/
│   │   └── user/
│   │       └── user.go            # 유저/Org 서비스 (생성, 조회, 토큰)
│   ├── handler/
│   │   ├── auth.go                # 인증 핸들러 (/auth/*)
│   │   └── user.go                # 유저 핸들러 (/api/v1/user, /api/v1/orgs)
│   └── middleware/
│       └── auth.go                # Bearer 토큰 인증 미들웨어
├── migrations/
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_org_users.up.sql
│   ├── 000002_create_org_users.down.sql
│   ├── 000003_create_access_tokens.up.sql
│   └── 000003_create_access_tokens.down.sql
├── go.mod
├── go.sum
├── Makefile
└── .env.example
```

---

### Task 1: 프로젝트 초기화

**Files:**
- Create: `clier-server/go.mod`
- Create: `clier-server/cmd/server/main.go`
- Create: `clier-server/Makefile`
- Create: `clier-server/.env.example`
- Create: `clier-server/.gitignore`

- [ ] **Step 1: 프로젝트 디렉토리 생성 + Go 모듈 초기화**

```bash
mkdir -p ~/jakeraft/clier-server/cmd/server
cd ~/jakeraft/clier-server
go mod init github.com/jakeraft/clier-server
```

- [ ] **Step 2: 핵심 의존성 설치**

```bash
cd ~/jakeraft/clier-server
go get github.com/labstack/echo/v4@latest
go get github.com/jmoiron/sqlx
go get github.com/Masterminds/squirrel
go get github.com/lib/pq
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

- [ ] **Step 3: main.go 작성 — Echo 서버 기본 뼈대**

```go
// cmd/server/main.go
package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 4: Makefile 작성**

```makefile
# Makefile
.PHONY: run build test migrate-up migrate-down

run:
	go run ./cmd/server

build:
	go build -o bin/clier-server ./cmd/server

test:
	go test ./... -v

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
```

- [ ] **Step 5: .env.example 작성**

```bash
# .env.example
PORT=8080
DATABASE_URL=postgres://clier:clier@localhost:5432/clier_server?sslmode=disable
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URL=http://localhost:8080/auth/github/callback
```

- [ ] **Step 6: .gitignore 작성**

```bash
cd ~/jakeraft/clier-server
curl -sL https://raw.githubusercontent.com/github/gitignore/main/Go.gitignore > .gitignore
cat >> .gitignore << 'EOF'

# env
.env

# binary
bin/

# IDE
.idea/
.vscode/
EOF
```

- [ ] **Step 7: 서버 실행 확인**

```bash
cd ~/jakeraft/clier-server
go run ./cmd/server &
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}
kill %1
```

- [ ] **Step 8: 커밋**

```bash
cd ~/jakeraft/clier-server
git init
git add .
git commit -m "init: project skeleton with Echo server"
```

---

### Task 2: PostgreSQL 연결 + DB 패키지

**Files:**
- Create: `internal/db/db.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: db.go 작성 — sqlx 초기화**

```go
// internal/db/db.go
package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func New(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}
```

- [ ] **Step 2: main.go에 DB 연결 추가**

```go
// cmd/server/main.go
package main

import (
	"log"
	"os"

	"github.com/jakeraft/clier-server/internal/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	database, err := db.New(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/healthz", func(c echo.Context) error {
		if err := database.Ping(); err != nil {
			return c.JSON(503, map[string]string{"status": "unhealthy"})
		}
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 3: PostgreSQL 로컬 DB 생성 + 연결 확인**

```bash
createdb clier_server
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" go run ./cmd/server &
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}
kill %1
```

- [ ] **Step 4: 커밋**

```bash
git add internal/db/ cmd/server/main.go
git commit -m "feat: add PostgreSQL connection with sqlx"
```

---

### Task 3: DB 마이그레이션 — users, org_users, access_tokens

**Files:**
- Create: `migrations/000001_create_users.up.sql`
- Create: `migrations/000001_create_users.down.sql`
- Create: `migrations/000002_create_org_users.up.sql`
- Create: `migrations/000002_create_org_users.down.sql`
- Create: `migrations/000003_create_access_tokens.up.sql`
- Create: `migrations/000003_create_access_tokens.down.sql`

- [ ] **Step 1: users 마이그레이션 작성**

```sql
-- migrations/000001_create_users.up.sql
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    github_id     BIGINT UNIQUE NOT NULL,
    login         TEXT NOT NULL,
    lower_login   TEXT UNIQUE NOT NULL,
    type          SMALLINT NOT NULL DEFAULT 0,
    visibility    SMALLINT NOT NULL DEFAULT 0,
    avatar_url    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_lower_login ON users(lower_login);
CREATE INDEX idx_users_type ON users(type);
```

```sql
-- migrations/000001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 2: org_users 마이그레이션 작성**

```sql
-- migrations/000002_create_org_users.up.sql
CREATE TABLE org_users (
    org_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX idx_org_users_user_id ON org_users(user_id);
```

```sql
-- migrations/000002_create_org_users.down.sql
DROP TABLE IF EXISTS org_users;
```

- [ ] **Step 3: access_tokens 마이그레이션 작성**

```sql
-- migrations/000003_create_access_tokens.up.sql
CREATE TABLE access_tokens (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_tokens_user_id ON access_tokens(user_id);
CREATE INDEX idx_access_tokens_token_hash ON access_tokens(token_hash);
```

```sql
-- migrations/000003_create_access_tokens.down.sql
DROP TABLE IF EXISTS access_tokens;
```

- [ ] **Step 4: 마이그레이션 실행 확인**

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" make migrate-up
# Expected: 000001, 000002, 000003 순서로 적용됨
```

- [ ] **Step 5: 롤백 확인**

```bash
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" make migrate-down
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" make migrate-up
# Expected: 정상 롤백 후 재적용
```

- [ ] **Step 6: 커밋**

```bash
git add migrations/
git commit -m "feat: add users, org_users, access_tokens migrations"
```

---

### Task 4: Domain 엔티티

**Files:**
- Create: `internal/domain/user.go`

- [ ] **Step 1: User, Org, AccessToken 엔티티 작성**

```go
// internal/domain/user.go
package domain

import "time"

// UserType 구분
const (
	UserTypeIndividual = 0
	UserTypeOrganization = 1
)

// Visibility 수준
const (
	VisibilityPublic  = 0
	VisibilityLimited = 1
	VisibilityPrivate = 2
)

// OrgRole 역할
const (
	OrgRoleOwner  = 0
	OrgRoleMember = 1
)

type User struct {
	ID         int64     `db:"id" json:"id"`
	GitHubID   int64     `db:"github_id" json:"github_id"`
	Login      string    `db:"login" json:"login"`
	LowerLogin string    `db:"lower_login" json:"-"`
	Type       int16     `db:"type" json:"type"`
	Visibility int16     `db:"visibility" json:"visibility"`
	AvatarURL  *string   `db:"avatar_url" json:"avatar_url,omitempty"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

func (u *User) IsOrganization() bool {
	return u.Type == UserTypeOrganization
}

type OrgUser struct {
	OrgID  int64 `db:"org_id" json:"org_id"`
	UserID int64 `db:"user_id" json:"user_id"`
	Role   int16 `db:"role" json:"role"`
}

type AccessToken struct {
	ID        int64      `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"user_id"`
	TokenHash string     `db:"token_hash" json:"-"`
	Name      string     `db:"name" json:"name"`
	ExpiresAt *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}
```

- [ ] **Step 2: 커밋**

```bash
git add internal/domain/
git commit -m "feat: add User, OrgUser, AccessToken domain entities"
```

---

### Task 5: User 서비스

**Files:**
- Create: `internal/services/user/user.go`
- Create: `internal/services/user/user_test.go`

- [ ] **Step 1: 테스트 작성 — CreateUserWithOrg**

```go
// internal/services/user/user_test.go
package user_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jakeraft/clier-server/internal/db"
	"github.com/jakeraft/clier-server/internal/domain"
	"github.com/jakeraft/clier-server/internal/services/user"
)

func setupTestDB(t *testing.T) *user.Service {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	database, err := db.New(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.Exec("DELETE FROM org_users")
		database.Exec("DELETE FROM access_tokens")
		database.Exec("DELETE FROM users")
		database.Close()
	})
	return user.NewService(database)
}

func TestCreateUserWithOrg(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	u, err := svc.CreateUserWithOrg(ctx, int64(12345), "JakeRaft", "https://avatars.githubusercontent.com/u/12345")
	if err != nil {
		t.Fatal(err)
	}

	if u.GitHubID != 12345 {
		t.Errorf("expected github_id 12345, got %d", u.GitHubID)
	}
	if u.Login != "JakeRaft" {
		t.Errorf("expected login JakeRaft, got %s", u.Login)
	}
	if u.LowerLogin != "jakeraft" {
		t.Errorf("expected lower_login jakeraft, got %s", u.LowerLogin)
	}

	// personal org도 생성되었는지 확인
	org, err := svc.GetOrgByName(ctx, "jakeraft")
	if err != nil {
		t.Fatal(err)
	}
	if org.Type != domain.UserTypeOrganization {
		t.Errorf("expected org type, got %d", org.Type)
	}
}

func TestCreateUserWithOrg_DuplicateGitHubID(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	_, err := svc.CreateUserWithOrg(ctx, int64(12345), "JakeRaft", "")
	if err != nil {
		t.Fatal(err)
	}

	// 같은 GitHub ID로 재생성 시 기존 유저 반환 (upsert)
	u, err := svc.FindOrCreateByGitHub(ctx, int64(12345), "JakeRaft", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "JakeRaft" {
		t.Errorf("expected existing user returned")
	}
}
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

```bash
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" go test ./internal/services/user/ -v
# Expected: FAIL — user.Service 미정의
```

- [ ] **Step 3: User 서비스 구현**

```go
// internal/services/user/user.go
package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jakeraft/clier-server/internal/domain"
	"github.com/jmoiron/sqlx"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateUserWithOrg(ctx context.Context, githubID int64, login, avatarURL string) (*domain.User, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	lowerLogin := strings.ToLower(login)

	// 1. 유저 생성
	var u domain.User
	query, args, err := psql.
		Insert("users").
		Columns("github_id", "login", "lower_login", "type", "avatar_url").
		Values(githubID, login, lowerLogin, domain.UserTypeIndividual, nullString(avatarURL)).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build user insert: %w", err)
	}
	if err := tx.QueryRowxContext(ctx, query, args...).StructScan(&u); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	// 2. personal org 생성
	var org domain.User
	query, args, err = psql.
		Insert("users").
		Columns("github_id", "login", "lower_login", "type", "avatar_url").
		Values(githubID*-1, login, lowerLogin+"-org", domain.UserTypeOrganization, nullString(avatarURL)).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build org insert: %w", err)
	}
	if err := tx.QueryRowxContext(ctx, query, args...).StructScan(&org); err != nil {
		return nil, fmt.Errorf("insert org: %w", err)
	}

	// 3. org_users 연결 (owner)
	query, args, err = psql.
		Insert("org_users").
		Columns("org_id", "user_id", "role").
		Values(org.ID, u.ID, domain.OrgRoleOwner).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build org_users insert: %w", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("insert org_user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &u, nil
}

func (s *Service) FindOrCreateByGitHub(ctx context.Context, githubID int64, login, avatarURL string) (*domain.User, error) {
	var u domain.User
	query, args, err := psql.
		Select("*").
		From("users").
		Where(sq.Eq{"github_id": githubID, "type": domain.UserTypeIndividual}).
		ToSql()
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRowxContext(ctx, query, args...).StructScan(&u)
	if err == nil {
		return &u, nil
	}

	return s.CreateUserWithOrg(ctx, githubID, login, avatarURL)
}

func (s *Service) GetOrgByName(ctx context.Context, name string) (*domain.User, error) {
	var u domain.User
	query, args, err := psql.
		Select("*").
		From("users").
		Where(sq.And{
			sq.Eq{"type": domain.UserTypeOrganization},
			sq.Eq{"lower_login": strings.ToLower(name) + "-org"},
		}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if err := s.db.QueryRowxContext(ctx, query, args...).StructScan(&u); err != nil {
		return nil, fmt.Errorf("get org by name: %w", err)
	}
	return &u, nil
}

func (s *Service) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	query, args, err := psql.
		Select("*").
		From("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if err := s.db.QueryRowxContext(ctx, query, args...).StructScan(&u); err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (s *Service) CreateAccessToken(ctx context.Context, userID int64, name string) (string, *domain.AccessToken, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(rawToken)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	var t domain.AccessToken
	query, args, err := psql.
		Insert("access_tokens").
		Columns("user_id", "token_hash", "name").
		Values(userID, tokenHash, name).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return "", nil, err
	}
	if err := s.db.QueryRowxContext(ctx, query, args...).StructScan(&t); err != nil {
		return "", nil, fmt.Errorf("insert token: %w", err)
	}

	return token, &t, nil
}

func (s *Service) GetUserByToken(ctx context.Context, token string) (*domain.User, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	var t domain.AccessToken
	query, args, err := psql.
		Select("*").
		From("access_tokens").
		Where(sq.Eq{"token_hash": tokenHash}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if err := s.db.QueryRowxContext(ctx, query, args...).StructScan(&t); err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	return s.GetUserByID(ctx, t.UserID)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 4: 테스트 실행 — 통과 확인**

```bash
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" go test ./internal/services/user/ -v
# Expected: PASS
```

- [ ] **Step 5: 커밋**

```bash
git add internal/services/user/
git commit -m "feat: add user service with CreateUserWithOrg and token management"
```

---

### Task 6: Auth 인터페이스 + GitHub OAuth 구현

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/github/github.go`
- Create: `internal/auth/github/github_test.go`

- [ ] **Step 1: auth 인터페이스 정의**

```go
// internal/auth/auth.go
package auth

import "context"

// GitHubUser represents user info returned from OAuth provider
type GitHubUser struct {
	ID        int64
	Login     string
	AvatarURL string
}

// Authenticator abstracts OAuth provider.
// Replace github/ implementation with logto/ or others by implementing this interface.
type Authenticator interface {
	AuthorizeURL(state string) string
	Exchange(ctx context.Context, code string) (*GitHubUser, error)
}
```

- [ ] **Step 2: GitHub OAuth 구현체 테스트 작성**

```go
// internal/auth/github/github_test.go
package github_test

import (
	"testing"

	"github.com/jakeraft/clier-server/internal/auth/github"
)

func TestAuthorizeURL(t *testing.T) {
	g := github.New("test-client-id", "test-secret", "http://localhost:8080/auth/github/callback")

	url := g.AuthorizeURL("test-state")

	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	expected := "https://github.com/login/oauth/authorize"
	if len(url) < len(expected) || url[:len(expected)] != expected {
		t.Errorf("expected URL to start with %s, got %s", expected, url)
	}
}
```

- [ ] **Step 3: 테스트 실행 — 실패 확인**

```bash
go test ./internal/auth/github/ -v
# Expected: FAIL — github.New 미정의
```

- [ ] **Step 4: GitHub OAuth 구현체 작성**

```go
// internal/auth/github/github.go
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jakeraft/clier-server/internal/auth"
)

const (
	authorizeURL = "https://github.com/login/oauth/authorize"
	tokenURL     = "https://github.com/login/oauth/access_token"
	userAPIURL   = "https://api.github.com/user"
)

type Client struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

func New(clientID, clientSecret, redirectURL string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

func (c *Client) AuthorizeURL(state string) string {
	params := url.Values{
		"client_id":    {c.clientID},
		"redirect_uri": {c.redirectURL},
		"scope":        {"read:user"},
		"state":        {state},
	}
	return authorizeURL + "?" + params.Encode()
}

func (c *Client) Exchange(ctx context.Context, code string) (*auth.GitHubUser, error) {
	// 1. Exchange code for access token
	data := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"redirect_uri":  {c.redirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("oauth error: %s", tokenResp.Error)
	}

	// 2. Fetch user info
	req, err = http.NewRequestWithContext(ctx, "GET", userAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read user response: %w", err)
	}

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &ghUser); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}

	return &auth.GitHubUser{
		ID:        ghUser.ID,
		Login:     ghUser.Login,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}
```

- [ ] **Step 5: 테스트 실행 — 통과 확인**

```bash
go test ./internal/auth/github/ -v
# Expected: PASS (AuthorizeURL 테스트)
```

- [ ] **Step 6: 커밋**

```bash
git add internal/auth/
git commit -m "feat: add auth interface and GitHub OAuth implementation"
```

---

### Task 7: Auth 미들웨어

**Files:**
- Create: `internal/middleware/auth.go`
- Create: `internal/middleware/auth_test.go`

- [ ] **Step 1: 테스트 작성 — Bearer 토큰 추출**

```go
// internal/middleware/auth_test.go
package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mw "github.com/jakeraft/clier-server/internal/middleware"
	"github.com/labstack/echo/v4"
)

type mockTokenValidator struct {
	userID int64
	err    error
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (int64, error) {
	return m.userID, m.err
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	validator := &mockTokenValidator{userID: 42, err: nil}
	handler := mw.RequireAuth(validator)(func(c echo.Context) error {
		userID := c.Get("user_id").(int64)
		if userID != 42 {
			t.Errorf("expected user_id 42, got %d", userID)
		}
		return c.NoContent(200)
	})

	err := handler(c)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	validator := &mockTokenValidator{}
	handler := mw.RequireAuth(validator)(func(c echo.Context) error {
		return c.NoContent(200)
	})

	_ = handler(c)
	if rec.Code != 401 {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

```bash
go test ./internal/middleware/ -v
# Expected: FAIL — middleware.RequireAuth 미정의
```

- [ ] **Step 3: Auth 미들웨어 구현**

```go
// internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID int64, err error)
}

func RequireAuth(validator TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			userID, err := validator.ValidateToken(c.Request().Context(), parts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			c.Set("user_id", userID)
			return next(c)
		}
	}
}
```

- [ ] **Step 4: 테스트 실행 — 통과 확인**

```bash
go test ./internal/middleware/ -v
# Expected: PASS
```

- [ ] **Step 5: 커밋**

```bash
git add internal/middleware/
git commit -m "feat: add Bearer token auth middleware"
```

---

### Task 8: Auth 핸들러 (OAuth 콜백 + 토큰 발급)

**Files:**
- Create: `internal/handler/auth.go`

- [ ] **Step 1: Auth 핸들러 작성**

```go
// internal/handler/auth.go
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/jakeraft/clier-server/internal/auth"
	"github.com/jakeraft/clier-server/internal/services/user"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authenticator auth.Authenticator
	userService   *user.Service
}

func NewAuthHandler(authenticator auth.Authenticator, userService *user.Service) *AuthHandler {
	return &AuthHandler{
		authenticator: authenticator,
		userService:   userService,
	}
}

// GET /auth/github — redirect to GitHub OAuth
func (h *AuthHandler) GitHubLogin(c echo.Context) error {
	state := generateState()
	// TODO: store state in session/cookie for CSRF validation
	url := h.authenticator.AuthorizeURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// GET /auth/github/callback — handle OAuth callback
func (h *AuthHandler) GitHubCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing code"})
	}

	ghUser, err := h.authenticator.Exchange(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication failed"})
	}

	u, err := h.userService.FindOrCreateByGitHub(c.Request().Context(), ghUser.ID, ghUser.Login, ghUser.AvatarURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	token, tokenObj, err := h.userService.CreateAccessToken(c.Request().Context(), u.ID, "oauth-login")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": token,
		"token_id":     tokenObj.ID,
		"user":         u,
	})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 2: 커밋**

```bash
git add internal/handler/auth.go
git commit -m "feat: add auth handler with GitHub OAuth callback"
```

---

### Task 9: User 핸들러

**Files:**
- Create: `internal/handler/user.go`

- [ ] **Step 1: User 핸들러 작성**

```go
// internal/handler/user.go
package handler

import (
	"net/http"

	"github.com/jakeraft/clier-server/internal/services/user"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userService *user.Service
}

func NewUserHandler(userService *user.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

// GET /api/v1/user — 현재 로그인 유저 정보
func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	userID := c.Get("user_id").(int64)

	u, err := h.userService.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	return c.JSON(http.StatusOK, u)
}

// GET /api/v1/orgs/:org — Org 정보
func (h *UserHandler) GetOrg(c echo.Context) error {
	orgName := c.Param("org")

	org, err := h.userService.GetOrgByName(c.Request().Context(), orgName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "org not found"})
	}

	return c.JSON(http.StatusOK, org)
}
```

- [ ] **Step 2: 커밋**

```bash
git add internal/handler/user.go
git commit -m "feat: add user handler with GetCurrentUser and GetOrg"
```

---

### Task 10: TokenValidator 어댑터 + main.go 라우팅 조립

**Files:**
- Create: `internal/services/user/token_validator.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: TokenValidator 어댑터 작성**

User 서비스가 middleware.TokenValidator 인터페이스를 만족하도록 어댑터 작성:

```go
// internal/services/user/token_validator.go
package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	sq "github.com/Masterminds/squirrel"
	"github.com/jakeraft/clier-server/internal/domain"
)

// ValidateToken implements middleware.TokenValidator
func (s *Service) ValidateToken(ctx context.Context, token string) (int64, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	var t domain.AccessToken
	query, args, err := psql.
		Select("*").
		From("access_tokens").
		Where(sq.Eq{"token_hash": tokenHash}).
		ToSql()
	if err != nil {
		return 0, err
	}
	if err := s.db.QueryRowxContext(ctx, query, args...).StructScan(&t); err != nil {
		return 0, err
	}

	return t.UserID, nil
}
```

- [ ] **Step 2: main.go 최종 조립**

```go
// cmd/server/main.go
package main

import (
	"log"
	"os"

	"github.com/jakeraft/clier-server/internal/auth/github"
	"github.com/jakeraft/clier-server/internal/db"
	"github.com/jakeraft/clier-server/internal/handler"
	"github.com/jakeraft/clier-server/internal/middleware"
	"github.com/jakeraft/clier-server/internal/services/user"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Config
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ghClientID := os.Getenv("GITHUB_CLIENT_ID")
	ghClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	ghRedirectURL := os.Getenv("GITHUB_REDIRECT_URL")

	// DB
	database, err := db.New(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Services
	userSvc := user.NewService(database)

	// Auth
	ghAuth := github.New(ghClientID, ghClientSecret, ghRedirectURL)

	// Handlers
	authHandler := handler.NewAuthHandler(ghAuth, userSvc)
	userHandler := handler.NewUserHandler(userSvc)

	// Echo
	e := echo.New()
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORS())

	// Health
	e.GET("/healthz", func(c echo.Context) error {
		if err := database.Ping(); err != nil {
			return c.JSON(503, map[string]string{"status": "unhealthy"})
		}
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	e.GET("/auth/github", authHandler.GitHubLogin)
	e.GET("/auth/github/callback", authHandler.GitHubCallback)

	// API routes (authenticated)
	api := e.Group("/api/v1", middleware.RequireAuth(userSvc))
	api.GET("/user", userHandler.GetCurrentUser)
	api.GET("/orgs/:org", userHandler.GetOrg)

	// Start
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(e.Start(":" + port))
}
```

- [ ] **Step 3: 컴파일 확인**

```bash
cd ~/jakeraft/clier-server
go build ./cmd/server
# Expected: 에러 없이 빌드 성공
```

- [ ] **Step 4: 전체 테스트 실행**

```bash
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" go test ./... -v
# Expected: 모든 테스트 PASS
```

- [ ] **Step 5: E2E 수동 확인**

```bash
# 서버 시작
DATABASE_URL="postgres://localhost:5432/clier_server?sslmode=disable" go run ./cmd/server &

# 1. Health check
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}

# 2. 미인증 요청
curl http://localhost:8080/api/v1/user
# Expected: 401 {"error":"missing authorization header"}

# 3. GitHub OAuth URL 확인 (리다이렉트됨)
curl -v http://localhost:8080/auth/github 2>&1 | grep Location
# Expected: Location: https://github.com/login/oauth/authorize?...

kill %1
```

- [ ] **Step 6: 커밋**

```bash
git add internal/services/user/token_validator.go cmd/server/main.go
git commit -m "feat: wire up auth, user handlers and middleware in main.go"
```

---

## Plan 1 완료 상태

이 plan이 완료되면:
- ✅ Echo 서버가 PostgreSQL에 연결되어 동작
- ✅ GitHub OAuth로 유저 가입/로그인 + personal org 자동 생성
- ✅ API 토큰 발급 + Bearer 토큰 인증 미들웨어
- ✅ `/api/v1/user`, `/api/v1/orgs/:org` 엔드포인트 동작
- ✅ 테스트 코드 포함

다음 Plan 2에서 리소스 CRUD API (profiles, prompts, members, teams)를 추가한다.
