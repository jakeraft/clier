package run

import (
	"errors"
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

// NewPlan builds a RunPlan from concrete terminal launch specs.
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

func copyInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
