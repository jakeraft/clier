package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"
)

// RunPlan is the persisted local run record saved to .clier/{RUN_ID}.json.
// It captures both the tmux execution plan and mutable runtime state.
type RunPlan struct {
	RunID     string            `json:"run_id"`
	Session   string            `json:"session"`
	Members   []MemberTerminal  `json:"members"`
	Status    string            `json:"status"`
	StartedAt time.Time         `json:"started_at"`
	StoppedAt *time.Time        `json:"stopped_at,omitempty"`
	Messages  []RecordedMessage `json:"messages,omitempty"`
	Notes     []RecordedNote    `json:"notes,omitempty"`
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

type RecordedMessage struct {
	FromTeamMemberID *int64    `json:"from_team_member_id,omitempty"`
	ToTeamMemberID   *int64    `json:"to_team_member_id,omitempty"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

type RecordedNote struct {
	TeamMemberID *int64    `json:"team_member_id,omitempty"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

// Launcher starts a run from a persisted RunPlan.
type Launcher interface {
	Launch(plan *RunPlan, members []domain.MemberPlan) error
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

	if err := SavePlan(workspaceBase, runID, plan); err != nil {
		return nil, fmt.Errorf("save plan: %w", err)
	}

	if err := r.launcher.Launch(plan, plans); err != nil {
		_ = os.Remove(PlanPath(workspaceBase, runID))
		return nil, fmt.Errorf("launch: %w", err)
	}

	return plan, nil
}

// NewPlan builds a persisted RunPlan from concrete member execution plans.
func NewPlan(runID, sessionName string, plans []domain.MemberPlan) *RunPlan {
	memberTerminals := make([]MemberTerminal, len(plans))
	for i, p := range plans {
		cwd := p.Workspace.Memberspace
		if p.Workspace.RepoDir != "" {
			cwd = filepath.Join(p.Workspace.Memberspace, p.Workspace.RepoDir)
		}
		memberTerminals[i] = MemberTerminal{
			TeamMemberID: p.TeamMemberID,
			Name:         p.MemberName,
			Window:       i,
			Memberspace:  p.Workspace.Memberspace,
			Cwd:          cwd,
			Command:      p.Terminal.Command,
		}
	}

	return &RunPlan{
		RunID:     runID,
		Session:   sessionName,
		Members:   memberTerminals,
		Status:    StatusRunning,
		StartedAt: time.Now(),
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

func (p *RunPlan) AddMessage(fromTeamMemberID, toTeamMemberID *int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("message content must not be empty")
	}
	p.Messages = append(p.Messages, RecordedMessage{
		FromTeamMemberID: cloneInt64Ptr(fromTeamMemberID),
		ToTeamMemberID:   cloneInt64Ptr(toTeamMemberID),
		Content:          content,
		CreatedAt:        time.Now(),
	})
	return nil
}

func (p *RunPlan) AddNote(teamMemberID *int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("note content must not be empty")
	}
	p.Notes = append(p.Notes, RecordedNote{
		TeamMemberID: cloneInt64Ptr(teamMemberID),
		Content:      content,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (p *RunPlan) MarkStopped() {
	now := time.Now()
	p.Status = StatusStopped
	p.StoppedAt = &now
}

// ParseTeamMemberID converts a command-line member ID to int64.
func ParseTeamMemberID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid team member id %q: %w", raw, err)
	}
	return id, nil
}

func cloneInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
