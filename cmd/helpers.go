package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

// sessionName generates a tmux-safe session name from a name and run ID.
func sessionName(name, runID string) string {
	n := strings.NewReplacer(".", "-", ":", "-", " ", "-", "/", "-").Replace(name)
	if runes := []rune(n); len(runes) > 20 {
		n = string(runes[:20])
	}
	short := runID
	if len(short) > 8 {
		short = short[:8]
	}
	return n + "-" + short
}

// parseMemberID converts a command-line member ID to int64.
func parseMemberID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid member id %q: %w", raw, err)
	}
	return id, nil
}

// buildMemberEnv returns the environment variables for a member agent.
// runID is a locally generated run ID; teamMemberID is the int64 member ID.
// teamID is set only for agents launched from a team local clone.
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
		parts = append(parts, fmt.Sprintf("export %s=%s", k, shellQuote(v)))
	}
	sort.Strings(parts) // deterministic order
	parts = append(parts, "cd "+shellQuote(cwd))
	parts = append(parts, command)
	return strings.Join(parts, " &&\n")
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'"'"'`) + "'"
}

func resolveCurrentDir() (string, error) {
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

// localPlanStore implements apprun.PlanStore by writing to the local clone's .clier/ directory.
type localPlanStore struct {
	copyRoot string
}

func newPlanStore() (*localPlanStore, error) {
	runtimeDir, err := resolveRuntimeDir()
	if err != nil {
		return nil, err
	}
	if runtimeDir == "" {
		return nil, errors.New("runtime dir not found in current local clone")
	}
	return &localPlanStore{copyRoot: filepath.Dir(runtimeDir)}, nil
}

func (s *localPlanStore) Save(plan *apprun.RunPlan) error {
	return apprun.SavePlan(s.copyRoot, plan.RunID, plan)
}

func resolveRunPlanPath(runID string) (string, error) {
	runtimeDir, err := resolveRuntimeDir()
	if err != nil {
		return "", err
	}
	if runtimeDir == "" {
		return "", errors.New("runtime dir not found in current local clone")
	}
	planPath := filepath.Join(runtimeDir, runID+".json")
	if _, err := os.Stat(planPath); err == nil {
		return planPath, nil
	}
	return "", fmt.Errorf("run plan %s not found in current local clone", runID)
}

func resolveRuntimeDir() (string, error) {
	base, err := resolveCurrentDir()
	if err != nil {
		return "", err
	}

	copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return filepath.Join(copyRoot, ".clier"), nil
}

func newRunID() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate run id: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:]), nil
}
