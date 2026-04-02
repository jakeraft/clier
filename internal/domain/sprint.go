package domain

import (
	"errors"
	"fmt"
	"time"
)

// UserMemberID is the reserved member ID for the human user who started the sprint.
const UserMemberID = "00000000-0000-0000-0000-000000000000"

type Sprint struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Snapshot  SprintSnapshot `json:"snapshot"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func NewSprint(id string, snapshot SprintSnapshot) (*Sprint, error) {
	if id == "" {
		return nil, errors.New("sprint id must not be empty")
	}
	if snapshot.TeamName == "" {
		return nil, errors.New("snapshot team name must not be empty")
	}
	if snapshot.RootMemberID == "" {
		return nil, errors.New("snapshot root member id must not be empty")
	}

	now := time.Now()
	return &Sprint{
		ID:        id,
		Name:      fmt.Sprintf("%s_%s", snapshot.TeamName, id[:8]),
		Snapshot:  snapshot,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
