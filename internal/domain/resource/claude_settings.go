package resource

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ClaudeSettings is a settings.json file for Claude Code.
// Written to CLAUDE_CONFIG_DIR/settings.json.
type ClaudeSettings struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeSettings(name, content string) (*ClaudeSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude settings name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude settings content must not be empty")
	}
	if !json.Valid([]byte(content)) {
		return nil, errors.New("claude settings content must be valid JSON")
	}

	now := time.Now()
	return &ClaudeSettings{
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *ClaudeSettings) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude settings name must not be empty")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude settings content must not be empty")
		}
		if !json.Valid([]byte(trimmed)) {
			return errors.New("claude settings content must be valid JSON")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
