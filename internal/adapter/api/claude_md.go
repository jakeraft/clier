package api

import (
	"fmt"
	"time"
)

// ClaudeMdResponse is the server's JSON representation of a ClaudeMd resource.
type ClaudeMdResponse struct {
	ID             int64     `json:"id"`
	OwnerID        int64     `json:"owner_id"`
	Name           string    `json:"name"`
	Content        string    `json:"content"`
	Visibility     int       `json:"visibility"`
	IsFork         bool      `json:"is_fork"`
	ForkID         *int64    `json:"fork_id,omitempty"`
	ForkCount      int       `json:"fork_count"`
	LatestVersion  *int      `json:"latest_version,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	OwnerLogin     string    `json:"owner_login"`
	OwnerAvatarURL *string   `json:"owner_avatar_url,omitempty"`
}

func (c *Client) CreateClaudeMd(owner string, body any) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-mds", owner), body, &r)
}

func (c *Client) GetClaudeMd(owner, name string) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, name), &r)
}

func (c *Client) ListClaudeMds(owner string) ([]ClaudeMdResponse, error) {
	var r []ClaudeMdResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-mds", owner), &r)
}

func (c *Client) UpdateClaudeMd(owner, name string, body any) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, name), body, &r)
}

func (c *Client) DeleteClaudeMd(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, name))
}

func (c *Client) ForkClaudeMd(owner, name string) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s/fork", owner, name), nil, &r)
}
