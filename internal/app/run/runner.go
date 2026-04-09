package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// RunPlan is the execution plan saved to .clier/{RUN_ID}.json.
// It captures the tmux session layout so that subsequent commands
// (attach, stop) can find the running processes.
type RunPlan struct {
	RunID   string           `json:"run_id"`
	Session string           `json:"session"`
	Members []MemberTerminal `json:"members"`
}

// MemberTerminal maps a member to its tmux window and launch command.
type MemberTerminal struct {
	TeamMemberID int64  `json:"team_member_id"`
	Name         string `json:"name"`
	Window       int    `json:"window"`
	Memberspace  string `json:"memberspace"`
	Cwd          string `json:"cwd"`
	Command      string `json:"command"`
}

// Launcher starts a run using a persisted RunPlan.
type Launcher interface {
	Launch(runID, planPath string, plan *RunPlan, members []domain.MemberPlan) error
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
// {workspaceBase}/.clier/{runID}.json, and launches via tmux.
func (r *Runner) Run(workspaceBase, runID, sessionName string, plans []domain.MemberPlan) (*RunPlan, error) {
	plan := NewPlan(runID, sessionName, plans)
	planPath := PlanPath(workspaceBase, runID)

	if err := SavePlan(workspaceBase, runID, plan); err != nil {
		return nil, fmt.Errorf("save plan: %w", err)
	}

	if err := r.launcher.Launch(runID, planPath, plan, plans); err != nil {
		return nil, fmt.Errorf("launch: %w", err)
	}

	return plan, nil
}

// NewPlan builds a persisted RunPlan from concrete member execution plans.
func NewPlan(runID, sessionName string, plans []domain.MemberPlan) *RunPlan {
	memberTerminals := make([]MemberTerminal, len(plans))
	for i, p := range plans {
		memberTerminals[i] = MemberTerminal{
			TeamMemberID: p.TeamMemberID,
			Name:         p.MemberName,
			Window:       i,
			Memberspace:  p.Workspace.Memberspace,
			Cwd:          filepath.Join(p.Workspace.Memberspace, "project"),
			Command:      p.Terminal.Command,
		}
	}

	return &RunPlan{
		RunID:   runID,
		Session: sessionName,
		Members: memberTerminals,
	}
}

// SavePlan writes the RunPlan to {workspaceBase}/.clier/{runID}.json.
func SavePlan(workspaceBase, runID string, plan *RunPlan) error {
	dir := filepath.Join(workspaceBase, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create plan dir: %w", err)
	}

	path := PlanPath(workspaceBase, runID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// PlanPath returns the absolute path of a run plan file under a workspace.
func PlanPath(workspaceBase, runID string) string {
	return filepath.Join(workspaceBase, ".clier", runID+".json")
}

// LoadPlan reads a saved RunPlan from {workspaceBase}/.clier/{runID}.json.
func LoadPlan(workspaceBase, runID string) (*RunPlan, error) {
	path := PlanPath(workspaceBase, runID)
	return LoadPlanFromPath(path)
}

// LoadPlanFromPath reads a saved RunPlan from an absolute file path.
func LoadPlanFromPath(path string) (*RunPlan, error) {
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

// SessionName generates a tmux-safe session name from a name and run ID.
func SessionName(name, runID string) string {
	n := strings.NewReplacer(".", "-", ":", "-", " ", "-", "/", "-").Replace(name)
	if len(n) > 20 {
		n = n[:20]
	}
	short := runID
	if len(short) > 8 {
		short = short[:8]
	}
	return n + "-" + short
}

// FindMember finds the terminal slot for a team member in the run plan.
func (p *RunPlan) FindMember(teamMemberID int64) (*MemberTerminal, bool) {
	for i := range p.Members {
		if p.Members[i].TeamMemberID == teamMemberID {
			return &p.Members[i], true
		}
	}
	return nil, false
}

// ParseTeamMemberID converts a command-line member ID to int64.
func ParseTeamMemberID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid team member id %q: %w", raw, err)
	}
	return id, nil
}
