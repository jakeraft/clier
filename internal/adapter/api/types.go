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

// --- Spec Types ---

// ContentSpec is the spec for claude-md, claude-settings, skill.
type ContentSpec struct {
	Content string `json:"content"`
}

// MemberSpec is the spec for member (response-only fields included).
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

type ContentWriteRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Summary string `json:"summary,omitempty"`
}

type ContentPatchRequest struct {
	Name    *string `json:"name,omitempty"`
	Content *string `json:"content,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

type MemberWriteRequest struct {
	Name           string               `json:"name"`
	Command        string               `json:"command"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     string               `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
	Summary        string               `json:"summary,omitempty"`
}

type MemberPatchRequest struct {
	Name           *string              `json:"name,omitempty"`
	Command        *string              `json:"command,omitempty"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     *string              `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
	Summary        *string              `json:"summary,omitempty"`
}

type ResourceRefRequest struct {
	ID      int64 `json:"id"`
	Version int   `json:"version"`
}

type TeamWriteRequest struct {
	Name        string                `json:"name"`
	TeamMembers []TeamMemberRequest   `json:"team_members"`
	Relations   []TeamRelationRequest `json:"relations"`
	Summary     string                `json:"summary,omitempty"`
}

type TeamPatchRequest struct {
	Name        *string               `json:"name,omitempty"`
	TeamMembers []TeamMemberRequest   `json:"team_members,omitempty"`
	Relations   []TeamRelationRequest `json:"relations,omitempty"`
	Summary     *string               `json:"summary,omitempty"`
}

type TeamMemberRequest struct {
	MemberID      int64 `json:"member_id"`
	MemberVersion int   `json:"member_version"`
}

type TeamRelationRequest struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// --- Upstream Types ---

type UpstreamStatusResponse struct {
	Status                string `json:"status"`
	ForkVersion           int    `json:"fork_version"`
	UpstreamName          string `json:"upstream_name,omitempty"`
	UpstreamOwner         string `json:"upstream_owner,omitempty"`
	UpstreamLatestVersion *int   `json:"upstream_latest_version,omitempty"`
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

// --- Org Types ---

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

// --- Resource Kind ---

type ResourceKind string

// ResourceKind constants match the server's canonical kind values.
const (
	KindMember         ResourceKind = "member"
	KindTeam           ResourceKind = "team"
	KindSkill          ResourceKind = "skill"
	KindClaudeMd       ResourceKind = "claude_md"
	KindClaudeSettings ResourceKind = "claude_setting"
)

// urlPath returns the URL path segment for write endpoints.
func (k ResourceKind) urlPath() string {
	switch k {
	case KindMember:
		return "members"
	case KindTeam:
		return "teams"
	case KindSkill:
		return "skills"
	case KindClaudeMd:
		return "claude-mds"
	case KindClaudeSettings:
		return "claude-settings"
	default:
		return string(k)
	}
}

type ListOptions struct {
	Kind   string
	Query  string
	Limit  int
	Offset int
}
