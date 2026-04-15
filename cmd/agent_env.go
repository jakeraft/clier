package cmd

import (
	"os"
	"strings"
)

const (
	envClierAgent      = "CLIER_AGENT"
	envClierRunID      = "CLIER_RUN_ID"
	envClierMemberName = "CLIER_MEMBER_NAME"
	envClierTeamName   = "CLIER_TEAM_NAME"

	// cmdNameRun and cmdNameNote are command names shared between
	// command registration (run.go) and the agent-mode filter (root.go).
	cmdNameRun  = "run"
	cmdNameNote = "note"
)

func isAgentMode() bool {
	return strings.TrimSpace(os.Getenv(envClierAgent)) == "true"
}

func isTeamAgent() bool {
	return strings.TrimSpace(os.Getenv(envClierTeamName)) != ""
}
