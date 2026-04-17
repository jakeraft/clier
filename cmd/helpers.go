package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
)

// sessionName generates a tmux-safe session name from a name and run ID.
// The runID suffix uses the last 8 characters (the random hex part of
// "<timestamp>-<hex>") so two runs started the same day do not collide
// on the timestamp prefix.
func sessionName(name, runID string) string {
	n := strings.NewReplacer(".", "-", ":", "-", " ", "-", "/", "-").Replace(name)
	if runes := []rune(n); len(runes) > 20 {
		n = string(runes[:20])
	}
	short := runID
	if len(short) > 8 {
		short = short[len(short)-8:]
	}
	return n + "-" + short
}

// buildAgentEnv returns the environment variables for an agent.
func buildAgentEnv(runID, agentID, teamID string) map[string]string {
	env := map[string]string{
		envClierRunID:         runID,
		envClierAgentName:     agentID,
		envClierAgent:         "true",
		"GIT_AUTHOR_NAME":     agentID,
		"GIT_AUTHOR_EMAIL":    "noreply@clier.com",
		"GIT_COMMITTER_NAME":  agentID,
		"GIT_COMMITTER_EMAIL": "noreply@clier.com",
	}
	if teamID != "" {
		env[envClierTeamName] = teamID
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

// rejectIfRunActive returns an error if a running plan already targets
// the given working copy. Same-directory concurrent runs would let two
// vendor processes mutate the same agent files at once, so we block
// the second start and tell the agent to stop the first run (or fork
// the team into a separate working copy if real parallelism is needed).
func rejectIfRunActive(base string) error {
	plans, err := apprun.ListPlans(runsDir())
	if err != nil {
		return err
	}
	for _, p := range plans {
		if p.WorkingCopyPath == base && p.Status == apprun.StatusRunning {
			return fmt.Errorf("run %s is already running for this working copy; clier run stop %s first (or fork the team to run in parallel)", p.RunID, p.RunID)
		}
	}
	return nil
}

// resolveRunPlan loads a run plan by run-id from the central runs directory.
func resolveRunPlan(runID string) (*apprun.RunPlan, error) {
	plan, err := apprun.LoadPlan(runsDir(), runID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("run %s not found", runID)
		}
		return nil, fmt.Errorf("load run plan: %w", err)
	}
	if plan.RunID != "" && plan.RunID != runID {
		return nil, fmt.Errorf("run plan for %s reports run id %s", runID, plan.RunID)
	}
	return plan, nil
}

// globalPlanStore implements apprun.PlanStore by writing to the central runs dir.
type globalPlanStore struct {
	dir string
}

func newPlanStore() *globalPlanStore {
	return &globalPlanStore{dir: runsDir()}
}

func (s *globalPlanStore) Save(plan *apprun.RunPlan) error {
	return apprun.SavePlan(s.dir, plan.RunID, plan)
}

func newRunID() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate run id: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:]), nil
}
