package team

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// Store defines the operations needed by the team service.
type Store interface {
	// Read
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetCliProfile(ctx context.Context, id string) (domain.CliProfile, error)
	GetSystemPrompt(ctx context.Context, id string) (domain.SystemPrompt, error)
	GetGitRepo(ctx context.Context, id string) (domain.GitRepo, error)
	GetEnv(ctx context.Context, id string) (domain.Env, error)

	// Write (used by Import)
	CreateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error
	CreateEnv(ctx context.Context, e *domain.Env) error
	CreateGitRepo(ctx context.Context, r *domain.GitRepo) error
	CreateCliProfile(ctx context.Context, p *domain.CliProfile) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	AddTeamMember(ctx context.Context, teamID, memberID string) error
	AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error
}

// Service provides team-level operations.
type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

// Snapshot aggregates a team's complete state from normalised entities.
func (s *Service) Snapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error) {
	team, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("get team: %w", err)
	}

	members := make([]domain.TeamMemberSnapshot, 0, len(team.MemberIDs))
	for _, id := range team.MemberIDs {
		ms, err := s.memberSnapshot(ctx, id)
		if err != nil {
			return domain.TeamSnapshot{}, fmt.Errorf("load member %s: %w", id, err)
		}
		ms.Relations = team.MemberRelations(id)
		members = append(members, ms)
	}

	return domain.TeamSnapshot{
		TeamName:     team.Name,
		RootMemberID: team.RootMemberID,
		Members:      members,
	}, nil
}

// Export returns a self-contained, name-based TeamExport for the given team.
func (s *Service) Export(ctx context.Context, teamID string) (domain.TeamExport, error) {
	snap, err := s.Snapshot(ctx, teamID)
	if err != nil {
		return domain.TeamExport{}, fmt.Errorf("snapshot: %w", err)
	}
	return domain.ExportFromSnapshot(snap)
}

func (s *Service) memberSnapshot(ctx context.Context, memberID string) (domain.TeamMemberSnapshot, error) {
	member, err := s.store.GetMember(ctx, memberID)
	if err != nil {
		return domain.TeamMemberSnapshot{}, fmt.Errorf("get member: %w", err)
	}

	profile, err := s.store.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return domain.TeamMemberSnapshot{}, fmt.Errorf("get cli profile: %w", err)
	}

	prompts := make([]domain.PromptSnapshot, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := s.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return domain.TeamMemberSnapshot{}, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, domain.PromptSnapshot{Name: sp.Name, Prompt: sp.Prompt})
	}

	envs := make([]domain.EnvSnapshot, 0, len(member.EnvIDs))
	for _, id := range member.EnvIDs {
		env, err := s.store.GetEnv(ctx, id)
		if err != nil {
			return domain.TeamMemberSnapshot{}, fmt.Errorf("get env %s: %w", id, err)
		}
		envs = append(envs, domain.EnvSnapshot{Name: env.Name, Key: env.Key, Value: env.Value})
	}

	var gitRepo *domain.GitRepoSnapshot
	if member.GitRepoID != "" {
		repo, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return domain.TeamMemberSnapshot{}, fmt.Errorf("get git repo: %w", err)
		}
		gitRepo = &domain.GitRepoSnapshot{Name: repo.Name, URL: repo.URL}
	}

	return domain.TeamMemberSnapshot{
		MemberID:       memberID,
		MemberName:     member.Name,
		Binary:         profile.Binary,
		Model:          profile.Model,
		CliProfileName: profile.Name,
		SystemArgs:     profile.SystemArgs,
		CustomArgs:     profile.CustomArgs,
		DotConfig:      profile.DotConfig,
		SystemPrompts:  prompts,
		GitRepo:        gitRepo,
		Envs:           envs,
	}, nil
}
