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
	Name        string `json:"name"`
	AgentType   string `json:"agent_type"`
	Window      int    `json:"window"`
	Memberspace string `json:"memberspace"`
	Cwd         string `json:"cwd"`
	Command     string `json:"command"`
}

type RecordedMessage struct {
	FromMember *string   `json:"from_member,omitempty"`
	ToMember   *string   `json:"to_member,omitempty"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type RecordedNote struct {
	Member    *string   `json:"member,omitempty"`
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

// FindMember finds the terminal slot for a member by name in the run plan.
func (p *RunPlan) FindMember(memberName string) (*MemberTerminal, bool) {
	for i := range p.Members {
		if p.Members[i].Name == memberName {
			return &p.Members[i], true
		}
	}
	return nil, false
}

func (p *RunPlan) AddMessage(fromMember, toMember *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("message content must not be empty")
	}
	p.Messages = append(p.Messages, RecordedMessage{
		FromMember: copyStrPtr(fromMember),
		ToMember:   copyStrPtr(toMember),
		Content:    content,
		CreatedAt:  time.Now(),
	})
	return nil
}

func (p *RunPlan) AddNote(member *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("note content must not be empty")
	}
	p.Notes = append(p.Notes, RecordedNote{
		Member:    copyStrPtr(member),
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

func copyStrPtr(v *string) *string {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
