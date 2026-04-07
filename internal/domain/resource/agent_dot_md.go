package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AgentDotMd is a project instruction file shared across agent types.
// Written as CLAUDE.md (Claude), AGENTS.md (Codex), or GEMINI.md (Gemini).
type AgentDotMd struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAgentDotMd(name, content string) (*AgentDotMd, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("agent dot md name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("agent dot md content must not be empty")
	}

	now := time.Now()
	return &AgentDotMd{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (a *AgentDotMd) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("agent dot md name must not be empty")
		}
		a.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("agent dot md content must not be empty")
		}
		a.Content = trimmed
	}
	a.UpdatedAt = time.Now()
	return nil
}
