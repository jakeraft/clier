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
// inside it. Cwd is the server-composed run_dir-relative cwd — CLI joins
// it with the run dir to get the absolute cwd, no client-side composition.
type GitPrepare struct {
	RepoURL string `json:"repo_url"`
	Subpath string `json:"subpath"`
	Dest    string `json:"dest"`
	Cwd     string `json:"cwd"`
}

// ProtocolPrepare carries the rendered markdown the CLI writes verbatim
// to Dest. Server has substituted every `{{key}}` placeholder.
type ProtocolPrepare struct {
	Content string `json:"content"`
	Dest    string `json:"dest"`
}

// AgentRun is the tmux launch context. The CLI sends Command + Args
// (per-item shell-escaped) as a single send-keys. TUI carries the
// vendor-specific TUI hints the server resolves so the CLI doesn't
// hardcode per-vendor behaviour (ADR-0002 §8).
type AgentRun struct {
	AgentType string   `json:"agent_type"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	TUI       AgentTUI `json:"tui"`
}

// AgentTUI mirrors the server-resolved TUI hints. Empty string is the
// "skip" sentinel for any of the three: empty ReadyMarker means
// "considered ready immediately", empty ExitCommand means "kill-only
// teardown", empty TrustResponse means "no trust prompt".
type AgentTUI struct {
	ReadyMarker   string `json:"ready_marker"`
	ExitCommand   string `json:"exit_command"`
	TrustResponse string `json:"trust_response"`
}

// MintRun calls POST /api/v1/teams/{ns}/{name}/runs (ADR-0002 §1).
// Public endpoint — no auth required, but the Client carries the bearer
// if the caller is logged in (server ignores it for mint).
func (c *Client) MintRun(namespace, name string) (*RunManifest, error) {
	var m RunManifest
	return &m, c.do("POST", teamPath(namespace, name)+"/runs", nil, &m)
}
