package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeJson is a .claude.json file that gets written to CLAUDE_CONFIG_DIR/.claude.json.
// Maps 1:1 to the Claude Code .claude.json project config.
type ClaudeJson struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeJson(name, content string) (*ClaudeJson, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude json name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude json content must not be empty")
	}

	now := time.Now()
	return &ClaudeJson{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c *ClaudeJson) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude json name must not be empty")
		}
		c.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude json content must not be empty")
		}
		c.Content = trimmed
	}
	c.UpdatedAt = time.Now()
	return nil
}
