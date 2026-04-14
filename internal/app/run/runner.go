package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	MemberID    int64  `json:"member_id"`
	Name        string `json:"name"`
	AgentType   string `json:"agent_type"`
	Window      int    `json:"window"`
	Memberspace string `json:"memberspace"`
	Cwd         string `json:"cwd"`
	Command     string `json:"command"`
}

type RecordedMessage struct {
	FromMemberID *int64    `json:"from_member_id,omitempty"`
	ToMemberID   *int64    `json:"to_member_id,omitempty"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type RecordedNote struct {
	MemberID  *int64    `json:"member_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

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

// NewPlan builds a persisted RunPlan from concrete terminal launch specs.
func NewPlan(runID, sessionName string, plans []MemberTerminal) *RunPlan {
	memberTerminals := make([]MemberTerminal, len(plans))
	copy(memberTerminals, plans)
	return &RunPlan{
		RunID:     runID,
		Session:   sessionName,
		Members:   memberTerminals,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}
}

// SavePlan writes the RunPlan to {copyRoot}/.clier/{runID}.json.
func SavePlan(copyRoot, runID string, plan *RunPlan) error {
	dir := filepath.Join(copyRoot, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create plan dir: %w", err)
	}

	path := PlanPath(copyRoot, runID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// PlanPath returns the absolute path of a run plan file under a local clone.
func PlanPath(copyRoot, runID string) string {
	return filepath.Join(copyRoot, ".clier", runID+".json")
}

// LoadPlan reads a saved RunPlan from {copyRoot}/.clier/{runID}.json.
func LoadPlan(copyRoot, runID string) (*RunPlan, error) {
	path := PlanPath(copyRoot, runID)
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

// FindMember finds the terminal slot for a member in the run plan.
func (p *RunPlan) FindMember(memberID int64) (*MemberTerminal, bool) {
	for i := range p.Members {
		if p.Members[i].MemberID == memberID {
			return &p.Members[i], true
		}
	}
	return nil, false
}

func (p *RunPlan) AddMessage(fromMemberID, toMemberID *int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("message content must not be empty")
	}
	p.Messages = append(p.Messages, RecordedMessage{
		FromMemberID: copyInt64Ptr(fromMemberID),
		ToMemberID:   copyInt64Ptr(toMemberID),
		Content:      content,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (p *RunPlan) AddNote(memberID *int64, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("note content must not be empty")
	}
	p.Notes = append(p.Notes, RecordedNote{
		MemberID:  copyInt64Ptr(memberID),
		Content:   content,
		CreatedAt: time.Now(),
	})
	return nil
}

func (p *RunPlan) MarkStopped() {
	now := time.Now()
	p.Status = StatusStopped
	p.StoppedAt = &now
}

// ParseMemberID converts a command-line member ID to int64.
func ParseMemberID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid member id %q: %w", raw, err)
	}
	return id, nil
}

func copyInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
