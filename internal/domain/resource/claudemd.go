package resource

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeMd is a CLAUDE.md file that gets written to {workspace}/project/CLAUDE.md.
// Maps 1:1 to the Claude Code project-level CLAUDE.md.
type ClaudeMd struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewClaudeMd(name, content string) (*ClaudeMd, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("claude md name must not be empty")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("claude md content must not be empty")
	}

	now := time.Now()
	return &ClaudeMd{
		ID:        uuid.NewString(),
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c *ClaudeMd) Update(name, content *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("claude md name must not be empty")
		}
		c.Name = trimmed
	}
	if content != nil {
		trimmed := strings.TrimSpace(*content)
		if trimmed == "" {
			return errors.New("claude md content must not be empty")
		}
		c.Content = trimmed
	}
	c.UpdatedAt = time.Now()
	return nil
}
