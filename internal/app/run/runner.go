package run

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Launcher starts a run from a persisted RunPlan.
type Launcher interface {
	Launch(plan *RunPlan) error
}

// Runner handles RunPlan creation and execution.
type Runner struct {
	launcher Launcher
}

// NewRunner creates a Runner with the given launcher adapter.
func NewRunner(launcher Launcher) *Runner {
	return &Runner{launcher: launcher}
}

// Run creates a RunPlan from the given member plans, saves it to
// {copyRoot}/.clier/{runID}.json, and launches via tmux.
func (r *Runner) Run(copyRoot, runID, sessionName string, plans []MemberTerminal) (*RunPlan, error) {
	plan := NewPlan(runID, sessionName, plans)

	if err := SavePlan(copyRoot, runID, plan); err != nil {
		return nil, fmt.Errorf("save plan: %w", err)
	}

	if err := r.launcher.Launch(plan); err != nil {
		_ = os.Remove(PlanPath(copyRoot, runID))
		return nil, fmt.Errorf("launch: %w", err)
	}

	return plan, nil
}

// SessionName generates a tmux-safe session name from a name and run ID.
func SessionName(name, runID string) string {
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

// ParseMemberID converts a command-line member ID to int64.
func ParseMemberID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid member id %q: %w", raw, err)
	}
	return id, nil
}
