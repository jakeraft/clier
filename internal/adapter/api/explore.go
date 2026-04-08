package api

// ListPublicTeams returns all public teams.
// GET /api/v1/teams
func (c *Client) ListPublicTeams() ([]TeamResponse, error) {
	var r []TeamResponse
	return r, c.get("/api/v1/teams", &r)
}

// ListPublicMembers returns all public members.
// GET /api/v1/members
func (c *Client) ListPublicMembers() ([]MemberResponse, error) {
	var r []MemberResponse
	return r, c.get("/api/v1/members", &r)
}

// ListPublicSkills returns all public skills.
// GET /api/v1/skills
func (c *Client) ListPublicSkills() ([]SkillResponse, error) {
	var r []SkillResponse
	return r, c.get("/api/v1/skills", &r)
}

// ListPublicClaudeMds returns all public claude-mds.
// GET /api/v1/claude-mds
func (c *Client) ListPublicClaudeMds() ([]ClaudeMdResponse, error) {
	var r []ClaudeMdResponse
	return r, c.get("/api/v1/claude-mds", &r)
}

// ListPublicClaudeSettings returns all public claude-settings.
// GET /api/v1/claude-settings
func (c *Client) ListPublicClaudeSettings() ([]ClaudeSettingsResponse, error) {
	var r []ClaudeSettingsResponse
	return r, c.get("/api/v1/claude-settings", &r)
}
