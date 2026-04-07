package team

import (
	"context"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// Store defines the operations needed by the team service.
type Store interface {
	// Read
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)

	// Write (used by Import)
	CreateAgentDotMd(ctx context.Context, cm *resource.AgentDotMd) error
	CreateSkill(ctx context.Context, sk *resource.Skill) error
	CreateClaudeSettings(ctx context.Context, st *resource.ClaudeSettings) error
	CreateClaudeJson(ctx context.Context, cj *resource.ClaudeJson) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	UpdateAgentDotMd(ctx context.Context, cm *resource.AgentDotMd) error
	UpdateSkill(ctx context.Context, sk *resource.Skill) error
	UpdateClaudeSettings(ctx context.Context, st *resource.ClaudeSettings) error
	UpdateClaudeJson(ctx context.Context, cj *resource.ClaudeJson) error
	UpdateMember(ctx context.Context, m *domain.Member) error
	UpdateTeam(ctx context.Context, t *domain.Team) error
	AddTeamMember(ctx context.Context, teamID string, tm domain.TeamMember) error
	AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error
	ReplaceTeamComposition(ctx context.Context, t *domain.Team) error
}

// Service provides team-level operations.
type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}
