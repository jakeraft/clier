package runplan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	domainruntime "github.com/jakeraft/clier/internal/domain/runtime"
)

const RunsDirName = ".runs"

type record struct {
	RunID           string          `json:"run_id"`
	Session         string          `json:"session"`
	WorkingCopyPath string          `json:"working_copy_path"`
	Agents          []agentRecord   `json:"agents"`
	Status          string          `json:"status"`
	StartedAt       time.Time       `json:"started_at"`
	StoppedAt       *time.Time      `json:"stopped_at,omitempty"`
	Messages        []messageRecord `json:"messages,omitempty"`
	Notes           []noteRecord    `json:"notes,omitempty"`
}

type agentRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AgentType string `json:"agent_type"`
	Window    int    `json:"window"`
	Workspace string `json:"workspace"`
	Cwd       string `json:"cwd"`
	Command   string `json:"command"`
}

type messageRecord struct {
	FromAgent *string   `json:"from_agent,omitempty"`
	ToAgent   *string   `json:"to_agent,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type noteRecord struct {
	Agent     *string   `json:"agent,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func Path(runsDir, runID string) string {
	return filepath.Join(runsDir, runID+".json")
}

func Save(runsDir, runID string, run *domainruntime.Run) error {
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return fmt.Errorf("create runs dir: %w", err)
	}

	data, err := json.MarshalIndent(recordFromDomain(run), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run plan: %w", err)
	}
	return os.WriteFile(Path(runsDir, runID), data, 0o644)
}

func Load(runsDir, runID string) (*domainruntime.Run, error) {
	return LoadFromPath(Path(runsDir, runID))
}

func LoadFromPath(path string) (*domainruntime.Run, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan: %w", err)
	}
	var rec record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}
	return rec.toDomain(), nil
}

func List(runsDir string) ([]*domainruntime.Run, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*domainruntime.Run{}, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}
	runs := make([]*domainruntime.Run, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		run, err := LoadFromPath(filepath.Join(runsDir, name))
		if err != nil {
			return nil, fmt.Errorf("load run plan %s: %w", name, err)
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func recordFromDomain(run *domainruntime.Run) record {
	agents := make([]agentRecord, 0, len(run.Agents))
	for _, agent := range run.Agents {
		agents = append(agents, agentRecord{
			ID:        agent.ID,
			Name:      agent.Name,
			AgentType: agent.AgentType,
			Window:    agent.Window,
			Workspace: agent.Workspace,
			Cwd:       agent.Cwd,
			Command:   agent.Command,
		})
	}
	messages := make([]messageRecord, 0, len(run.Messages))
	for _, msg := range run.Messages {
		messages = append(messages, messageRecord{
			FromAgent: msg.FromAgent,
			ToAgent:   msg.ToAgent,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}
	notes := make([]noteRecord, 0, len(run.Notes))
	for _, note := range run.Notes {
		notes = append(notes, noteRecord{
			Agent:     note.Agent,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
		})
	}
	return record{
		RunID:           run.RunID,
		Session:         run.Session,
		WorkingCopyPath: run.WorkingCopyPath,
		Agents:          agents,
		Status:          run.Status,
		StartedAt:       run.StartedAt,
		StoppedAt:       run.StoppedAt,
		Messages:        messages,
		Notes:           notes,
	}
}

func (r record) toDomain() *domainruntime.Run {
	agents := make([]domainruntime.AgentTerminal, 0, len(r.Agents))
	for _, agent := range r.Agents {
		agents = append(agents, domainruntime.AgentTerminal{
			ID:        agent.ID,
			Name:      agent.Name,
			AgentType: agent.AgentType,
			Window:    agent.Window,
			Workspace: agent.Workspace,
			Cwd:       agent.Cwd,
			Command:   agent.Command,
		})
	}
	messages := make([]domainruntime.RecordedMessage, 0, len(r.Messages))
	for _, msg := range r.Messages {
		messages = append(messages, domainruntime.RecordedMessage{
			FromAgent: msg.FromAgent,
			ToAgent:   msg.ToAgent,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}
	notes := make([]domainruntime.RecordedNote, 0, len(r.Notes))
	for _, note := range r.Notes {
		notes = append(notes, domainruntime.RecordedNote{
			Agent:     note.Agent,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
		})
	}
	return &domainruntime.Run{
		RunID:           r.RunID,
		Session:         r.Session,
		WorkingCopyPath: r.WorkingCopyPath,
		Agents:          agents,
		Status:          r.Status,
		StartedAt:       r.StartedAt,
		StoppedAt:       r.StoppedAt,
		Messages:        messages,
		Notes:           notes,
	}
}
