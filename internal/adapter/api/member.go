package api

import (
	"fmt"
	"time"
)

// MemberResponse is the server's JSON representation of a Member resource.
type MemberResponse struct {
	ID               int64         `json:"id"`
	OwnerID          int64         `json:"owner_id"`
	Name             string        `json:"name"`
	AgentType        string        `json:"agent_type"`
	Command          string        `json:"command"`
	GitRepoURL       string        `json:"git_repo_url"`
	ClaudeMdID       *int64        `json:"claude_md_id,omitempty"`
	ClaudeSettingsID *int64        `json:"claude_settings_id,omitempty"`
	Visibility       int           `json:"visibility"`
	IsFork           bool          `json:"is_fork"`
	ForkID           *int64        `json:"fork_id,omitempty"`
	ForkName         string        `json:"fork_name"`
	ForkOwnerLogin   string        `json:"fork_owner_login"`
	ForkCount        int           `json:"fork_count"`
	LatestVersion    *int          `json:"latest_version,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	OwnerLogin       string        `json:"owner_login"`
	OwnerAvatarURL   *string       `json:"owner_avatar_url,omitempty"`
	ClaudeMd         *ResourceRef  `json:"claude_md,omitempty"`
	ClaudeSettings   *ResourceRef  `json:"claude_settings,omitempty"`
	Skills           []ResourceRef `json:"skills"`
}

func (c *Client) CreateMember(owner string, body any) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/members", owner), body, &r)
}

func (c *Client) GetMember(owner, name string) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, name), &r)
}

func (c *Client) ListMembers(owner string) ([]MemberResponse, error) {
	var r []MemberResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/members", owner), &r)
}

func (c *Client) UpdateMember(owner, name string, body any) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, name), body, &r)
}

func (c *Client) DeleteMember(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, name))
}

func (c *Client) ForkMember(owner, name string) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/members/%s/fork", owner, name), nil, &r)
}
