package api

import (
	"fmt"
	"time"
)

// SkillResponse is the server's JSON representation of a Skill resource.
type SkillResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Client) CreateSkill(owner string, body any) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/skills", owner), body, &r)
}

func (c *Client) GetSkill(owner, id string) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, id), &r)
}

func (c *Client) ListSkills(owner string) ([]SkillResponse, error) {
	var r []SkillResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/skills", owner), &r)
}

func (c *Client) UpdateSkill(owner, id string, body any) (*SkillResponse, error) {
	var r SkillResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, id), body, &r)
}

func (c *Client) DeleteSkill(owner, id string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/skills/%s", owner, id))
}
