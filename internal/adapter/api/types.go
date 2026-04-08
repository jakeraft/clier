package api

// ResourceRef is a lightweight reference to a SaaS resource,
// used in Member and Team responses to represent linked resources.
type ResourceRef struct {
	ID        int64   `json:"id"`
	Owner     string  `json:"owner"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Name      string  `json:"name"`
}

// commonFields are shared by all SaaS resources (not embedded, just documented):
// id, owner_id, name, visibility, is_fork, fork_id, fork_count, latest_version,
// created_at, updated_at, owner_login, owner_avatar_url, fork_name, fork_owner_login
