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
	CreateSystemPrompt(ctx context.Context, sp *resource.SystemPrompt) error
	CreateEnv(ctx context.Context, e *resource.Env) error
	CreateGitRepo(ctx context.Context, r *resource.GitRepo) error
	CreateCliProfile(ctx context.Context, p *resource.CliProfile) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	UpdateSystemPrompt(ctx context.Context, sp *resource.SystemPrompt) error
	UpdateEnv(ctx context.Context, e *resource.Env) error
	UpdateGitRepo(ctx context.Context, r *resource.GitRepo) error
	UpdateCliProfile(ctx context.Context, p *resource.CliProfile) error
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
