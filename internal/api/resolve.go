package api

// RunManifest is the resolved view of a team — every mount the CLI must
// clone and every agent the CLI must launch in tmux. The shape mirrors the
// server's RunManifest schema (ADR-0002 §2 + amendment 2026-04-29).
type RunManifest struct {
	Mounts []Mount     `json:"mounts"`
	Agents []AgentSpec `json:"agents"`
}

// Mount is a unique (git_repo_url, git_subpath) pair surfaced by the server.
// The CLI clones one workspace dir per Mount.Name into the run scratch dir.
type Mount struct {
	Name       string `json:"name"`
	GitRepoURL string `json:"git_repo_url"`
	GitSubpath string `json:"git_subpath"`
}

// AgentSpec is one runnable agent.
//
// The CLI is vendor-blind: it sends `Command` followed by per-item
// shell-escaped `Args` as a single tmux send-keys, then presses Enter.
// Args carries the runtime protocol wrapped in the agent-type-specific
// injection flag (e.g. `--append-system-prompt …` for claude,
// `-c developer_instructions=…` for codex). The CLI does not inspect it.
type AgentSpec struct {
	ID        string   `json:"id"`
	Window    int      `json:"window"`
	Mount     string   `json:"mount"`
	Cwd       string   `json:"cwd"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	AgentType string   `json:"agent_type"`
}

// ResolveTeam fetches the RunManifest for the given team.
func (c *Client) ResolveTeam(namespace, name string) (*RunManifest, error) {
	var m RunManifest
	return &m, c.do("GET", "/api/v1/teams/"+namespace+"/"+name+"/resolve", nil, &m)
}
