# Unified Resource API Migration — Big-Bang Design Spec

**Date**: 2026-04-13
**Status**: Approved
**Scope**: Full migration to unified ResourceResponse API + structural refactoring + Organization feature

## Context

clier-server underwent a major API overhaul:

1. All resources now return a unified `ResourceResponse{kind, metadata, spec, refs}` instead of per-type response types
2. Read endpoints unified: `GET /api/v1/orgs/{owner}/resources/{name}` replaces per-type GETs
3. Write endpoints remain per-type but with schema changes
4. Organization system added
5. Resource names are flat-unique per owner (kind-agnostic)
6. `agent_type` removed from write requests (server-derived)
7. Team `root_index` removed
8. Team relations use member resource IDs (not indices) — `from`/`to` match `refs[].target_id`
9. `spec` field formally typed as `oneOf [MemberSpec, TeamSpec, ContentSpec]`

This is a big-bang migration: all layers (API adapter, domain, workspace, cmd) change together. The migration also resolves accumulated structural debt.

## Design Decisions

- **Approach B**: Generic Resource Client + unified Response — eliminates O(N) code duplication via Go generics and template patterns
- **Organization**: Included in this migration as a new `org` command group
- **Explore subcommand absorbed**: `explore <kind> <owner/name>` becomes `<kind> get <owner/name>`, `<kind> list`, etc.

---

## 1. Unified API Types

### File: `internal/adapter/api/types.go`

Replace 10 per-type response structs with unified types.

#### Core Response Types

```go
// ResourceResponse — unified response for all resource kinds.
type ResourceResponse struct {
    Kind       string            `json:"kind"`
    Metadata   ResourceMetadata  `json:"metadata"`
    Spec       json.RawMessage   `json:"spec"`
    Refs       []ResolvedRef     `json:"refs"`
    AgentTypes []string          `json:"agent_types"`
}

// ResourceMetadata — shared metadata across all resource kinds.
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

// ResolvedRef — resolved reference between resources.
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

// ListResponse — paginated list wrapper.
type ListResponse struct {
    Items []ResourceResponse `json:"items"`
    Total int                `json:"total"`
}

// ResourceVersionResponse — unified version response.
type ResourceVersionResponse struct {
    ID             int64           `json:"id"`
    ResourceID     int64           `json:"resource_id"`
    Version        int             `json:"version"`
    Snapshot       json.RawMessage `json:"snapshot"`
    CreatedAt      time.Time       `json:"created_at"`
    OwnerName      string          `json:"owner_name"`
    OwnerAvatarURL string          `json:"owner_avatar_url,omitempty"`
}
```

#### Spec Types (formally typed via `oneOf` in OpenAPI schema)

The server's `spec` field is a discriminated union (`oneOf`) keyed by `kind`:
- `member` → `MemberSpec`
- `team` → `TeamSpec`
- `skill`, `claude_md`, `claude_setting` → `ContentSpec`

```go
// ContentSpec — spec for claude-md, claude-settings, skill.
// Contains only content (name lives in metadata).
type ContentSpec struct {
    Content string `json:"content"`
}

// MemberSpec — spec for member (read-only fields included).
// agent_type is server-derived and returned in response, not sent in write requests.
type MemberSpec struct {
    AgentType  string `json:"agent_type"`
    Command    string `json:"command"`
    GitRepoURL string `json:"git_repo_url"`
}

// TeamSpec — spec for team.
// Team members are represented in refs (not in spec).
// Only relations are stored in spec.
type TeamSpec struct {
    Relations []TeamRelation `json:"relations"`
}

// TeamRelation — relation in team spec response.
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
```

Key observations:
- `name` is NOT in any spec — it lives in `metadata.name`
- `agent_type` appears in `MemberSpec` (response) but not in `MemberWriteRequest` (server-derived)
- Team members are resolved into `refs` array, not stored in `spec` — spec only holds `relations`

#### Request Types

```go
// ContentWriteRequest — for claude-md, claude-settings, skill create/update.
type ContentWriteRequest struct {
    Name    string `json:"name"`
    Content string `json:"content"`
    Summary string `json:"summary,omitempty"`
}

// ContentPatchRequest — for claude-md, claude-settings, skill partial update.
type ContentPatchRequest struct {
    Name    *string `json:"name,omitempty"`
    Content *string `json:"content,omitempty"`
    Summary *string `json:"summary,omitempty"`
}

// MemberWriteRequest — for member create/update.
type MemberWriteRequest struct {
    Name           string               `json:"name"`
    Command        string               `json:"command"`
    Skills         []ResourceRefRequest `json:"skills,omitempty"`
    GitRepoURL     string               `json:"git_repo_url,omitempty"`
    ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
    ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
    Summary        string               `json:"summary,omitempty"`
}

// MemberPatchRequest — for member partial update.
type MemberPatchRequest struct {
    Name           *string              `json:"name,omitempty"`
    Command        *string              `json:"command,omitempty"`
    Skills         []ResourceRefRequest `json:"skills,omitempty"`
    GitRepoURL     *string              `json:"git_repo_url,omitempty"`
    ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
    ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
    Summary        *string              `json:"summary,omitempty"`
}

// ResourceRefRequest — lightweight reference for linking resources.
type ResourceRefRequest struct {
    ID      int64 `json:"id"`
    Version int   `json:"version"`
}

// TeamWriteRequest — for team create/update.
type TeamWriteRequest struct {
    Name        string               `json:"name"`
    TeamMembers []TeamMemberRequest  `json:"team_members"`
    Relations   []TeamRelationRequest `json:"relations"`
    Summary     string               `json:"summary,omitempty"`
}

// TeamPatchRequest — for team partial update.
type TeamPatchRequest struct {
    Name        *string               `json:"name,omitempty"`
    TeamMembers []TeamMemberRequest   `json:"team_members,omitempty"`
    Relations   []TeamRelationRequest `json:"relations,omitempty"`
    Summary     *string               `json:"summary,omitempty"`
}

// TeamMemberRequest — member reference in a team write.
type TeamMemberRequest struct {
    MemberID      int64 `json:"member_id"`
    MemberVersion int   `json:"member_version"`
}

// TeamRelationRequest — relation in a team write.
// from/to are member resource IDs (same as TeamRelation in response).
type TeamRelationRequest struct {
    From int64 `json:"from"`
    To   int64 `json:"to"`
}
```

#### Upstream Types (new)

```go
type UpstreamStatusResponse struct {
    Status               string `json:"status"`
    ForkVersion          int    `json:"fork_version"`
    UpstreamName         string `json:"upstream_name,omitempty"`
    UpstreamOwner        string `json:"upstream_owner,omitempty"`
    UpstreamLatestVersion *int  `json:"upstream_latest_version,omitempty"`
}

type RefUpstreamStatusResponse struct {
    RelType       string `json:"rel_type"`
    TargetID      int64  `json:"target_id"`
    TargetName    string `json:"target_name"`
    TargetOwner   string `json:"target_owner"`
    TargetVersion int    `json:"target_version"`
    LatestVersion int    `json:"latest_version"`
    Status        string `json:"status"`
}
```

#### Org Types (new)

```go
type CreateOrgRequest struct {
    Name string `json:"name"`
}

type OrgResponse struct {
    ID         int64     `json:"id"`
    Name       string    `json:"name"`
    Visibility int       `json:"visibility"`
    AvatarURL  string    `json:"avatar_url,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

type OrgMemberResponse struct {
    UserID int64 `json:"user_id"`
    Role   int   `json:"role"`
}

type InviteMemberRequest struct {
    Name string `json:"name"`
    Role int    `json:"role"`
}
```

#### Auth Types

Existing types retained. `UserResponse` field changes:

- `Login` → `Name` (matches server's `name` field)
- `Type`, `Visibility` fields added

#### Deleted Types

All of these are removed:

- `MemberResponse`, `MemberVersionResponse`
- `TeamResponse`, `TeamMemberResponse`, `TeamRelationResponse`, `TeamVersionResponse`
- `SkillResponse`, `SkillVersionResponse`
- `ClaudeMdResponse`, `ClaudeMdVersionResponse`
- `ClaudeSettingsResponse`, `ClaudeSettingsVersionResponse`
- `ResourceRef`, `MemberRef` (replaced by `ResolvedRef`)

---

## 2. Generic Resource Client

### File structure

```
internal/adapter/api/
├── client.go          # HTTP core (unchanged)
├── types.go           # All types from section 1
├── resources.go       # Unified read + generic write methods
├── auth.go            # Auth endpoints (+ logout)
├── org.go             # Org CRUD (new)
└── upstream.go        # Upstream status (new)
```

**Deleted files**: `member.go`, `team.go`, `skill.go`, `claude_md.go`, `claude_settings.go`, `explore.go`

### resources.go — Unified Read

```go
func (c *Client) GetResource(owner, name string) (*ResourceResponse, error)
func (c *Client) ListResources(owner string, opts ListOptions) (*ListResponse, error)
func (c *Client) ListPublicResources(opts ListOptions) (*ListResponse, error)
func (c *Client) ListResourceVersions(owner, name string) ([]ResourceVersionResponse, error)
func (c *Client) GetResourceVersion(owner, name string, version int) (*ResourceVersionResponse, error)

type ListOptions struct {
    Kind   string
    Query  string
    Limit  int
    Offset int
}
```

### resources.go — Generic Write

```go
type ResourceKind string

const (
    KindMember         ResourceKind = "members"
    KindTeam           ResourceKind = "teams"
    KindSkill          ResourceKind = "skills"
    KindClaudeMd       ResourceKind = "claude-mds"
    KindClaudeSettings ResourceKind = "claude-settings"
)

// Note: ResourceKind values are plural (URL path segments for write endpoints).
// The server's ResourceResponse.Kind field may use a different form.
// Maintain a bidirectional map between the two — exact server kind values
// to be confirmed during implementation by inspecting actual responses.

func (c *Client) CreateResource(kind ResourceKind, owner string, body any) (*ResourceResponse, error)
func (c *Client) UpdateResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error)
func (c *Client) PatchResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error)
func (c *Client) DeleteResource(kind ResourceKind, owner, name string) error
func (c *Client) ForkResource(kind ResourceKind, owner, name string) (*ResourceResponse, error)
```

### auth.go — Logout addition

```go
func (c *Client) Logout() error  // POST /api/v1/auth/logout
```

### org.go — Organization endpoints

```go
func (c *Client) CreateOrg(body CreateOrgRequest) (*OrgResponse, error)
func (c *Client) DeleteOrg(owner string) error
func (c *Client) ListMyOrgs() ([]OrgResponse, error)
func (c *Client) ListOrgMembers(owner string) ([]OrgMemberResponse, error)
func (c *Client) InviteOrgMember(owner string, body InviteMemberRequest) error
func (c *Client) RemoveOrgMember(owner, name string) error
```

### upstream.go — Upstream status

```go
func (c *Client) GetUpstreamStatus(owner, name string) (*UpstreamStatusResponse, error)
func (c *Client) GetRefsUpstreamStatus(owner, name string) ([]RefUpstreamStatusResponse, error)
```

### Result

45 per-type methods → 12 unified methods + 6 org methods + 2 upstream methods = 20 total. Net reduction: ~25 methods.

---

## 3. cmd/ Restructure

### File structure

```
cmd/
├── root.go                    # Main CLI setup (updated group structure)
├── resource.go                # Unified CRUD + read command factory (new)
├── resource_specs.go          # Per-kind spec definitions (new)
├── org.go                     # Org commands (new)
├── auth.go                    # Auth commands (+ logout)
├── run.go                     # Runtime commands (unchanged)
├── clone.go                   # Clone (updated to unified API)
├── fork.go                    # Fork (updated to unified API)
├── push.go                    # Push (unchanged interface, updated internals)
├── pull.go                    # Pull (unchanged)
├── status.go                  # Status (unchanged)
├── fetch.go                   # Fetch (updated to unified API)
├── merge.go                   # Merge (updated)
├── diff.go                    # Diff (updated)
├── config.go                  # Config (unchanged)
├── tutorial.go                # Tutorial (unchanged)
├── helpers.go                 # Helpers (updated)
├── working_copy_validation.go # Manifest validation (unchanged)
├── working_copy_paths.go      # Path resolution (unchanged)
├── output.go                  # JSON output (unchanged)
├── agent_env.go               # Agent env constants (unchanged)
└── resource_kinds.go          # Kind constants (updated)
```

**Deleted files**: `member.go`, `team.go`, `skill.go`, `claudemd.go`, `claudesettings.go`, `explore.go`, `resource_lookup.go` (absorbed into resource.go)

### resource_specs.go — Per-kind configuration

```go
type resourceSpec struct {
    Kind      api.ResourceKind
    Singular  string                                   // CLI command name
    BuildWrite func(cmd *cobra.Command) (any, error)   // Flags → WriteRequest
    BuildPatch func(cmd *cobra.Command) (any, error)   // Flags → PatchRequest
    AddFlags   func(createCmd, editCmd *cobra.Command)  // Register kind-specific flags
}
```

Five specs defined: member, team, skill, claude-md, claude-settings.

- claude-md, claude-settings, skill share `ContentWriteRequest`/`ContentPatchRequest` — flags: `--name`, `--content`, `--summary`
- member: flags: `--name`, `--command`, `--git-repo-url`, `--claude-md`, `--claude-settings`, `--skill`
- team: flags: `--name`, `--member` (repeatable), `--relation` (repeatable), `--summary`

### resource.go — Unified command factory

```go
func newResourceCmd(spec resourceSpec) *cobra.Command
```

Generates subcommands per resource:

```
clier <resource> create [flags]
clier <resource> edit <name> [flags]
clier <resource> delete <name>
clier <resource> get <owner/name>
clier <resource> list [owner] [--kind ...] [--query ...]
clier <resource> versions <owner/name>
```

Explore subcommand removed — `get`, `list`, `versions` live under each resource command.

### resource_lookup.go elimination

```go
// Single function replaces 5 per-type *ExistsOnServer() functions.
func resourceExistsOnServer(client *api.Client, owner, name string) (bool, error)
```

### org.go — Organization commands

```
clier org create <name>
clier org delete <name>
clier org list
clier org members <org-name>
clier org invite <org-name> <user-name> --role <role>
clier org remove <org-name> <user-name>
```

### auth.go — Logout addition

```
clier auth logout
```

Calls `client.Logout()` + deletes local credentials file.

### root.go — Updated command groups

```
Resources:  member, team, skill, claude-md, claude-settings  (each with create/edit/delete/get/list/versions)
Runtime:    run, clone, fork
Workspace:  push, pull, status, fetch, merge, diff
Settings:   auth, config, org
Discovery:  tutorial
```

---

## 4. Workspace/Service Refactoring

### File split from service.go (798 lines)

```
internal/app/workspace/
├── service.go         # Service struct + constructor (~50 lines)
├── clone.go           # CloneMember, CloneTeam (~120 lines)
├── push.go            # Push with unified kind dispatch (~80 lines)
├── pull.go            # Pull, PullForce (~60 lines)
├── status.go          # Status, ModifiedResources (~80 lines)
├── materialize.go     # materializeMember, materializeTeam (~200 lines)
├── projection.go      # Projection types (updated)
├── snapshots.go       # Unified snapshot decoding
├── upstream.go        # Upstream sync (updated to unified API)
├── manifest.go        # Manifest lifecycle (unchanged)
├── writer.go          # File materialization (updated to unified API)
├── protocol.go        # Protocol generation (unchanged)
├── layout.go          # Directory naming (unchanged)
├── port.go            # Interface definitions (unchanged)
└── git_repo.go        # Git abstraction (unchanged)
```

### Push — unified dispatch

Existing 95-line switch-case over 5 kinds replaced with generic flow:

1. Load raw projection (kind-agnostic JSON file)
2. Get remote state via `client.GetResource()`
3. Version conflict check
4. Build mutation via `buildMutationFromProjection(kind, data)` — only this step branches by kind
5. Update via `client.UpdateResource(kind, ...)`

### Projection type changes

- `MemberProjection`: `AgentType` field removed
- `TeamProjection`: `RootTeamMemberID` field removed
- `TeamMemberProjection`: `TeamMemberID` removed, `MemberID` + `MemberVersion` added
- `TeamRelationProjection`: `FromTeamMemberID`/`ToTeamMemberID` → `From`/`To` (member resource IDs)

### Snapshot deserialization — unified

Replace asymmetric member/team snapshot handling with generic `decodeSnapshot[T]()`:

```go
func decodeSnapshot[T any](snapshot json.RawMessage) (*T, error) {
    var s T
    return &s, json.Unmarshal(snapshot, &s)
}
```

Both member and team snapshots use the same decode path.

### Writer — updated for ResourceResponse

`writer.go` methods updated to accept `*api.ResourceResponse` instead of per-type responses. Spec and refs extracted via `DecodeSpec[T]()` and ref filtering by `rel_type`.

---

## 5. Domain Model Changes

Minimal changes due to clean hexagonal separation:

- **Member**: Remove `AgentType` field and `agentType` parameter from `NewMember()`
- **Team**: Remove `RootTeamMemberID` field. `Relation` fields change from `FromTeamMemberID`/`ToTeamMemberID` to `From`/`To` (member resource IDs)
- **Run**: `AgentProfile` lookup changes from agent-type-based to command-based
- **resource/**: ClaudeMd, ClaudeSettings, Skill — no changes

---

## 6. Auth Extension

- API client: `Logout()` method — `POST /api/v1/auth/logout`
- cmd: `auth logout` subcommand — server token revoke + local credentials file deletion

---

## 7. Test Strategy

- **API client tests**: Mock unified `ResourceResponse` via `httptest.Server`. Replace per-type test suites with unified tests.
- **Workspace tests**: Migrate to `ResourceResponse`-based fixtures. Focus on Push unified dispatch logic.
- **cmd tests**: Update `api_contracts_test.go` to new API schema.
- **Approach**: Delete existing tests and rewrite — big-bang migration makes incremental test fixes impractical.

---

## Summary: Files Changed

### New files
- `internal/adapter/api/resources.go`
- `internal/adapter/api/org.go`
- `internal/adapter/api/upstream.go`
- `cmd/resource.go`
- `cmd/resource_specs.go`
- `cmd/org.go`
- `internal/app/workspace/clone.go` (split from service.go)
- `internal/app/workspace/push.go` (split from service.go)
- `internal/app/workspace/pull.go` (split from service.go)
- `internal/app/workspace/status.go` (split from service.go)
- `internal/app/workspace/materialize.go` (split from service.go)

### Deleted files
- `internal/adapter/api/member.go`
- `internal/adapter/api/team.go`
- `internal/adapter/api/skill.go`
- `internal/adapter/api/claude_md.go`
- `internal/adapter/api/claude_settings.go`
- `internal/adapter/api/explore.go`
- `cmd/member.go`
- `cmd/team.go`
- `cmd/skill.go`
- `cmd/claudemd.go`
- `cmd/claudesettings.go`
- `cmd/explore.go`
- `cmd/resource_lookup.go`

### Heavily modified files
- `internal/adapter/api/types.go` — complete rewrite
- `internal/adapter/api/auth.go` — logout addition
- `internal/app/workspace/service.go` — split into multiple files
- `internal/app/workspace/projection.go` — field changes
- `internal/app/workspace/snapshots.go` — unified decode
- `internal/app/workspace/writer.go` — ResourceResponse adaptation
- `internal/app/workspace/upstream.go` — unified API
- `internal/domain/member.go` — AgentType removal
- `internal/domain/team.go` — RootTeamMemberID removal, relation index change
- `cmd/root.go` — command group restructure
- `cmd/auth.go` — logout subcommand
- `cmd/fork.go` — unified API
- `cmd/clone.go` — unified API
- `cmd/helpers.go` — updated helpers
