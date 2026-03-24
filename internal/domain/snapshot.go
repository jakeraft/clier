package domain

type TeamSnapshot struct {
	TeamName     string
	RootMemberID string
	Members      []MemberSnapshot
}

type MemberSnapshot struct {
	MemberID       string
	MemberName     string
	Binary         CliBinary
	Model          string
	CliProfileName string
	SystemArgs     []string
	CustomArgs     []string
	DotConfig      DotConfig
	SystemPrompts  []PromptSnapshot
	Environments   []EnvironmentSnapshot
	GitRepo        *GitRepoSnapshot // nil means not set
	Relations      MemberRelations
}

type PromptSnapshot struct {
	Name   string
	Prompt string
}

type EnvironmentSnapshot struct {
	Name  string
	Key   string
	Value string
}

type GitRepoSnapshot struct {
	Name string
	URL  string
}
