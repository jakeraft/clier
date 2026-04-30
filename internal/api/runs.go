package api

// RunManifest is the response from POST /runs (ADR-0002 §2). The server
// mints `RunID` per call (stateless), so every CLI invocation gets its
// own value back. `Agents` is BFS-ordered; the array index doubles as
// the tmux window number — no explicit field.
type RunManifest struct {
	RunID  string      `json:"run_id"`
	Agents []AgentSpec `json:"agents"`
}

// AgentSpec is one runnable agent — agent-grouped: identity (`ID`),
// what to put on disk (`Prepare`), how to launch (`Run`).
type AgentSpec struct {
	ID      string       `json:"id"`
	Prepare AgentPrepare `json:"prepare"`
	Run     AgentRun     `json:"run"`
}

// AgentPrepare bundles the file-system artefacts the CLI must place
// before launch (ADR-0002 §6 layout). Git is always present. Protocol
// is optional — vendors that inline the protocol into run.args (e.g.
// codex) omit it, and the CLI skips the redundant file write.
type AgentPrepare struct {
	Git      GitPrepare       `json:"git"`
	Protocol *ProtocolPrepare `json:"protocol,omitempty"`
}

// GitPrepare names the external repo the CLI clones plus the cwd offset
// inside it.
type GitPrepare struct {
	RepoURL string `json:"repo_url"`
	Subpath string `json:"subpath"`
	Dest    string `json:"dest"`
}

// ProtocolPrepare carries the rendered markdown the CLI writes verbatim
// to Dest. Server has substituted every `{{key}}` placeholder.
type ProtocolPrepare struct {
	Content string `json:"content"`
	Dest    string `json:"dest"`
}

// AgentRun is the tmux launch context. The CLI sends Command + Args
// (per-item shell-escaped) as a single send-keys.
type AgentRun struct {
	AgentType string   `json:"agent_type"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
}

// MintRun calls POST /api/v1/teams/{ns}/{name}/runs (ADR-0002 §1).
// Public endpoint — no auth required, but the Client carries the bearer
// if the caller is logged in (server ignores it for mint).
func (c *Client) MintRun(namespace, name string) (*RunManifest, error) {
	var m RunManifest
	return &m, c.do("POST", "/api/v1/teams/"+namespace+"/"+name+"/runs", nil, &m)
}
