package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// buildMemberEnv returns the environment variables for a member agent.
// runID is the int64 server-assigned run ID; teamMemberID is the int64 member ID.
func buildMemberEnv(runID int64, teamMemberID int64, memberName, runPlanPath, memberspace string) map[string]string {
	return map[string]string{
		"CLIER_RUN_PLAN":      runPlanPath,
		"CLIER_RUN_ID":        strconv.FormatInt(runID, 10),
		"CLIER_MEMBER_ID":     strconv.FormatInt(teamMemberID, 10),
		"CLIER_AGENT":         "true",
		"CLAUDE_CONFIG_DIR":   filepath.Join(memberspace, ".claude"),
		"GIT_AUTHOR_NAME":     memberName,
		"GIT_AUTHOR_EMAIL":    "noreply@clier.com",
		"GIT_COMMITTER_NAME":  memberName,
		"GIT_COMMITTER_EMAIL": "noreply@clier.com",
	}
}

// buildFullCommand assembles a shell command with env exports, cd, and the agent command.
func buildFullCommand(env map[string]string, command, cwd string) string {
	var parts []string
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s='%s'", k, v))
	}
	sort.Strings(parts) // deterministic order
	parts = append(parts, fmt.Sprintf("cd '%s'", cwd))
	parts = append(parts, command)
	return strings.Join(parts, " &&\n")
}
