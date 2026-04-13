# Unified Resource API Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate clier CLI to the unified ResourceResponse server API, eliminate per-type code duplication, add Organization support.

**Architecture:** Bottom-up migration — API types first, then client methods, domain models, workspace service, and finally cmd/ layer. Each task produces compilable code. Old files deleted only after replacements are in place.

**Tech Stack:** Go 1.25.8, cobra, net/http, encoding/json

**Spec:** `docs/superpowers/specs/2026-04-13-unified-resource-api-migration.md`

---

### Task 1: Rewrite API types

**Files:**
- Rewrite: `internal/adapter/api/types.go`
- Delete: `internal/adapter/api/types_test.go` (will be rewritten later)

- [ ] **Step 1: Rewrite types.go with unified types**

Replace the entire contents of `internal/adapter/api/types.go` with:

```go
package api

import (
	"encoding/json"
	"time"
)

// --- Response Types ---

// ResourceResponse is the unified response for all resource kinds.
type ResourceResponse struct {
	Kind       string           `json:"kind"`
	Metadata   ResourceMetadata `json:"metadata"`
	Spec       json.RawMessage  `json:"spec"`
	Refs       []ResolvedRef    `json:"refs"`
	AgentTypes []string         `json:"agent_types"`
}

// ResourceMetadata contains shared metadata across all resource kinds.
type ResourceMetadata struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Summary        string    `json:"summary"`
	Visibility     int       `json:"visibility"`
	IsFork         bool      `json:"is_fork"`
	ForkCount      int       `json:"fork_count"`
	ForkID         *int64    `json:"fork_id,omitempty"`
	ForkName       string    `json:"fork_name,omitempty"`
	ForkOwnerName  string    `json:"fork_owner_name,omitempty"`
	ForkVersion    *int64    `json:"fork_version,omitempty"`
	OwnerName      string    `json:"owner_name"`
	OwnerType      int       `json:"owner_type"`
	OwnerAvatarURL string    `json:"owner_avatar_url,omitempty"`
	LatestVersion  *int      `json:"latest_version,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ResolvedRef is a resolved reference between resources.
type ResolvedRef struct {
	ID             int64  `json:"id"`
	TargetID       int64  `json:"target_id"`
	TargetVersion  int    `json:"target_version"`
	RelType        string `json:"rel_type"`
	Name           string `json:"name"`
	OwnerName      string `json:"owner_name"`
	AgentType      string `json:"agent_type,omitempty"`
	Command        string `json:"command,omitempty"`
	OwnerAvatarURL string `json:"owner_avatar_url,omitempty"`
}

// ListResponse is a paginated list wrapper.
type ListResponse struct {
	Items []ResourceResponse `json:"items"`
	Total int                `json:"total"`
}

// ResourceVersionResponse is the unified version response.
type ResourceVersionResponse struct {
	ID             int64           `json:"id"`
	ResourceID     int64           `json:"resource_id"`
	Version        int             `json:"version"`
	Snapshot       json.RawMessage `json:"snapshot"`
	CreatedAt      time.Time       `json:"created_at"`
	OwnerName      string          `json:"owner_name"`
	OwnerAvatarURL string          `json:"owner_avatar_url,omitempty"`
}

// --- Spec Types (decoded from ResourceResponse.Spec per kind) ---

// ContentSpec is the spec for claude-md, claude-settings, skill.
type ContentSpec struct {
	Content string `json:"content"`
}

// MemberSpec is the spec for member.
type MemberSpec struct {
	AgentType  string `json:"agent_type"`
	Command    string `json:"command"`
	GitRepoURL string `json:"git_repo_url"`
}

// TeamSpec is the spec for team.
type TeamSpec struct {
	Relations []TeamRelation `json:"relations"`
}

// TeamRelation is a relation in team spec response.
// from/to are member resource IDs (match refs[].target_id).
type TeamRelation struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// DecodeSpec extracts a typed spec from ResourceResponse.
func DecodeSpec[T any](r *ResourceResponse) (*T, error) {
	var spec T
	return &spec, json.Unmarshal(r.Spec, &spec)
}

// --- Request Types ---

// ContentWriteRequest is for claude-md, claude-settings, skill create/update.
type ContentWriteRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Summary string `json:"summary,omitempty"`
}

// ContentPatchRequest is for claude-md, claude-settings, skill partial update.
type ContentPatchRequest struct {
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

// MemberWriteRequest is for member create/update.
type MemberWriteRequest struct {
	Name           string               `json:"name"`
	Command        string               `json:"command"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     string               `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
	Summary        string               `json:"summary,omitempty"`
}

// MemberPatchRequest is for member partial update.
type MemberPatchRequest struct {
	Name           *string              `json:"name,omitempty"`
	Command        *string              `json:"command,omitempty"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     *string              `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
	Summary        *string              `json:"summary,omitempty"`
}

// ResourceRefRequest is a lightweight reference for linking resources.
type ResourceRefRequest struct {
	ID      int64 `json:"id"`
	Version int   `json:"version"`
}

// TeamWriteRequest is for team create/update.
type TeamWriteRequest struct {
	Name        string               `json:"name"`
	TeamMembers []TeamMemberRequest  `json:"team_members"`
	Relations   []TeamRelationRequest `json:"relations"`
	Summary     string               `json:"summary,omitempty"`
}

// TeamPatchRequest is for team partial update.
type TeamPatchRequest struct {
	Name        *string               `json:"name,omitempty"`
	TeamMembers []TeamMemberRequest   `json:"team_members,omitempty"`
	Relations   []TeamRelationRequest `json:"relations,omitempty"`
	Summary     *string               `json:"summary,omitempty"`
}

// TeamMemberRequest is a member reference in a team write.
type TeamMemberRequest struct {
	MemberID      int64 `json:"member_id"`
	MemberVersion int   `json:"member_version"`
}

// TeamRelationRequest is a relation in a team write.
// from/to are member resource IDs.
type TeamRelationRequest struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// --- Upstream Types ---

// UpstreamStatusResponse is the upstream fork status.
type UpstreamStatusResponse struct {
	Status                string `json:"status"`
	ForkVersion           int    `json:"fork_version"`
	UpstreamName          string `json:"upstream_name,omitempty"`
	UpstreamOwner         string `json:"upstream_owner,omitempty"`
	UpstreamLatestVersion *int   `json:"upstream_latest_version,omitempty"`
}

// RefUpstreamStatusResponse is the upstream status of a single ref.
type RefUpstreamStatusResponse struct {
	RelType       string `json:"rel_type"`
	TargetID      int64  `json:"target_id"`
	TargetName    string `json:"target_name"`
	TargetOwner   string `json:"target_owner"`
	TargetVersion int    `json:"target_version"`
	LatestVersion int    `json:"latest_version"`
	Status        string `json:"status"`
}

// --- Org Types ---

// CreateOrgRequest is for creating an organization.
type CreateOrgRequest struct {
	Name string `json:"name"`
}

// OrgResponse is the organization response.
type OrgResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Visibility int       `json:"visibility"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// OrgMemberResponse is an organization member.
type OrgMemberResponse struct {
	UserID int64 `json:"user_id"`
	Role   int   `json:"role"`
}

// InviteMemberRequest is for inviting a user to an org.
type InviteMemberRequest struct {
	Name string `json:"name"`
	Role int    `json:"role"`
}

// --- Resource Kind ---

// ResourceKind is the URL path segment for a resource type.
type ResourceKind string

const (
	KindMember         ResourceKind = "members"
	KindTeam           ResourceKind = "teams"
	KindSkill          ResourceKind = "skills"
	KindClaudeMd       ResourceKind = "claude-mds"
	KindClaudeSettings ResourceKind = "claude-settings"
)

// ListOptions configures resource list queries.
type ListOptions struct {
	Kind   string
	Query  string
	Limit  int
	Offset int
}
```

- [ ] **Step 2: Delete old types_test.go**

```bash
rm internal/adapter/api/types_test.go
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/adapter/api/...`
Expected: Compilation errors from files referencing old types — this is expected, we fix them in subsequent tasks.

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/api/types.go
git rm internal/adapter/api/types_test.go
git commit -m "refactor: rewrite API types for unified ResourceResponse"
```

---

### Task 2: Rewrite API client — resources.go, delete per-type files

**Files:**
- Create: `internal/adapter/api/resources.go`
- Delete: `internal/adapter/api/member.go`
- Delete: `internal/adapter/api/team.go`
- Delete: `internal/adapter/api/skill.go`
- Delete: `internal/adapter/api/claude_md.go`
- Delete: `internal/adapter/api/claude_settings.go`
- Delete: `internal/adapter/api/explore.go`

- [ ] **Step 1: Create resources.go**

```go
package api

import "fmt"

// --- Unified Read ---

func (c *Client) GetResource(owner, name string) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s", owner, name), &r)
}

func (c *Client) ListResources(owner string, opts ListOptions) (*ListResponse, error) {
	var r ListResponse
	path := fmt.Sprintf("/api/v1/orgs/%s/resources?%s", owner, buildListQuery(opts))
	return &r, c.get(path, &r)
}

func (c *Client) ListPublicResources(opts ListOptions) (*ListResponse, error) {
	var r ListResponse
	path := fmt.Sprintf("/api/v1/resources?%s", buildListQuery(opts))
	return &r, c.get(path, &r)
}

func (c *Client) ListResourceVersions(owner, name string) ([]ResourceVersionResponse, error) {
	var r []ResourceVersionResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/versions", owner, name), &r)
}

func (c *Client) GetResourceVersion(owner, name string, version int) (*ResourceVersionResponse, error) {
	var r ResourceVersionResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/versions/%d", owner, name, version), &r)
}

// --- Generic Write ---

func (c *Client) CreateResource(kind ResourceKind, owner string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/%s", owner, kind), body, &r)
}

func (c *Client) UpdateResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind, name), body, &r)
}

func (c *Client) PatchResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.patch(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind, name), body, &r)
}

func (c *Client) DeleteResource(kind ResourceKind, owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind, name))
}

func (c *Client) ForkResource(kind ResourceKind, owner, name string) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/%s/%s/fork", owner, kind, name), nil, &r)
}

// --- Upstream ---

func (c *Client) GetUpstreamStatus(owner, name string) (*UpstreamStatusResponse, error) {
	var r UpstreamStatusResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/upstream", owner, name), &r)
}

func (c *Client) GetRefsUpstreamStatus(owner, name string) ([]RefUpstreamStatusResponse, error) {
	var r []RefUpstreamStatusResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/refs-upstream", owner, name), &r)
}

// --- Helpers ---

func buildListQuery(opts ListOptions) string {
	q := ""
	sep := ""
	if opts.Kind != "" {
		q += sep + "kind=" + opts.Kind
		sep = "&"
	}
	if opts.Query != "" {
		q += sep + "q=" + opts.Query
		sep = "&"
	}
	if opts.Limit > 0 {
		q += sep + fmt.Sprintf("limit=%d", opts.Limit)
		sep = "&"
	}
	if opts.Offset > 0 {
		q += sep + fmt.Sprintf("offset=%d", opts.Offset)
	}
	return q
}
```

- [ ] **Step 2: Delete per-type files**

```bash
rm internal/adapter/api/member.go
rm internal/adapter/api/team.go
rm internal/adapter/api/skill.go
rm internal/adapter/api/claude_md.go
rm internal/adapter/api/claude_settings.go
rm internal/adapter/api/explore.go
```

- [ ] **Step 3: Verify API package compiles**

Run: `go build ./internal/adapter/api/...`
Expected: PASS (api package is self-contained now)

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/api/resources.go
git rm internal/adapter/api/member.go internal/adapter/api/team.go internal/adapter/api/skill.go internal/adapter/api/claude_md.go internal/adapter/api/claude_settings.go internal/adapter/api/explore.go
git commit -m "refactor: replace per-type API methods with unified resource client"
```

---

### Task 3: Add org.go API client

**Files:**
- Create: `internal/adapter/api/org.go`

- [ ] **Step 1: Create org.go**

```go
package api

import "fmt"

func (c *Client) CreateOrg(body CreateOrgRequest) (*OrgResponse, error) {
	var r OrgResponse
	return &r, c.post("/api/v1/orgs", body, &r)
}

func (c *Client) DeleteOrg(owner string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s", owner))
}

func (c *Client) ListMyOrgs() ([]OrgResponse, error) {
	var r []OrgResponse
	return r, c.get("/api/v1/user/orgs", &r)
}

func (c *Client) ListOrgMembers(owner string) ([]OrgMemberResponse, error) {
	var r []OrgMemberResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/org-members", owner), &r)
}

func (c *Client) InviteOrgMember(owner string, body InviteMemberRequest) error {
	return c.post(fmt.Sprintf("/api/v1/orgs/%s/org-members", owner), body, nil)
}

func (c *Client) RemoveOrgMember(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/org-members/%s", owner, name))
}
```

- [ ] **Step 2: Update auth.go — add Logout + fix UserResponse fields**

Update `internal/adapter/api/auth.go`:
- Add `Logout()` method
- Change `UserResponse.Login` to `Name`
- Add `Type`, `Visibility` fields

```go
package api

import "time"

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DevicePollResponse struct {
	Status      string        `json:"status"`
	AccessToken string        `json:"access_token"`
	User        *UserResponse `json:"user"`
}

type UserResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	GitHubID   *int64    `json:"github_id,omitempty"`
	Type       int       `json:"type"`
	Visibility int       `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	var r DeviceCodeResponse
	return &r, c.post("/auth/device", nil, &r)
}

func (c *Client) PollDeviceAuth(deviceCode string) (*DevicePollResponse, error) {
	var r DevicePollResponse
	body := map[string]string{"device_code": deviceCode}
	return &r, c.post("/auth/device/poll", body, &r)
}

func (c *Client) GetCurrentUser() (*UserResponse, error) {
	var r UserResponse
	return &r, c.get("/api/v1/user", &r)
}

func (c *Client) Logout() error {
	return c.post("/api/v1/auth/logout", nil, nil)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/adapter/api/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/adapter/api/org.go internal/adapter/api/auth.go
git commit -m "feat: add org API client and auth logout"
```

---

### Task 4: Update domain models

**Files:**
- Modify: `internal/domain/member.go`
- Modify: `internal/domain/member_test.go`
- Modify: `internal/domain/team.go`
- Modify: `internal/domain/team_test.go`

- [ ] **Step 1: Remove AgentType from Member**

In `internal/domain/member.go`:
- Remove `AgentType string` field from `Member` struct
- Remove `agentType` parameter from `NewMember()` signature
- Remove `agentType` trimming and validation

- [ ] **Step 2: Update Team — remove RootTeamMemberID, change Relation to ID-based**

In `internal/domain/team.go`:
- Remove `RootTeamMemberID *int64` field from `Team` struct
- Change `Relation` struct: `FromTeamMemberID`/`ToTeamMemberID` → `From`/`To` (int64, member resource IDs)
- Update all methods that reference these fields

- [ ] **Step 3: Update tests**

Update `internal/domain/member_test.go` — remove agentType from all `NewMember()` calls.
Update `internal/domain/team_test.go` — update Relation field names, remove RootTeamMemberID references.

- [ ] **Step 4: Verify tests pass**

Run: `go test ./internal/domain/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/member.go internal/domain/member_test.go internal/domain/team.go internal/domain/team_test.go
git commit -m "refactor: remove AgentType from Member, update Team relations to ID-based"
```

---

### Task 5: Update workspace projection types

**Files:**
- Modify: `internal/app/workspace/projection.go`
- Modify: `internal/app/workspace/snapshots.go`

- [ ] **Step 1: Update projection types**

In `internal/app/workspace/projection.go`:
- `MemberProjection`: remove `AgentType` field
- `TeamProjection`: remove `RootTeamMemberID` field
- `TeamMemberProjection`: remove `TeamMemberID`, add `MemberID int64`, `MemberVersion int`
- `TeamRelationProjection`: change `FromTeamMemberID`/`ToTeamMemberID` → `From`/`To` (int64)

- [ ] **Step 2: Update snapshots.go**

Rewrite `internal/app/workspace/snapshots.go` to use generic `decodeSnapshot[T]`:

```go
package workspace

import "encoding/json"

func decodeSnapshot[T any](snapshot json.RawMessage) (*T, error) {
	var s T
	return &s, json.Unmarshal(snapshot, &s)
}
```

Remove old `memberSnapshot`, `loadVersionedContent`, `loadMemberSnapshot`, `memberResponseFromSnapshot` — these will be replaced by `DecodeSpec` usage in materialization code.

- [ ] **Step 3: Commit**

```bash
git add internal/app/workspace/projection.go internal/app/workspace/snapshots.go
git commit -m "refactor: update projection types and unify snapshot decoding"
```

---

### Task 6: Rewrite workspace service — split god file + adapt to unified API

**Files:**
- Rewrite: `internal/app/workspace/service.go` (keep struct + constructor only)
- Create: `internal/app/workspace/clone.go` (split from service.go)
- Create: `internal/app/workspace/push.go` (split from service.go)
- Create: `internal/app/workspace/pull.go` (split from service.go)
- Create: `internal/app/workspace/wk_status.go` (split from service.go, avoid name clash with cmd/status.go)
- Create: `internal/app/workspace/materialize.go` (split from service.go)
- Modify: `internal/app/workspace/writer.go`
- Modify: `internal/app/workspace/upstream.go`
- Modify: `internal/app/workspace/manifest.go`

This is the largest task. The approach:
1. Keep `Service` struct and `NewService()` in `service.go`
2. Move each responsibility to its own file
3. Adapt all methods to use `ResourceResponse` instead of per-type responses
4. Replace Push switch-case with unified dispatch

- [ ] **Step 1: Rewrite service.go — struct only**

Trim `internal/app/workspace/service.go` to just the struct and constructor (~30 lines).

- [ ] **Step 2: Create clone.go**

Move `CloneMember`, `CloneTeam` from service.go. Update to use `client.GetResource()` → `DecodeSpec` instead of per-type getters.

- [ ] **Step 3: Create push.go**

Move Push logic. Replace 95-line switch-case with unified flow:
1. Load raw projection
2. `client.GetResource()` for version check
3. `buildMutationFromProjection(kind, data)` — only this branches by kind
4. `client.UpdateResource(kind, ...)`

- [ ] **Step 4: Create pull.go**

Move Pull, PullForce logic. Update to use unified API.

- [ ] **Step 5: Create wk_status.go**

Move Status, ModifiedResources logic.

- [ ] **Step 6: Create materialize.go**

Move `materializeMember`, `materializeTeam` and projection builders. Update to accept `*api.ResourceResponse`.

- [ ] **Step 7: Update writer.go**

Update `MaterializeMemberFiles`, `MaterializeTeamFiles` to work with `ResourceResponse` + `DecodeSpec` + ref filtering by `rel_type`.

- [ ] **Step 8: Update upstream.go**

Replace per-type version fetching with `client.GetResourceVersion()`. Use `client.GetUpstreamStatus()` and `client.GetRefsUpstreamStatus()`.

- [ ] **Step 9: Verify compilation**

Run: `go build ./internal/app/workspace/...`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/app/workspace/
git commit -m "refactor: split workspace service god file, adapt to unified API"
```

---

### Task 7: Rewrite cmd/ — unified resource commands

**Files:**
- Create: `cmd/resource.go`
- Create: `cmd/resource_specs.go`
- Delete: `cmd/member.go`, `cmd/team.go`, `cmd/skill.go`, `cmd/claudemd.go`, `cmd/claudesettings.go`
- Delete: `cmd/explore.go`
- Delete: `cmd/resource_lookup.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create resource_specs.go — per-kind configuration**

```go
package cmd

import (
	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/spf13/cobra"
)

type resourceSpec struct {
	Kind       api.ResourceKind
	Singular   string
	BuildWrite func(cmd *cobra.Command) (any, error)
	BuildPatch func(cmd *cobra.Command) (any, error)
	AddFlags   func(createCmd, editCmd *cobra.Command)
}
```

Define 5 specs: contentSpec (shared by claude-md, claude-settings, skill), memberSpec, teamSpec. Each spec implements BuildWrite, BuildPatch, AddFlags.

- [ ] **Step 2: Create resource.go — unified command factory**

```go
func newResourceCmd(spec resourceSpec) *cobra.Command
```

Generates: create, edit, delete, get, list, versions subcommands. Each uses `client.CreateResource()`, `client.PatchResource()`, `client.DeleteResource()`, `client.GetResource()`, `client.ListResources()`, `client.ListResourceVersions()`.

Also include the unified `resourceExistsOnServer()` helper.

- [ ] **Step 3: Delete old per-type command files**

```bash
rm cmd/member.go cmd/team.go cmd/skill.go cmd/claudemd.go cmd/claudesettings.go cmd/explore.go cmd/resource_lookup.go
```

- [ ] **Step 4: Update root.go**

Register 5 resource commands via `newResourceCmd(spec)` + update command groups. Remove old command registrations.

- [ ] **Step 5: Update clone.go, fork.go**

Update to use unified API: `client.GetResource()`, `client.ForkResource()`.

- [ ] **Step 6: Update fetch.go, merge.go, diff.go**

Update to use unified API.

- [ ] **Step 7: Update helpers.go**

Remove references to old API types. Update run planning to work without AgentType on Member.

- [ ] **Step 8: Verify compilation**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add cmd/
git rm cmd/member.go cmd/team.go cmd/skill.go cmd/claudemd.go cmd/claudesettings.go cmd/explore.go cmd/resource_lookup.go
git commit -m "refactor: replace per-type commands with unified resource command factory"
```

---

### Task 8: Add org commands

**Files:**
- Create: `cmd/org.go`
- Modify: `cmd/root.go` (register org command)

- [ ] **Step 1: Create cmd/org.go**

Implement org subcommands: create, delete, list, members, invite, remove.

- [ ] **Step 2: Register in root.go**

Add `newOrgCmd()` to Settings group.

- [ ] **Step 3: Add auth logout subcommand**

In `cmd/auth.go`, add `logout` subcommand that calls `client.Logout()` + deletes local credentials.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/org.go cmd/auth.go cmd/root.go
git commit -m "feat: add org commands and auth logout"
```

---

### Task 9: Update resource_kinds.go and working_copy files

**Files:**
- Modify: `cmd/resource_kinds.go`
- Modify: `cmd/working_copy_validation.go`
- Modify: `cmd/working_copy_paths.go`
- Modify: `cmd/working_copy_helpers.go`

- [ ] **Step 1: Update resource_kinds.go**

Align kind constants with `api.ResourceKind` values.

- [ ] **Step 2: Update working copy files**

Update any references to old API types or field names (owner_login → owner_name, etc.).

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/
git commit -m "refactor: align working copy helpers with unified API"
```

---

### Task 10: Update and add tests

**Files:**
- Create: `internal/adapter/api/types_test.go`
- Modify: `cmd/api_contracts_test.go`
- Modify: `cmd/root_test.go`
- Modify: `cmd/helpers_test.go`
- Modify: `cmd/working_copy_validation_test.go`
- Modify: `cmd/working_copy_paths_test.go`

- [ ] **Step 1: Write API types tests**

Test `DecodeSpec` with each kind, `ResourceResponse` JSON roundtrip, `ListResponse` deserialization.

- [ ] **Step 2: Update cmd tests**

Fix all test files to use new API types and command structure.

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: update tests for unified resource API"
```

---

### Task 11: Final verification and cleanup

- [ ] **Step 1: Full build**

Run: `go build ./...`
Expected: PASS, no warnings

- [ ] **Step 2: Full test suite**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 3: Verify no dead code**

Check for any unused imports, unreferenced types, orphaned files.

- [ ] **Step 4: Final commit if needed**

```bash
git add -A
git commit -m "chore: cleanup after unified API migration"
```
