package api

// ResourceRef is a lightweight reference to a SaaS resource,
// used in Member and Team responses to represent linked resources.
type ResourceRef struct {
	ID        int64   `json:"id"`
	Owner     string  `json:"owner"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Name      string  `json:"name"`
	Version   int     `json:"version"`
}

type ResourceRefRequest struct {
	ID      int64 `json:"id"`
	Version int   `json:"version"`
}

type MemberRefRequest struct {
	ID      int64 `json:"id"`
	Version int   `json:"version"`
}

// MemberRef is a lightweight reference to a Member resource,
// used in TeamMemberResponse to include agent-specific fields.
type MemberRef struct {
	ResourceRef
	AgentType string `json:"agent_type"`
	Command   string `json:"command"`
}

type ClaudeMdMutationRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ClaudeSettingsMutationRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type SkillMutationRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type MemberMutationRequest struct {
	Name           string               `json:"name"`
	AgentType      string               `json:"agent_type"`
	Command        string               `json:"command"`
	GitRepoURL     string               `json:"git_repo_url"`
	ClaudeMd       *ResourceRefRequest  `json:"claude_md,omitempty"`
	ClaudeSettings *ResourceRefRequest  `json:"claude_settings,omitempty"`
	Skills         []ResourceRefRequest `json:"skills"`
}

type TeamMemberRequest struct {
	Member MemberRefRequest `json:"member"`
	Name   string           `json:"name"`
}

type TeamRelationRequest struct {
	FromIndex int `json:"from_index"`
	ToIndex   int `json:"to_index"`
}

type TeamMutationRequest struct {
	Name        string                `json:"name"`
	TeamMembers []TeamMemberRequest   `json:"team_members"`
	Relations   []TeamRelationRequest `json:"relations"`
	RootIndex   *int                  `json:"root_index"`
}

// commonFields are shared by all SaaS resources (not embedded, just documented):
// id, owner_id, name, visibility, is_fork, fork_id, fork_count, latest_version,
// created_at, updated_at, owner_login, owner_avatar_url, fork_name, fork_owner_login
