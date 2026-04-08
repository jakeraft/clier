package api

import (
	"fmt"
	"time"
)

// TeamMemberResponse is a team member instance within a team.
type TeamMemberResponse struct {
	ID       string `json:"id"`
	MemberID string `json:"member_id"`
	Name     string `json:"name"`
}

// RelationResponse is a leader-worker relation.
type RelationResponse struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TeamResponse is the server's JSON representation of a Team.
type TeamResponse struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	RootTeamMemberID string               `json:"root_team_member_id"`
	TeamMembers      []TeamMemberResponse `json:"team_members"`
	Relations        []RelationResponse   `json:"relations"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

func (c *Client) CreateTeam(owner string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams", owner), body, &r)
}

func (c *Client) GetTeam(owner, id string) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, id), &r)
}

func (c *Client) ListTeams(owner string) ([]TeamResponse, error) {
	var r []TeamResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/teams", owner), &r)
}

func (c *Client) UpdateTeam(owner, id string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, id), body, &r)
}

func (c *Client) DeleteTeam(owner, id string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, id))
}

func (c *Client) AddTeamMember(owner, teamID string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams/%s/members", owner, teamID), body, &r)
}

func (c *Client) RemoveTeamMember(owner, teamID, teamMemberID string) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.do("DELETE", fmt.Sprintf("/api/v1/orgs/%s/teams/%s/members/%s", owner, teamID, teamMemberID), nil, &r)
}

func (c *Client) AddTeamRelation(owner, teamID string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams/%s/relations", owner, teamID), body, &r)
}

func (c *Client) RemoveTeamRelation(owner, teamID string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.do("DELETE", fmt.Sprintf("/api/v1/orgs/%s/teams/%s/relations", owner, teamID), body, &r)
}

func (c *Client) ImportTeam(owner string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams/import", owner), body, &r)
}
