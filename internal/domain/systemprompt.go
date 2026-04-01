package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SystemPrompt struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prompt    string    `json:"prompt"`
	BuiltIn   bool      `json:"built_in"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSystemPrompt(name, prompt string) (*SystemPrompt, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("system prompt name must not be empty")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, errors.New("system prompt text must not be empty")
	}

	now := time.Now()
	return &SystemPrompt{
		ID:        uuid.NewString(),
		Name:      name,
		Prompt:    prompt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *SystemPrompt) Update(name, prompt *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("system prompt name must not be empty")
		}
		s.Name = trimmed
	}
	if prompt != nil {
		trimmed := strings.TrimSpace(*prompt)
		if trimmed == "" {
			return errors.New("system prompt text must not be empty")
		}
		s.Prompt = trimmed
	}
	s.UpdatedAt = time.Now()
	return nil
}
