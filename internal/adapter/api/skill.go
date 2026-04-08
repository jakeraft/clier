package api

import (
	"fmt"
	"time"
)

// SkillResponse is the server's JSON representation of a Skill resource.
type SkillResponse struct {
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

func (c *Client) CreateSkill(owner string, body any) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/skills", owner), body, &r)
}

func (c *Client) GetSkill(owner, name string) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, name), &r)
}

func (c *Client) ListSkills(owner string) ([]SkillResponse, error) {
	var r []SkillResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/skills", owner), &r)
}

func (c *Client) UpdateSkill(owner, name string, body any) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, name), body, &r)
}

func (c *Client) DeleteSkill(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, name))
}
