package run

import (
	"errors"
	"fmt"
)

// Launcher starts a run from a persisted RunPlan.
type Launcher interface {
	Launch(plan *RunPlan) error
}

type RunnerStore interface {
	Save(plan *RunPlan) error
	Delete(runID string) error
}

// Runner handles RunPlan creation and execution.
type Runner struct {
	launcher Launcher
	store    RunnerStore
}

// NewRunner creates a Runner with the given launcher adapter.
func NewRunner(launcher Launcher, store RunnerStore) *Runner {
	return &Runner{launcher: launcher, store: store}
}

// Run creates a RunPlan, persists it, and launches via tmux.
func (r *Runner) Run(workingCopyPath, runID, sessionName string, plans []AgentTerminal) (*RunPlan, error) {
	plan := NewPlan(runID, sessionName, workingCopyPath, plans)

	if err := r.store.Save(plan); err != nil {
		return nil, fmt.Errorf("save plan: %w", err)
	}

	if err := r.launcher.Launch(plan); err != nil {
		if removeErr := r.store.Delete(runID); removeErr != nil {
			return nil, fmt.Errorf("launch: %w", errors.Join(err, fmt.Errorf("remove plan: %w", removeErr)))
		}
		return nil, fmt.Errorf("launch: %w", err)
	}

	return plan, nil
}
