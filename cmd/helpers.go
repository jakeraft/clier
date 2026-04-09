package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	apprun "github.com/jakeraft/clier/internal/app/run"
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

func resolveWorkspaceBase() (string, error) {
	base, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	return filepath.Abs(base)
}

func resolveRunPlan(runID string) (*apprun.RunPlan, error) {
	planPath, err := resolveRunPlanPath(runID)
	if err != nil {
		return nil, err
	}
	plan, err := apprun.LoadPlanFromPath(planPath)
	if err != nil {
		return nil, fmt.Errorf("load run plan: %w", err)
	}
	if plan.RunID != "" && plan.RunID != runID {
		return nil, fmt.Errorf("run plan %s belongs to run %s", planPath, plan.RunID)
	}
	return plan, nil
}

func resolveRunPlanPath(runID string) (string, error) {
	if planPath := strings.TrimSpace(os.Getenv("CLIER_RUN_PLAN")); planPath != "" {
		plan, err := apprun.LoadPlanFromPath(planPath)
		if err != nil {
			return "", fmt.Errorf("load CLIER_RUN_PLAN: %w", err)
		}
		if runID != "" && plan.RunID != "" && plan.RunID != runID {
			return "", fmt.Errorf("CLIER_RUN_PLAN points to run %s, not %s", plan.RunID, runID)
		}
		return planPath, nil
	}

	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", err
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		planPath := apprun.PlanPath(dir, runID)
		if _, err := os.Stat(planPath); err == nil {
			return planPath, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", fmt.Errorf("run plan %s not found in current workspace", runID)
}
