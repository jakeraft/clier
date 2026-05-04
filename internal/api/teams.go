package api

import (
	"net/url"
	"strconv"
)

// Team mirrors the server's Team envelope (ADR-0013 §4.1). Read paths
// (Get / Create / Update / List items) all return this shape — the rich
// projection (`subteams`, `layout`, `namespace_profile`, `star`) is
// composed server-side so the CLI does not fan out per-subteam queries.
type Team struct {
	Namespace        string           `json:"namespace"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	AgentType        string           `json:"agent_type"`
	Command          string           `json:"command"`
	GitRepoURL       string           `json:"git_repo_url"`
	GitSubpath       string           `json:"git_subpath"`
	Protocol         string           `json:"protocol"`
	Subteams         []Subteam        `json:"subteams"`
	CreatedAt        string           `json:"created_at"`
	UpdatedAt        string           `json:"updated_at"`
	Layout           Layout           `json:"layout"`
	NamespaceProfile NamespaceProfile `json:"namespace_profile"`
	Star             StarStatus       `json:"star"`
}

// Subteam is the rich projection embedded in a parent Team's read response.
// Inputs (Create / Update) take TeamKey by natural key only.
type Subteam struct {
	Namespace        string           `json:"namespace"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	AgentType        string           `json:"agent_type"`
	NamespaceProfile NamespaceProfile `json:"namespace_profile"`
}

// Layout is the AgentType vendor convention with the team's git_subpath
// already prefixed (server-side, ADR-0001 §5).
type Layout struct {
	InstructionPath string `json:"instruction_path"`
	SkillsDirPath   string `json:"skills_dir_path"`
	SettingsPath    string `json:"settings_path"`
}

// NamespaceProfile is the owner's display fields (avatar etc.) embedded so
// consumers do not synthesise from the login (ADR-0013 §4.1).
type NamespaceProfile struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// StarStatus is the caller-aware star envelope (ADR-0013 §4.1). Anonymous
// callers receive `Starred=false`.
type StarStatus struct {
	Starred bool  `json:"starred"`
	Count   int64 `json:"count"`
}

// TeamKey is the natural-key reference used when inputting subteam links
// (ADR-0001 §3 자연키).
type TeamKey struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// CreateTeamRequest is the body of POST /api/v1/teams (ADR-0013 §1.1).
// Both `namespace` and `name` live in the body — the resource is
// flat-composite so no path parent exists.
type CreateTeamRequest struct {
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	AgentType   string    `json:"agent_type"`
	Command     string    `json:"command"`
	GitRepoURL  string    `json:"git_repo_url"`
	GitSubpath  string    `json:"git_subpath,omitempty"`
	Subteams    []TeamKey `json:"subteams,omitempty"`
}

// ListTeamsResponse is the cursor-paginated list envelope (ADR-0013 §3.2).
type ListTeamsResponse struct {
	Data []Team   `json:"data"`
	Meta PageMeta `json:"meta"`
}

// PageMeta carries the cursor metadata. `NextCursor` is empty when
// `HasNext` is false (ADR-0013 §3.2).
type PageMeta struct {
	HasNext    bool   `json:"has_next"`
	NextCursor string `json:"next_cursor"`
}

// ListTeamsQuery is the optional filter / pagination set for list calls.
// Empty fields are omitted from the query string. `Sort` enum values are
// validated server-side against the 4-enum set (ADR-0013 §5.3).
type ListTeamsQuery struct {
	Namespace string
	AgentType string
	Sort      string
	Q         string
	PageSize  int
	PageToken string
}

// ListTeams calls GET /api/v1/teams. Public endpoint — caller-aware
// `star.starred` is filled when the client carries a session token.
func (c *Client) ListTeams(q ListTeamsQuery) (*ListTeamsResponse, error) {
	v := url.Values{}
	if q.Namespace != "" {
		v.Set("namespace", q.Namespace)
	}
	if q.AgentType != "" {
		v.Set("agent_type", q.AgentType)
	}
	if q.Sort != "" {
		v.Set("sort", q.Sort)
	}
	if q.Q != "" {
		v.Set("q", q.Q)
	}
	if q.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(q.PageSize))
	}
	if q.PageToken != "" {
		v.Set("page_token", q.PageToken)
	}
	path := "/api/v1/teams"
	if encoded := v.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var r ListTeamsResponse
	return &r, c.do("GET", path, nil, &r)
}

// GetTeam calls GET /api/v1/teams/{ns}/{name}. Public endpoint.
func (c *Client) GetTeam(namespace, name string) (*Team, error) {
	var t Team
	return &t, c.do("GET", "/api/v1/teams/"+namespace+"/"+name, nil, &t)
}

// CreateTeam calls POST /api/v1/teams. Requires session + ownership
// (`actor.namespace == body.namespace`, ADR-0005 §3.1).
func (c *Client) CreateTeam(req CreateTeamRequest) (*Team, error) {
	var t Team
	return &t, c.do("POST", "/api/v1/teams", req, &t)
}

// UpdateTeam calls PATCH /api/v1/teams/{ns}/{name} with a JSON Merge
// Patch body (RFC 7396, ADR-0013 §2.1). Caller passes a sparse map of
// only the fields to change; immutable fields (namespace / name /
// agent_type) are excluded from the request schema server-side.
func (c *Client) UpdateTeam(namespace, name string, patch map[string]any) (*Team, error) {
	var t Team
	return &t, c.do("PATCH", "/api/v1/teams/"+namespace+"/"+name, patch, &t)
}

// DeleteTeam calls DELETE /api/v1/teams/{ns}/{name}. Returns 204 on
// success.
func (c *Client) DeleteTeam(namespace, name string) error {
	return c.do("DELETE", "/api/v1/teams/"+namespace+"/"+name, nil, nil)
}

// StarTeam calls PUT /api/v1/teams/{ns}/{name}/star. Idempotent set —
// the response Team envelope reflects the new star state.
func (c *Client) StarTeam(namespace, name string) error {
	return c.do("PUT", "/api/v1/teams/"+namespace+"/"+name+"/star", nil, nil)
}

// UnstarTeam calls DELETE /api/v1/teams/{ns}/{name}/star. Idempotent
// unset.
func (c *Client) UnstarTeam(namespace, name string) error {
	return c.do("DELETE", "/api/v1/teams/"+namespace+"/"+name+"/star", nil, nil)
}
