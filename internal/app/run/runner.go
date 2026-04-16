package run

import (
	"fmt"
	"os"
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

// Run creates a RunPlan from the given agent plans, saves it to
// {copyRoot}/.clier/{runID}.json, and launches via tmux.
func (r *Runner) Run(copyRoot, runID, sessionName string, plans []AgentTerminal) (*RunPlan, error) {
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
