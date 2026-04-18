package runtime

import (
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"
)

type Run struct {
	RunID           string
	Session         string
	WorkingCopyPath string
	Agents          []AgentTerminal
	Status          string
	StartedAt       time.Time
	StoppedAt       *time.Time
	Messages        []RecordedMessage
	Notes           []RecordedNote
}

type AgentTerminal struct {
	ID        string
	Name      string
	AgentType string
	Window    int
	Workspace string
	Cwd       string
	Command   string
}

type RecordedMessage struct {
	FromAgent *string
	ToAgent   *string
	Content   string
	CreatedAt time.Time
}

type RecordedNote struct {
	Agent     *string
	Content   string
	CreatedAt time.Time
}

func NewRun(runID, sessionName, workingCopyPath string, plans []AgentTerminal) *Run {
	agentTerminals := make([]AgentTerminal, len(plans))
	copy(agentTerminals, plans)
	return &Run{
		RunID:           runID,
		Session:         sessionName,
		WorkingCopyPath: workingCopyPath,
		Agents:          agentTerminals,
		Status:          StatusRunning,
		StartedAt:       time.Now(),
	}
}

func (r *Run) FindAgent(agentID string) (*AgentTerminal, bool) {
	for i := range r.Agents {
		if r.Agents[i].ID == agentID {
			return &r.Agents[i], true
		}
	}
	return nil, false
}

func (r *Run) AddMessage(fromAgent, toAgent *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return &domain.Fault{Kind: domain.KindContentRequired}
	}
	r.Messages = append(r.Messages, RecordedMessage{
		FromAgent: copyStrPtr(fromAgent),
		ToAgent:   copyStrPtr(toAgent),
		Content:   content,
		CreatedAt: time.Now(),
	})
	return nil
}

func (r *Run) AddNote(agent *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return &domain.Fault{Kind: domain.KindContentRequired}
	}
	r.Notes = append(r.Notes, RecordedNote{
		Agent:     copyStrPtr(agent),
		Content:   content,
		CreatedAt: time.Now(),
	})
	return nil
}

func (r *Run) MarkStopped() {
	now := time.Now()
	r.Status = StatusStopped
	r.StoppedAt = &now
}

func copyStrPtr(v *string) *string {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
