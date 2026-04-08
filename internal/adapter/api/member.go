package api

import (
	"fmt"
	"time"
)

// MemberResponse is the server's JSON representation of a Member.
type MemberResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Command          string    `json:"command"`
	ClaudeMdID       string    `json:"claude_md_id"`
	SkillIDs         []string  `json:"skill_ids"`
	ClaudeSettingsID string    `json:"claude_settings_id"`
	GitRepoURL       string    `json:"git_repo_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (c *Client) CreateMember(owner string, body any) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/members", owner), body, &r)
}

func (c *Client) GetMember(owner, id string) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, id), &r)
}

func (c *Client) ListMembers(owner string) ([]MemberResponse, error) {
	var r []MemberResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/members", owner), &r)
}

func (c *Client) UpdateMember(owner, id string, body any) (*MemberResponse, error) {
	var r MemberResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, id), body, &r)
}

func (c *Client) DeleteMember(owner, id string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/members/%s", owner, id))
}
