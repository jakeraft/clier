package team

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/app/runplan"
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
	UpdateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error
	UpdateEnv(ctx context.Context, e *domain.Env) error
	UpdateGitRepo(ctx context.Context, r *domain.GitRepo) error
	UpdateCliProfile(ctx context.Context, p *domain.CliProfile) error
	UpdateMember(ctx context.Context, m *domain.Member) error
	UpdateTeam(ctx context.Context, t *domain.Team) error
	AddTeamMember(ctx context.Context, teamID, memberID string) error
	AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error
	ReplaceTeamComposition(ctx context.Context, t *domain.Team) error
	UpdateTeamPlan(ctx context.Context, t *domain.Team) error
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
		TeamID:       team.ID,
		TeamName:     team.Name,
		RootMemberID: team.RootMemberID,
		Members:      members,
	}, nil
}

// BuildPlan computes the execution plan from current team state and persists it.
func (s *Service) BuildPlan(ctx context.Context, teamID string) (*domain.Team, error) {
	snap, err := s.Snapshot(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}

	td := snapshotToTeamData(snap)
	plan, err := runplan.BuildPlan(td)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	t, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	t.Plan = plan
	if err := s.store.UpdateTeamPlan(ctx, &t); err != nil {
		return nil, fmt.Errorf("update team plan: %w", err)
	}
	return &t, nil
}

// snapshotToTeamData converts a TeamSnapshot into runplan.TeamData.
func snapshotToTeamData(snap domain.TeamSnapshot) runplan.TeamData {
	members := make([]runplan.MemberData, 0, len(snap.Members))
	for _, m := range snap.Members {
		members = append(members, runplan.MemberData{
			MemberID:      m.MemberID,
			MemberName:    m.MemberName,
			Binary:        m.Binary,
			Model:         m.Model,
			SystemArgs:    m.SystemArgs,
			CustomArgs:    m.CustomArgs,
			DotConfig:     m.DotConfig,
			SystemPrompts: m.SystemPrompts,
			GitRepo:       m.GitRepo,
			Envs:          m.Envs,
			Relations:     m.Relations,
		})
	}
	return runplan.TeamData{
		TeamID:       snap.TeamID,
		TeamName:     snap.TeamName,
		RootMemberID: snap.RootMemberID,
		Members:      members,
	}
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
		prompts = append(prompts, domain.PromptSnapshot{ID: sp.ID, Name: sp.Name, Prompt: sp.Prompt})
	}

	envs := make([]domain.EnvSnapshot, 0, len(member.EnvIDs))
	for _, id := range member.EnvIDs {
		env, err := s.store.GetEnv(ctx, id)
		if err != nil {
			return domain.TeamMemberSnapshot{}, fmt.Errorf("get env %s: %w", id, err)
		}
		envs = append(envs, domain.EnvSnapshot{ID: env.ID, Name: env.Name, Key: env.Key, Value: env.Value})
	}

	var gitRepo *domain.GitRepoSnapshot
	if member.GitRepoID != "" {
		repo, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return domain.TeamMemberSnapshot{}, fmt.Errorf("get git repo: %w", err)
		}
		gitRepo = &domain.GitRepoSnapshot{ID: repo.ID, Name: repo.Name, URL: repo.URL}
	}

	return domain.TeamMemberSnapshot{
		MemberID:       memberID,
		MemberName:     member.Name,
		Binary:         profile.Binary,
		Model:          profile.Model,
		CliProfileID:   profile.ID,
		CliProfileName: profile.Name,
		SystemArgs:     profile.SystemArgs,
		CustomArgs:     profile.CustomArgs,
		DotConfig:      profile.DotConfig,
		SystemPrompts:  prompts,
		GitRepo:        gitRepo,
		Envs:           envs,
	}, nil
}
