package api

import (
	"context"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
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
		Plan:      resp.Plan,
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

func (s *Store) GetTeam(_ context.Context, id string) (domain.Team, error) {
	resp, err := s.client.GetTeam(s.owner, id)
	if err != nil {
		return domain.Team{}, err
	}
	members := make([]domain.TeamMember, 0, len(resp.TeamMembers))
	for _, tm := range resp.TeamMembers {
		members = append(members, domain.TeamMember{
			ID:       tm.ID,
			MemberID: tm.MemberID,
			Name:     tm.Name,
		})
	}
	relations := make([]domain.Relation, 0, len(resp.Relations))
	for _, r := range resp.Relations {
		relations = append(relations, domain.Relation{From: r.From, To: r.To})
	}
	return domain.Team{
		ID:               resp.ID,
		Name:             resp.Name,
		RootTeamMemberID: resp.RootTeamMemberID,
		TeamMembers:      members,
		Relations:        relations,
		CreatedAt:        resp.CreatedAt,
		UpdatedAt:        resp.UpdatedAt,
	}, nil
}

func (s *Store) GetMember(_ context.Context, id string) (domain.Member, error) {
	resp, err := s.client.GetMember(s.owner, id)
	if err != nil {
		return domain.Member{}, err
	}
	skillIDs := resp.SkillIDs
	if skillIDs == nil {
		skillIDs = []string{}
	}
	return domain.Member{
		ID:               resp.ID,
		Name:             resp.Name,
		Command:          resp.Command,
		ClaudeMdID:       resp.ClaudeMdID,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: resp.ClaudeSettingsID,
		GitRepoURL:       resp.GitRepoURL,
		CreatedAt:        resp.CreatedAt,
		UpdatedAt:        resp.UpdatedAt,
	}, nil
}

func (s *Store) GetClaudeMd(_ context.Context, id string) (resource.ClaudeMd, error) {
	resp, err := s.client.GetClaudeMd(s.owner, id)
	if err != nil {
		return resource.ClaudeMd{}, err
	}
	return resource.ClaudeMd{
		ID:        resp.ID,
		Name:      resp.Name,
		Content:   resp.Content,
		CreatedAt: resp.CreatedAt,
		UpdatedAt: resp.UpdatedAt,
	}, nil
}

func (s *Store) GetSkill(_ context.Context, id string) (resource.Skill, error) {
	resp, err := s.client.GetSkill(s.owner, id)
	if err != nil {
		return resource.Skill{}, err
	}
	return resource.Skill{
		ID:        resp.ID,
		Name:      resp.Name,
		Content:   resp.Content,
		CreatedAt: resp.CreatedAt,
		UpdatedAt: resp.UpdatedAt,
	}, nil
}

func (s *Store) GetClaudeSettings(_ context.Context, id string) (resource.ClaudeSettings, error) {
	resp, err := s.client.GetClaudeSettings(s.owner, id)
	if err != nil {
		return resource.ClaudeSettings{}, err
	}
	return resource.ClaudeSettings{
		ID:        resp.ID,
		Name:      resp.Name,
		Content:   resp.Content,
		CreatedAt: resp.CreatedAt,
		UpdatedAt: resp.UpdatedAt,
	}, nil
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
