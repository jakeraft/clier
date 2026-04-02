package domain

type TeamSnapshot struct {
	TeamName     string           `json:"team_name"`
	RootMemberID string           `json:"root_member_id"`
	Members      []MemberSnapshot `json:"members"`
}

type MemberSnapshot struct {
	MemberID       string           `json:"member_id"`
	MemberName     string           `json:"member_name"`
	Binary         CliBinary        `json:"binary"`
	Model          string           `json:"model"`
	CliProfileName string           `json:"cli_profile_name"`
	SystemArgs     []string         `json:"system_args"`
	CustomArgs     []string         `json:"custom_args"`
	DotConfig      DotConfig        `json:"dot_config"`
	SystemPrompts  []PromptSnapshot `json:"system_prompts"`
	GitRepo        *GitRepoSnapshot `json:"git_repo"` // nil means not set
	Envs           []EnvSnapshot    `json:"envs"`
	Relations      MemberRelations  `json:"relations"`
}

type PromptSnapshot struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type GitRepoSnapshot struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EnvSnapshot struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SprintSnapshot is the resolved execution plan stored in a Sprint.
// Built from TeamSnapshot by the sprint service.
type SprintSnapshot struct {
	TeamName     string                 `json:"team_name"`
	RootMemberID string                 `json:"root_member_id"`
	Members      []SprintMemberSnapshot `json:"members"`
}

// SprintMemberSnapshot is a fully resolved member execution plan.
type SprintMemberSnapshot struct {
	// Identity + relations (whoami, message validation)
	MemberID   string          `json:"member_id"`
	MemberName string          `json:"member_name"`
	Relations  MemberRelations `json:"relations"`

	// Workspace preparation (filesystem materialization)
	Home      string           `json:"home"`
	WorkDir   string           `json:"work_dir"`
	Binary    CliBinary        `json:"binary"`
	DotConfig DotConfig        `json:"dot_config"`
	GitRepo   *GitRepoSnapshot `json:"git_repo"`

	// Execution (fully resolved shell command)
	Command string `json:"command"`
}
