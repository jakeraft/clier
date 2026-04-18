package view

import (
	"time"

	apprun "github.com/jakeraft/clier/internal/app/run"
)

type RunList struct {
	Items []RunSummary `json:"items"`
}

type RunStartResult struct {
	RunID   string `json:"run_id"`
	Session string `json:"session"`
	Hint    string `json:"hint,omitempty"`
}

type RunStopResult struct {
	Stopped string `json:"stopped"`
}

type RunTellResult struct {
	Status string  `json:"status"`
	From   *string `json:"from"`
	To     *string `json:"to"`
	Run    string  `json:"run"`
}

type RunNoteResult struct {
	Status string  `json:"status"`
	Agent  *string `json:"agent"`
	Run    string  `json:"run"`
}

type RunSummary struct {
	RunID           string     `json:"run_id"`
	Session         string     `json:"session"`
	WorkingCopyPath string     `json:"working_copy_path"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at"`
	AgentCount      int        `json:"agent_count"`
}

type RunDetail struct {
	RunID           string       `json:"run_id"`
	Session         string       `json:"session"`
	WorkingCopyPath string       `json:"working_copy_path"`
	Status          string       `json:"status"`
	StartedAt       time.Time    `json:"started_at"`
	StoppedAt       *time.Time   `json:"stopped_at"`
	Agents          []RunAgent   `json:"agents"`
	Messages        []RunMessage `json:"messages"`
	Notes           []RunNote    `json:"notes"`
}

type RunAgent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	Window    int    `json:"window"`
	Workspace string `json:"workspace"`
	Cwd       string `json:"cwd"`
}

type RunMessage struct {
	FromAgent *string   `json:"from_agent"`
	ToAgent   *string   `json:"to_agent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type RunNote struct {
	Agent     *string   `json:"agent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func RunListOf(plans []*apprun.RunPlan) RunList {
	items := make([]RunSummary, 0, len(plans))
	for _, plan := range plans {
		items = append(items, RunSummary{
			RunID:           plan.RunID,
			Session:         plan.Session,
			WorkingCopyPath: plan.WorkingCopyPath,
			Status:          plan.Status,
			StartedAt:       plan.StartedAt,
			StoppedAt:       plan.StoppedAt,
			AgentCount:      len(plan.Agents),
		})
	}
	return RunList{Items: items}
}

func RunStartOf(runID, session string, hint *string) RunStartResult {
	result := RunStartResult{
		RunID:   runID,
		Session: session,
	}
	if hint != nil {
		result.Hint = *hint
	}
	return result
}

func RunStopOf(runID string) RunStopResult {
	return RunStopResult{Stopped: runID}
}

func RunTellOf(runID string, fromAgent, toAgent *string) RunTellResult {
	return RunTellResult{
		Status: "delivered",
		From:   fromAgent,
		To:     toAgent,
		Run:    runID,
	}
}

func RunNoteOf(runID string, agent *string) RunNoteResult {
	return RunNoteResult{
		Status: "posted",
		Agent:  agent,
		Run:    runID,
	}
}

func RunDetailOf(plan *apprun.RunPlan) RunDetail {
	agents := make([]RunAgent, 0, len(plan.Agents))
	for _, agent := range plan.Agents {
		agents = append(agents, RunAgent{
			ID:        agent.ID,
			Name:      agent.Name,
			AgentType: agent.AgentType,
			Window:    agent.Window,
			Workspace: agent.Workspace,
			Cwd:       agent.Cwd,
		})
	}

	messages := make([]RunMessage, 0, len(plan.Messages))
	for _, message := range plan.Messages {
		messages = append(messages, RunMessage{
			FromAgent: message.FromAgent,
			ToAgent:   message.ToAgent,
			Content:   message.Content,
			CreatedAt: message.CreatedAt,
		})
	}

	notes := make([]RunNote, 0, len(plan.Notes))
	for _, note := range plan.Notes {
		notes = append(notes, RunNote{
			Agent:     note.Agent,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
		})
	}

	return RunDetail{
		RunID:           plan.RunID,
		Session:         plan.Session,
		WorkingCopyPath: plan.WorkingCopyPath,
		Status:          plan.Status,
		StartedAt:       plan.StartedAt,
		StoppedAt:       plan.StoppedAt,
		Agents:          agents,
		Messages:        messages,
		Notes:           notes,
	}
}
