package api

import (
	"encoding/json"
	"time"
)

// --- Response Types ---

// ResourceResponse is the unified response for all resource kinds.
type ResourceResponse struct {
	Kind           string           `json:"kind"`
	Metadata       ResourceMetadata `json:"metadata"`
	Spec           json.RawMessage  `json:"spec"`
	Refs           []ResolvedRef    `json:"refs"`
	AgentTypes     []string         `json:"agent_types"`
	VersionCreated bool             `json:"version_created,omitempty"`
}

// ResourceMetadata contains shared metadata across all resource kinds.
type ResourceMetadata struct {
	Name           string    `json:"name"`
	Summary        string    `json:"summary"`
	Visibility     int       `json:"visibility"`
	RefCount       int       `json:"ref_count"`
	StarCount      int       `json:"star_count"`
	Starred        bool      `json:"starred"`
	OwnerName      string    `json:"owner_name"`
	OwnerType      int       `json:"owner_type"`
	OwnerAvatarURL string    `json:"owner_avatar_url,omitempty"`
	LatestVersion  int       `json:"latest_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ResolvedRef is a resolved reference between resources.
type ResolvedRef struct {
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
	Version        int             `json:"version"`
	Snapshot       json.RawMessage `json:"snapshot"`
	Refs           []ResolvedRef   `json:"refs"`
	CreatedAt      time.Time       `json:"created_at"`
	OwnerName      string          `json:"owner_name"`
	OwnerAvatarURL string          `json:"owner_avatar_url,omitempty"`
}

// --- Spec Types ---

// ContentSpec is the spec for instruction (claude-md, codex-md), settings (claude-setting, codex-setting), and skill.
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

// ResourceIdentifier identifies a resource by owner and name.
type ResourceIdentifier struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// TeamRelation is a relation in team spec response.
type TeamRelation struct {
	From ResourceIdentifier `json:"from"`
	To   ResourceIdentifier `json:"to"`
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
	Content *string `json:"content,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

type MemberWriteRequest struct {
	Name           string               `json:"name"`
	Command        string               `json:"command"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     string               `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude-md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude-setting,omitempty"`
	CodexMd        *ResourceRefRequest  `json:"codex-md,omitempty"`
	CodexSettings  *ResourceRefRequest  `json:"codex-setting,omitempty"`
	Summary        string               `json:"summary,omitempty"`
}

type MemberPatchRequest struct {
	Command        *string              `json:"command,omitempty"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	GitRepoURL     *string              `json:"git_repo_url,omitempty"`
	ClaudeMd       *ResourceRefRequest  `json:"claude-md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude-setting,omitempty"`
	CodexMd        *ResourceRefRequest  `json:"codex-md,omitempty"`
	CodexSettings  *ResourceRefRequest  `json:"codex-setting,omitempty"`
	Summary        *string              `json:"summary,omitempty"`
}

type ResourceRefRequest struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type TeamWriteRequest struct {
	Name        string                `json:"name"`
	TeamMembers []TeamMemberRequest   `json:"team_members"`
	Relations   []TeamRelationRequest `json:"relations"`
	Summary     string                `json:"summary,omitempty"`
}

type TeamPatchRequest struct {
	TeamMembers []TeamMemberRequest   `json:"team_members,omitempty"`
	Relations   []TeamRelationRequest `json:"relations,omitempty"`
	Summary     *string               `json:"summary,omitempty"`
}

type TeamMemberRequest struct {
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	MemberVersion int    `json:"member_version"`
}

type TeamRelationRequest struct {
	From ResourceIdentifier `json:"from"`
	To   ResourceIdentifier `json:"to"`
}

// --- Org Types ---

type CreateOrgRequest struct {
	Name string `json:"name"`
}

type OrgResponse struct {
	Name       string    `json:"name"`
	Visibility int       `json:"visibility"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type OrgMemberResponse struct {
	Name string `json:"name"`
	Role int    `json:"role"`
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
	KindClaudeMd       ResourceKind = "claude-md"
	KindClaudeSettings ResourceKind = "claude-setting"
	KindCodexMd        ResourceKind = "codex-md"
	KindCodexSettings  ResourceKind = "codex-setting"
)

// urlPath returns the plural URL path segment for write endpoints.
var kindURLPaths = map[ResourceKind]string{
	KindMember:         "members",
	KindTeam:           "teams",
	KindSkill:          "skills",
	KindClaudeMd:       "claude-mds",
	KindClaudeSettings: "claude-settings",
	KindCodexMd:        "codex-mds",
	KindCodexSettings:  "codex-settings",
}

func (k ResourceKind) urlPath() string {
	if p, ok := kindURLPaths[k]; ok {
		return p
	}
	return string(k) + "s"
}

type ListOptions struct {
	Kind    string
	Query   string
	Uses    string
	Starred *bool
	Limit   int
	Offset  int
	Sort    string
	Order   string
}

// SetInstructionRef sets the instruction ref field matching the given kind.
func (r *MemberWriteRequest) SetInstructionRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeMd:
		r.ClaudeMd = ref
	case KindCodexMd:
		r.CodexMd = ref
	}
}

// SetSettingsRef sets the settings ref field matching the given kind.
func (r *MemberWriteRequest) SetSettingsRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeSettings:
		r.ClaudeSettings = ref
	case KindCodexSettings:
		r.CodexSettings = ref
	}
}

// SetInstructionRef sets the instruction ref field matching the given kind.
func (r *MemberPatchRequest) SetInstructionRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeMd:
		r.ClaudeMd = ref
	case KindCodexMd:
		r.CodexMd = ref
	}
}

// SetSettingsRef sets the settings ref field matching the given kind.
func (r *MemberPatchRequest) SetSettingsRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeSettings:
		r.ClaudeSettings = ref
	case KindCodexSettings:
		r.CodexSettings = ref
	}
}

// IsInstructionKind returns true for agent instruction resource kinds.
func IsInstructionKind(kind string) bool {
	switch ResourceKind(kind) {
	case KindClaudeMd, KindCodexMd:
		return true
	}
	return false
}

// IsSettingsKind returns true for agent settings resource kinds.
func IsSettingsKind(kind string) bool {
	switch ResourceKind(kind) {
	case KindClaudeSettings, KindCodexSettings:
		return true
	}
	return false
}

// IsContentKind returns true for all content-based resource kinds (instruction, settings, skill).
func IsContentKind(kind string) bool {
	return IsInstructionKind(kind) || IsSettingsKind(kind) || ResourceKind(kind) == KindSkill
}
