package domain

// TeamRelations describes an agent's connections within a team.
// Used by the protocol generator to build agent-facing team protocol files.
type TeamRelations struct {
	Leaders []string `json:"leaders"`
	Workers []string `json:"workers"`
}
