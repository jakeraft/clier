package api

import (
	"context"

	"github.com/jakeraft/clier/internal/domain"
)

// Store wraps the API Client to implement the RunStore and RefStore interfaces
// used by the run service and terminal adapter. The owner field is resolved
// from configuration at startup.
type Store struct {
	client *Client
	owner  string
}

// NewStore creates an API-backed store.
func NewStore(client *Client, owner string) *Store {
	return &Store{client: client, owner: owner}
}

// --- RunStore interface (used by internal/app/run) ---

func (s *Store) CreateRun(_ context.Context, r *domain.Run) error {
	_, err := s.client.CreateRun(r)
	return err
}

func (s *Store) GetRun(_ context.Context, id string) (domain.Run, error) {
	resp, err := s.client.GetRun(id)
	if err != nil {
		return domain.Run{}, err
	}
	return domain.Run{
		ID:        resp.ID,
		Name:      resp.Name,
		TeamID:    resp.TeamID,
		Status:    resp.Status,
		StartedAt: resp.StartedAt,
		StoppedAt: resp.StoppedAt,
	}, nil
}

func (s *Store) UpdateRunStatus(_ context.Context, r *domain.Run) error {
	return s.client.UpdateRunStatus(r.ID, map[string]any{
		"status":     string(r.Status),
		"stopped_at": r.StoppedAt,
	})
}

func (s *Store) CreateMessage(_ context.Context, msg *domain.Message) error {
	_, err := s.client.AddMessage(msg.RunID, msg)
	return err
}

func (s *Store) CreateNote(_ context.Context, n *domain.Note) error {
	_, err := s.client.AddNote(n.RunID, n)
	return err
}

// --- RefStore interface (used by internal/adapter/terminal) ---

func (s *Store) SaveRefs(_ context.Context, runID, memberID string, refs map[string]string) error {
	return s.client.SaveTerminalRefs(runID, memberID, refs)
}

func (s *Store) GetRefs(_ context.Context, runID, memberID string) (map[string]string, error) {
	return s.client.GetTerminalRefs(runID, memberID)
}

func (s *Store) GetRunRefs(_ context.Context, runID string) (map[string]string, error) {
	return s.client.GetRunTerminalRefs(runID)
}

func (s *Store) DeleteRefs(_ context.Context, runID string) error {
	return s.client.DeleteTerminalRefs(runID)
}
