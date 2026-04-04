package domain

// MemberPlan is a fully-resolved execution plan for a single team member.
// Binary, Model, Envs are NOT stored — they are already resolved into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
//
// Plan retains {{CLIER_*}} placeholders; these are resolved at session start
// into concrete paths. The stored plan is safe for name/ID lookups but should
// not be used to reconstruct the workspace without re-resolving placeholders.
type MemberPlan struct {
	TeamMemberID string        `json:"team_member_id"`
	MemberName   string        `json:"member_name"`
	Terminal     TerminalPlan  `json:"terminal"`
	Workspace    WorkspacePlan `json:"workspace"`
}

// TerminalPlan holds the shell command that launches the member agent.
type TerminalPlan struct {
	Command string `json:"command"`
}

// WorkspacePlan holds the filesystem setup for a member's isolated environment.
type WorkspacePlan struct {
	Memberspace string      `json:"memberspace"`
	Files       []FileEntry `json:"files"`
	GitRepo     *GitRepoRef `json:"git_repo"`
}

type GitRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FileEntry is a resolved config file to write to a member's workspace.
type FileEntry struct {
	Path    string `json:"path"`    // relative to memberspace
	Content string `json:"content"`
}

// PromptSnapshot is a resolved system prompt used by plan build logic.
type PromptSnapshot struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

// EnvSnapshot is a resolved environment variable used by plan build logic.
type EnvSnapshot struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}
