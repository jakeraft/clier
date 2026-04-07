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
	CreateClaudeMd(ctx context.Context, cm *resource.ClaudeMd) error
	CreateSkill(ctx context.Context, sk *resource.Skill) error
	CreateSettings(ctx context.Context, st *resource.Settings) error
	CreateClaudeJson(ctx context.Context, cj *resource.ClaudeJson) error
	CreateGitRepo(ctx context.Context, r *resource.GitRepo) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	UpdateClaudeMd(ctx context.Context, cm *resource.ClaudeMd) error
	UpdateSkill(ctx context.Context, sk *resource.Skill) error
	UpdateSettings(ctx context.Context, st *resource.Settings) error
	UpdateClaudeJson(ctx context.Context, cj *resource.ClaudeJson) error
	UpdateGitRepo(ctx context.Context, r *resource.GitRepo) error
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
