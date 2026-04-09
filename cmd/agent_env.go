package cmd

import (
	"os"
	"strings"
)

const (
	envClierAgent    = "CLIER_AGENT"
	envClierRunID    = "CLIER_RUN_ID"
	envClierMemberID = "CLIER_MEMBER_ID"
	envClierTeamID   = "CLIER_TEAM_ID"
)

func isAgentMode() bool {
	return strings.TrimSpace(os.Getenv(envClierAgent)) == "true"
}

func isTeamAgent() bool {
	return strings.TrimSpace(os.Getenv(envClierTeamID)) != ""
}
