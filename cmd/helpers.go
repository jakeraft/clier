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
	"github.com/jakeraft/clier/internal/domain"
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
	repo, err := newRunRepository()
	if err != nil {
		return err
	}
	plan, found, err := repo.FindRunningForWorkingCopy(base)
	if err != nil {
		return err
	}
	if found {
		return &domain.Fault{
			Kind:    domain.KindRunAlreadyRunning,
			Subject: map[string]string{"run_id": plan.RunID},
		}
	}
	return nil
}

// resolveRunPlan loads a run plan by run-id from the central runs directory.
func resolveRunPlan(runID string) (*apprun.RunPlan, error) {
	repo, err := newRunRepository()
	if err != nil {
		return nil, err
	}
	plan, err := repo.Load(runID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &domain.Fault{
				Kind:    domain.KindRunNotFound,
				Subject: map[string]string{"run_id": runID},
			}
		}
		return nil, fmt.Errorf("load run plan: %w", err)
	}
	if plan.RunID != "" && plan.RunID != runID {
		return nil, &domain.Fault{
			Kind: domain.KindInternal,
			Subject: map[string]string{
				"detail": "run plan for " + runID + " reports run id " + plan.RunID,
			},
		}
	}
	return plan, nil
}

func newRunID() (string, error) {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", fmt.Errorf("generate run id: %w", err)
	}
	return time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:]), nil
}
