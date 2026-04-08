package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// buildMemberEnv returns the environment variables for a member agent.
func buildMemberEnv(runID, memberName, runPlanPath, memberspace string) map[string]string {
	return map[string]string{
		"CLIER_RUN_PLAN":      runPlanPath,
		"CLIER_RUN_ID":        runID,
		"CLIER_MEMBER_ID":     memberName,
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
