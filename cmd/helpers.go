package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	appclone "github.com/jakeraft/clier/internal/app/clone"
	apprun "github.com/jakeraft/clier/internal/app/run"
)

// buildMemberEnv returns the environment variables for a member agent.
// runID is a locally generated run ID; teamMemberID is the int64 member ID.
// teamID is set only for agents launched as part of a team run.
func buildMemberEnv(runID string, teamMemberID int64, teamID *int64, memberName string) map[string]string {
	env := map[string]string{
		envClierRunID:         runID,
		envClierMemberID:      strconv.FormatInt(teamMemberID, 10),
		envClierAgent:         "true",
		"GIT_AUTHOR_NAME":     memberName,
		"GIT_AUTHOR_EMAIL":    "noreply@clier.com",
		"GIT_COMMITTER_NAME":  memberName,
		"GIT_COMMITTER_EMAIL": "noreply@clier.com",
	}
	if teamID != nil {
		env[envClierTeamID] = strconv.FormatInt(*teamID, 10)
	}
	return env
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

func saveRunPlan(runID string, plan *apprun.RunPlan) error {
	runtimeDir, err := resolveRuntimeDir()
	if err != nil {
		return err
	}
	if runtimeDir == "" {
		return fmt.Errorf("runtime dir not found in current clone")
	}
	workspaceBase := filepath.Dir(runtimeDir)
	return apprun.SavePlan(workspaceBase, runID, plan)
}

func resolveRunPlanPath(runID string) (string, error) {
	runtimeDir, err := resolveRuntimeDir()
	if err != nil {
		return "", err
	}
	if runtimeDir == "" {
		return "", fmt.Errorf("runtime dir not found in current clone")
	}
	planPath := filepath.Join(runtimeDir, runID+".json")
	if _, err := os.Stat(planPath); err == nil {
		return planPath, nil
	}
	return "", fmt.Errorf("run plan %s not found in current clone", runID)
}

func resolveRuntimeDir() (string, error) {
	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", err
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		runtimeDir := filepath.Join(dir, ".clier")
		cloneMeta := filepath.Join(runtimeDir, appclone.CloneMetadataFile)
		if stat, err := os.Stat(runtimeDir); err == nil && stat.IsDir() {
			if _, err := os.Stat(cloneMeta); err == nil {
				return runtimeDir, nil
			}
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat runtime dir: %w", err)
		}
		if _, err := os.Stat(cloneMeta); err == nil {
			return runtimeDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return "", nil
}

func newRunID() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate run id: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:]), nil
}
