package domain

type TeamSnapshot struct {
	TeamName     string           `json:"team_name"`
	RootMemberID string           `json:"root_member_id"`
	Members      []MemberSnapshot `json:"members"`
}

type MemberSnapshot struct {
	MemberID       string               `json:"member_id"`
	MemberName     string               `json:"member_name"`
	Binary         CliBinary            `json:"binary"`
	Model          string               `json:"model"`
	CliProfileName string               `json:"cli_profile_name"`
	SystemArgs     []string             `json:"system_args"`
	CustomArgs     []string             `json:"custom_args"`
	DotConfig      DotConfig            `json:"dot_config"`
	SystemPrompts  []PromptSnapshot     `json:"system_prompts"`
	GitRepo        *GitRepoSnapshot     `json:"git_repo"` // nil means not set
	Relations      MemberRelations      `json:"relations"`
}

type PromptSnapshot struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type GitRepoSnapshot struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
