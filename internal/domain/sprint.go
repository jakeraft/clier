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

	// UserMemberID is the reserved member ID for the human user who started the sprint.
	UserMemberID = "00000000-0000-0000-0000-000000000000"
)

type Sprint struct {
	ID           string       `json:"id"`
	TeamSnapshot TeamSnapshot `json:"team_snapshot"`
	Name         string       `json:"name"`
	State        SprintState  `json:"state"`
	Error        string       `json:"error"` // empty string means no error
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func NewSprint(snapshot TeamSnapshot) (*Sprint, error) {
	if snapshot.TeamName == "" {
		return nil, fmt.Errorf("snapshot team name must not be empty")
	}
	if snapshot.RootMemberID == "" {
		return nil, fmt.Errorf("snapshot root member id must not be empty")
	}

	id := uuid.NewString()
	now := time.Now()
	return &Sprint{
		ID:           id,
		TeamSnapshot: snapshot,
		Name:         fmt.Sprintf("%s_%s", snapshot.TeamName, id[:8]),
		State:        SprintRunning,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
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
