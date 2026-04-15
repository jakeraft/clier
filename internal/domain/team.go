package domain

// MemberRelations describes a team member's connections within a team.
// Used by the protocol generator to build agent-facing team protocol files.
type MemberRelations struct {
	Leaders []int64 `json:"leaders"`
	Workers []int64 `json:"workers"`
}
