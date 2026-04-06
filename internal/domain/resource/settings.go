package resource

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Settings is a settings.json file that gets written to CLAUDE_CONFIG_DIR/settings.json.
// Maps 1:1 to the Claude Code settings.json.
type Settings struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSettings(name, content string) (*Settings, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("settings name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("settings content must not be empty")
	}
	if !json.Valid([]byte(content)) {
		return nil, errors.New("settings content must be valid JSON")
	}

	now := time.Now()
	return &Settings{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Settings) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("settings name must not be empty")
		}
		s.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("settings content must not be empty")
		}
		if !json.Valid([]byte(trimmed)) {
			return errors.New("settings content must be valid JSON")
		}
		s.Content = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
