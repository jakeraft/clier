package api

import (
	"fmt"
	"time"
)

// ClaudeMdResponse is the server's JSON representation of a ClaudeMd resource.
type ClaudeMdResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Client) CreateClaudeMd(owner string, body any) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-mds", owner), body, &r)
}

func (c *Client) GetClaudeMd(owner, id string) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, id), &r)
}

func (c *Client) ListClaudeMds(owner string) ([]ClaudeMdResponse, error) {
	var r []ClaudeMdResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-mds", owner), &r)
}

func (c *Client) UpdateClaudeMd(owner, id string, body any) (*ClaudeMdResponse, error) {
	var r ClaudeMdResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, id), body, &r)
}

func (c *Client) DeleteClaudeMd(owner, id string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/claude-mds/%s", owner, id))
}
