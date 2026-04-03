package domain

// MemberSessionPlan is a fully-resolved execution plan for a single member.
// Binary, Model, Envs are NOT stored — they are already resolved into Command.
// Relations are NOT stored — they are in Team.Relations and baked into the prompt.
type MemberSessionPlan struct {
	MemberID   string        `json:"member_id"`
	MemberName string        `json:"member_name"`
	Terminal   TerminalPlan  `json:"terminal"`
	Workspace  WorkspacePlan `json:"workspace"`
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
