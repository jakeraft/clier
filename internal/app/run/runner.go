package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
)

// RunPlan is the execution plan saved to .clier/{RUN_ID}.json.
// It captures the tmux session layout so that subsequent commands
// (attach, stop) can find the running processes.
type RunPlan struct {
	Session string           `json:"session"`
	Members []MemberTerminal `json:"members"`
}

// MemberTerminal maps a member to its tmux window and launch command.
type MemberTerminal struct {
	Name    string `json:"name"`
	Window  int    `json:"window"`
	Cwd     string `json:"cwd"`
	Command string `json:"command"`
}

// Runner handles RunPlan creation and tmux execution.
// It reuses the existing Terminal interface for tmux operations.
type Runner struct {
	terminal Terminal
}

// NewRunner creates a Runner with the given terminal adapter.
func NewRunner(term Terminal) *Runner {
	return &Runner{terminal: term}
}

// Run creates a RunPlan from the given member plans, saves it to
// {workspaceBase}/.clier/{runID}.json, and launches via tmux.
func (r *Runner) Run(workspaceBase, runID, sessionName string, plans []domain.MemberPlan) error {
	memberTerminals := make([]MemberTerminal, len(plans))
	for i, p := range plans {
		memberTerminals[i] = MemberTerminal{
			Name:    p.MemberName,
			Window:  i,
			Cwd:     p.Workspace.Memberspace,
			Command: p.Terminal.Command,
		}
	}

	plan := &RunPlan{
		Session: sessionName,
		Members: memberTerminals,
	}

	if err := savePlan(workspaceBase, runID, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}

	if err := r.terminal.Launch(runID, sessionName, plans); err != nil {
		return fmt.Errorf("launch: %w", err)
	}

	return nil
}

// savePlan writes the RunPlan to {workspaceBase}/.clier/{runID}.json.
func savePlan(workspaceBase, runID string, plan *RunPlan) error {
	dir := filepath.Join(workspaceBase, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create plan dir: %w", err)
	}

	path := filepath.Join(dir, runID+".json")
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadPlan reads a saved RunPlan from {workspaceBase}/.clier/{runID}.json.
func LoadPlan(workspaceBase, runID string) (*RunPlan, error) {
	path := filepath.Join(workspaceBase, ".clier", runID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	var plan RunPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}
	return &plan, nil
}
