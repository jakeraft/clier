package api

import (
	"context"
	"strconv"

	"github.com/jakeraft/clier/internal/domain"
)

// Store wraps the API Client to implement the RunStore interface
// used by the run service. The owner field is resolved from configuration
// at startup.
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
	body := map[string]any{
		"name": r.Name,
	}
	if r.TeamID != nil {
		body["team_id"] = *r.TeamID
	}
	if r.MemberID != nil {
		body["member_id"] = *r.MemberID
	}
	resp, err := s.client.CreateRun(body)
	if err != nil {
		return err
	}
	// Update the domain run with server-assigned values.
	r.ID = strconv.FormatInt(resp.ID, 10)
	r.UserID = resp.UserID
	r.Status = domain.RunStatus(resp.Status)
	r.StartedAt = resp.StartedAt
	return nil
}

func (s *Store) GetRun(_ context.Context, id string) (domain.Run, error) {
	resp, err := s.client.GetRun(id)
	if err != nil {
		return domain.Run{}, err
	}
	return domain.Run{
		ID:        strconv.FormatInt(resp.ID, 10),
		UserID:    resp.UserID,
		Name:      resp.Name,
		TeamID:    resp.TeamID,
		MemberID:  resp.MemberID,
		Status:    domain.RunStatus(resp.Status),
		StartedAt: resp.StartedAt,
		StoppedAt: resp.StoppedAt,
	}, nil
}

func (s *Store) UpdateRunStatus(_ context.Context, r *domain.Run) error {
	return s.client.UpdateRunStatus(r.ID, map[string]any{
		"status": string(r.Status),
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
