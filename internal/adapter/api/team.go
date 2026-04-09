package api

import (
	"fmt"
	"time"
)

// TeamMemberResponse is a team member instance within a team.
type TeamMemberResponse struct {
	ID     int64     `json:"id"`
	TeamID int64     `json:"team_id"`
	Name   string    `json:"name"`
	Member MemberRef `json:"member"`
}

// TeamRelationResponse is a leader-worker relation within a team.
type TeamRelationResponse struct {
	TeamID           int64 `json:"team_id"`
	FromTeamMemberID int64 `json:"from_team_member_id"`
	ToTeamMemberID   int64 `json:"to_team_member_id"`
}

// TeamResponse is the server's JSON representation of a Team resource.
type TeamResponse struct {
	ID               int64                  `json:"id"`
	OwnerID          int64                  `json:"owner_id"`
	Name             string                 `json:"name"`
	AgentTypes       []string               `json:"agent_types"`
	RootTeamMemberID *int64                 `json:"root_team_member_id,omitempty"`
	TeamMembers      []TeamMemberResponse   `json:"team_members"`
	Relations        []TeamRelationResponse `json:"relations"`
	Visibility       int                    `json:"visibility"`
	IsFork           bool                   `json:"is_fork"`
	ForkID           *int64                 `json:"fork_id,omitempty"`
	ForkName         string                 `json:"fork_name"`
	ForkOwnerLogin   string                 `json:"fork_owner_login"`
	ForkCount        int                    `json:"fork_count"`
	LatestVersion    *int                   `json:"latest_version,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	OwnerLogin       string                 `json:"owner_login"`
	OwnerAvatarURL   *string                `json:"owner_avatar_url,omitempty"`
}

func (c *Client) CreateTeam(owner string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams", owner), body, &r)
}

func (c *Client) GetTeam(owner, name string) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, name), &r)
}

func (c *Client) ListTeams(owner string) ([]TeamResponse, error) {
	var r []TeamResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/teams", owner), &r)
}

func (c *Client) UpdateTeam(owner, name string, body any) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, name), body, &r)
}

func (c *Client) DeleteTeam(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/teams/%s", owner, name))
}

func (c *Client) ForkTeam(owner, name string) (*TeamResponse, error) {
	var r TeamResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/teams/%s/fork", owner, name), nil, &r)
}
