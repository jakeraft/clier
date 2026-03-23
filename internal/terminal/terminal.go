package terminal

type MemberLaunch struct {
	MemberID   string
	MemberName string
	Command    string
	Env        []string
}

type SprintTerminal interface {
	Launch(sprintID, sprintName string, members []MemberLaunch) error
	DeliverText(sprintID, memberID, text string) error
	Terminate(sprintID string) error
}
