package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Env struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewEnv(name, key, value string) (*Env, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("env name must not be empty")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("env key must not be empty")
	}

	now := time.Now()
	return &Env{
		ID:        uuid.NewString(),
		Name:      name,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (e *Env) Update(name, key, value *string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("env name must not be empty")
		}
		e.Name = trimmed
	}
	if key != nil {
		trimmed := strings.TrimSpace(*key)
		if trimmed == "" {
			return errors.New("env key must not be empty")
		}
		e.Key = trimmed
	}
	if value != nil {
		e.Value = *value
	}
	e.UpdatedAt = time.Now()
	return nil
}
