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
	Agents    []AgentTerminal   `json:"agents"`
	Status    string            `json:"status"`
	StartedAt time.Time         `json:"started_at"`
	StoppedAt *time.Time        `json:"stopped_at,omitempty"`
	Messages  []RecordedMessage `json:"messages,omitempty"`
	Notes     []RecordedNote    `json:"notes,omitempty"`
}

// AgentTerminal maps an agent to its tmux window and launch command.
type AgentTerminal struct {
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	Window    int    `json:"window"`
	Workspace string `json:"workspace"`
	Cwd       string `json:"cwd"`
	Command   string `json:"command"`
}

type RecordedMessage struct {
	FromAgent *string   `json:"from_agent,omitempty"`
	ToAgent   *string   `json:"to_agent,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type RecordedNote struct {
	Agent     *string   `json:"agent,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// NewPlan builds a RunPlan from concrete terminal launch specs.
func NewPlan(runID, sessionName string, plans []AgentTerminal) *RunPlan {
	agentTerminals := make([]AgentTerminal, len(plans))
	copy(agentTerminals, plans)
	return &RunPlan{
		RunID:     runID,
		Session:   sessionName,
		Agents:    agentTerminals,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}
}

// FindAgent finds the terminal slot for an agent by name in the run plan.
func (p *RunPlan) FindAgent(agentName string) (*AgentTerminal, bool) {
	for i := range p.Agents {
		if p.Agents[i].Name == agentName {
			return &p.Agents[i], true
		}
	}
	return nil, false
}

func (p *RunPlan) AddMessage(fromAgent, toAgent *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("message content must not be empty")
	}
	p.Messages = append(p.Messages, RecordedMessage{
		FromAgent: copyStrPtr(fromAgent),
		ToAgent:   copyStrPtr(toAgent),
		Content:   content,
		CreatedAt: time.Now(),
	})
	return nil
}

func (p *RunPlan) AddNote(agent *string, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("note content must not be empty")
	}
	p.Notes = append(p.Notes, RecordedNote{
		Agent:     copyStrPtr(agent),
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
