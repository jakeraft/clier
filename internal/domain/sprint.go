package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SprintState string

const (
	SprintRunning   SprintState = "running"
	SprintCompleted SprintState = "completed"
	SprintErrored   SprintState = "errored"
)

type Sprint struct {
	ID           string
	TeamSnapshot TeamSnapshot
	Name         string
	State        SprintState
	Error        string // empty string means no error
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewSprint(snapshot TeamSnapshot) *Sprint {
	id := uuid.NewString()
	now := time.Now()
	return &Sprint{
		ID:           id,
		TeamSnapshot: snapshot,
		Name:         fmt.Sprintf("%s_%s", snapshot.TeamName, id[:8]),
		State:        SprintRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (s *Sprint) Complete() error {
	if s.State != SprintRunning {
		return fmt.Errorf("cannot complete sprint in state %q", s.State)
	}
	s.State = SprintCompleted
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Sprint) Fail(errMsg string) error {
	if s.State != SprintRunning {
		return fmt.Errorf("cannot fail sprint in state %q", s.State)
	}
	s.State = SprintErrored
	s.Error = errMsg
	s.UpdatedAt = time.Now()
	return nil
}
