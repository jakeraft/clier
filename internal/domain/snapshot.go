package domain

type TeamSnapshot struct {
	TeamName     string
	RootMemberID string
	Members      []MemberSnapshot
}

type MemberSnapshot struct {
	MemberID      string
	MemberName    string
	Binary        CliBinary
	Model         string
	CliProfileName string
	SystemArgs    []string
	CustomArgs    []string
	DotConfig     DotConfig
	SystemPrompts []SnapshotPrompt
	Environments  []SnapshotEnvironment
	GitRepo       *SnapshotGitRepo // nil means not set
	Relations     MemberRelations
	ComposedPrompt string
}

type SnapshotPrompt struct {
	Name   string
	Prompt string
}

type SnapshotEnvironment struct {
	Name  string
	Key   string
	Value string
}

type SnapshotGitRepo struct {
	Name string
	URL  string
}
