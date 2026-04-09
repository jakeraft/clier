package api

import (
	"fmt"
	"time"
)

// ClaudeSettingsResponse is the server's JSON representation of a ClaudeSettings resource.
type ClaudeSettingsResponse struct {
	ID             int64     `json:"id"`
	OwnerID        int64     `json:"owner_id"`
	Name           string    `json:"name"`
	Content        string    `json:"content"`
	Visibility     int       `json:"visibility"`
	IsFork         bool      `json:"is_fork"`
	ForkID         *int64    `json:"fork_id,omitempty"`
	ForkName       string    `json:"fork_name"`
	ForkOwnerLogin string    `json:"fork_owner_login"`
	ForkCount      int       `json:"fork_count"`
	LatestVersion  *int      `json:"latest_version,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	OwnerLogin     string    `json:"owner_login"`
	OwnerAvatarURL *string   `json:"owner_avatar_url,omitempty"`
}

func (c *Client) CreateClaudeSettings(owner string, body any) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-settings", owner), body, &r)
}

func (c *Client) GetClaudeSettings(owner, name string) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, name), &r)
}

func (c *Client) ListClaudeSettings(owner string) ([]ClaudeSettingsResponse, error) {
	var r []ClaudeSettingsResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-settings", owner), &r)
}

func (c *Client) UpdateClaudeSettings(owner, name string, body any) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, name), body, &r)
}

func (c *Client) DeleteClaudeSettings(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, name))
}

func (c *Client) ForkClaudeSettings(owner, name string) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s/fork", owner, name), nil, &r)
}
