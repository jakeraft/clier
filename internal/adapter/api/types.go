package api

import (
	"encoding/json"
	"time"
)

// --- Response Types ---

type ResourceResponse struct {
	Kind           string           `json:"kind"`
	Metadata       ResourceMetadata `json:"metadata"`
	Spec           json.RawMessage  `json:"spec"`
	Refs           []ResolvedRef    `json:"refs"`
	VersionCreated bool             `json:"version_created,omitempty"`
}

type ResourceMetadata struct {
	Name           string    `json:"name"`
	Summary        string    `json:"summary"`
	RefCount       int       `json:"ref_count"`
	StarCount      int       `json:"star_count"`
	Starred        bool      `json:"starred"`
	OwnerName      string    `json:"owner_name"`
	OwnerType      int       `json:"owner_type"`
	OwnerAvatarURL string    `json:"owner_avatar_url,omitempty"`
	LatestVersion  int       `json:"latest_version,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ResolvedRef struct {
	TargetVersion      int    `json:"target_version"`
	RelType            string `json:"rel_type"`
	Name               string `json:"name"`
	OwnerName          string `json:"owner_name"`
	AgentType          string `json:"agent_type,omitempty"`
	Command            string `json:"command,omitempty"`
	OwnerAvatarURL     string `json:"owner_avatar_url,omitempty"`
	Deleted            bool   `json:"deleted,omitempty"`
	UnavailableReason  string `json:"unavailable_reason,omitempty"`
	UnavailableMessage string `json:"unavailable_message,omitempty"`
}

type ListResponse struct {
	Items []ResourceResponse `json:"items"`
	Total int                `json:"total"`
}

type ResourceVersionResponse struct {
	Version        int             `json:"version"`
	Snapshot       json.RawMessage `json:"snapshot"`
	Refs           []ResolvedRef   `json:"refs"`
	CreatedAt      time.Time       `json:"created_at"`
	OwnerName      string          `json:"owner_name"`
	OwnerAvatarURL string          `json:"owner_avatar_url,omitempty"`
}

// --- Resolve Types ---

type ResolveResponse struct {
	Root      ResolvedResource   `json:"root"`
	Resources []ResolvedResource `json:"resources"`
}

type ResolvedResource struct {
	Kind           string          `json:"kind"`
	OwnerName      string          `json:"owner_name"`
	Name           string          `json:"name"`
	Version        int             `json:"version"`
	Snapshot       json.RawMessage `json:"snapshot"`
	Versions       []VersionMeta   `json:"versions"`
	OwnerAvatarURL string          `json:"owner_avatar_url,omitempty"`
}

type VersionMeta struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Spec Types ---

type ContentSpec struct {
	Content string `json:"content"`
}

type TeamSpec struct {
	AgentType  string     `json:"agent_type"`
	Command    string     `json:"command"`
	GitRepoURL string     `json:"git_repo_url,omitempty"`
	Children   []ChildRef `json:"children"`
}

type ChildRef struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type ResourceIdentifier struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

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

type ResourceRefRequest struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type TeamWriteRequest struct {
	Name           string               `json:"name"`
	Command        string               `json:"command"`
	GitRepoURL     string               `json:"git_repo_url,omitempty"`
	Instruction    *ResourceRefRequest  `json:"instruction,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude-setting,omitempty"`
	CodexSettings  *ResourceRefRequest  `json:"codex-setting,omitempty"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	Children       []ChildRefRequest    `json:"children,omitempty"`
	Summary        string               `json:"summary,omitempty"`
}

type TeamPatchRequest struct {
	Command        *string              `json:"command,omitempty"`
	GitRepoURL     *string              `json:"git_repo_url,omitempty"`
	Instruction    *ResourceRefRequest  `json:"instruction,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude-setting,omitempty"`
	CodexSettings  *ResourceRefRequest  `json:"codex-setting,omitempty"`
	Skills         []ResourceRefRequest `json:"skills,omitempty"`
	Children       []ChildRefRequest    `json:"children,omitempty"`
	Summary        *string              `json:"summary,omitempty"`
}

type ChildRefRequest struct {
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	ChildVersion int    `json:"child_version"`
}

// --- Org Types ---

type CreateOrgRequest struct {
	Name            string          `json:"name"`
	NamespaceAccess NamespaceAccess `json:"namespace_access"`
}

type OrgResponse struct {
	Name            string          `json:"name"`
	NamespaceAccess NamespaceAccess `json:"namespace_access"`
	AvatarURL       string          `json:"avatar_url,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type OrgMemberResponse struct {
	Name string `json:"name"`
	Role int    `json:"role"`
}

type InviteMemberRequest struct {
	Name string `json:"name"`
	Role int    `json:"role"`
}

type NamespaceAccess int

const (
	NamespaceAccessPublic NamespaceAccess = iota
	NamespaceAccessPrivate
)

// --- Resource Kind ---

type ResourceKind string

const (
	KindTeam           ResourceKind = "team"
	KindSkill          ResourceKind = "skill"
	KindInstruction    ResourceKind = "instruction"
	KindClaudeSettings ResourceKind = "claude-setting"
	KindCodexSettings  ResourceKind = "codex-setting"
)

var kindURLPaths = map[ResourceKind]string{
	KindTeam:           "teams",
	KindSkill:          "skills",
	KindInstruction:    "instructions",
	KindClaudeSettings: "claude-settings",
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

func (r *TeamWriteRequest) SetInstructionRef(ref *ResourceRefRequest) {
	r.Instruction = ref
}

func (r *TeamWriteRequest) SetSettingsRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeSettings:
		r.ClaudeSettings = ref
	case KindCodexSettings:
		r.CodexSettings = ref
	}
}

func (r *TeamPatchRequest) SetInstructionRef(ref *ResourceRefRequest) {
	r.Instruction = ref
}

func (r *TeamPatchRequest) SetSettingsRef(kind string, ref *ResourceRefRequest) {
	switch ResourceKind(kind) {
	case KindClaudeSettings:
		r.ClaudeSettings = ref
	case KindCodexSettings:
		r.CodexSettings = ref
	}
}

func IsInstructionKind(kind string) bool {
	return ResourceKind(kind) == KindInstruction
}

func IsSettingsKind(kind string) bool {
	switch ResourceKind(kind) {
	case KindClaudeSettings, KindCodexSettings:
		return true
	}
	return false
}

func IsContentKind(kind string) bool {
	return IsInstructionKind(kind) || IsSettingsKind(kind) || ResourceKind(kind) == KindSkill
}
