package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewEnvironment(name, key, value string) (*Environment, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("environment name must not be empty")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("environment key must not be empty")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("environment value must not be empty")
	}

	now := time.Now()
	return &Environment{
		ID:        uuid.NewString(),
		Name:      name,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (e *Environment) Update(name, key, value *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return fmt.Errorf("environment name must not be empty")
		}
		e.Name = trimmed
	}
	if key != nil {
		trimmed := strings.TrimSpace(*key)
		if trimmed == "" {
			return fmt.Errorf("environment key must not be empty")
		}
		e.Key = trimmed
	}
	if value != nil {
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			return fmt.Errorf("environment value must not be empty")
		}
		e.Value = trimmed
	}
	e.UpdatedAt = time.Now()
	return nil
}
