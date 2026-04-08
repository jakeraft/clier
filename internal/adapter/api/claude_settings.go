package api

import (
	"fmt"
	"time"
)

// ClaudeSettingsResponse is the server's JSON representation of a ClaudeSettings resource.
type ClaudeSettingsResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Client) CreateClaudeSettings(owner string, body any) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/claude-settings", owner), body, &r)
}

func (c *Client) GetClaudeSettings(owner, id string) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, id), &r)
}

func (c *Client) ListClaudeSettings(owner string) ([]ClaudeSettingsResponse, error) {
	var r []ClaudeSettingsResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/claude-settings", owner), &r)
}

func (c *Client) UpdateClaudeSettings(owner, id string, body any) (*ClaudeSettingsResponse, error) {
	var r ClaudeSettingsResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, id), body, &r)
}

func (c *Client) DeleteClaudeSettings(owner, id string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/claude-settings/%s", owner, id))
}
