package terminal

type MemberLaunch struct {
	MemberID   string
	MemberName string
	Command    string
	Env        []string
}

type LaunchResult struct {
	WorkspaceRef string
	Surfaces     map[string]string // memberID → surface ref
}

type SprintTerminal interface {
	Launch(sprintID, sprintName string, members []MemberLaunch) (*LaunchResult, error)
	Terminate(sprintID string) error
}
