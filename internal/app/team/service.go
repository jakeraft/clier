package team

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderSessionID   = "{{CLIER_SESSION_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
	PlaceholderAuthCodex   = "{{CLIER_AUTH_CODEX}}"
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
	AddTeamMember(ctx context.Context, teamID string, tm domain.TeamMember) error
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

// BuildPlan computes the execution plan from current team state and persists it.
// For each TeamMember, loads the member spec, profile, prompts, envs, and repo,
// then builds a MemberPlan directly.
func (s *Service) BuildPlan(ctx context.Context, teamID string) (*domain.Team, error) {
	team, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}

	// Build nameByID using TeamMember IDs.
	nameByID := make(map[string]string, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		nameByID[tm.ID] = tm.Name
	}

	plans := make([]domain.MemberPlan, 0, len(team.TeamMembers))

	for _, tm := range team.TeamMembers {
		member, err := s.store.GetMember(ctx, tm.MemberID)
		if err != nil {
			return nil, fmt.Errorf("get member %s: %w", tm.MemberID, err)
		}

		profile, err := s.store.GetCliProfile(ctx, member.CliProfileID)
		if err != nil {
			return nil, fmt.Errorf("get cli profile for %s: %w", tm.Name, err)
		}

		prompts := make([]domain.PromptSnapshot, 0, len(member.SystemPromptIDs))
		for _, id := range member.SystemPromptIDs {
			sp, err := s.store.GetSystemPrompt(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("get prompt %s: %w", id, err)
			}
			prompts = append(prompts, domain.PromptSnapshot{ID: sp.ID, Name: sp.Name, Prompt: sp.Prompt})
		}

		envs := make([]domain.EnvSnapshot, 0, len(member.EnvIDs))
		for _, id := range member.EnvIDs {
			env, err := s.store.GetEnv(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("get env %s: %w", id, err)
			}
			envs = append(envs, domain.EnvSnapshot{ID: env.ID, Name: env.Name, Key: env.Key, Value: env.Value})
		}

		var gitRepo *domain.GitRepoRef
		if member.GitRepoID != "" {
			repo, err := s.store.GetGitRepo(ctx, member.GitRepoID)
			if err != nil {
				return nil, fmt.Errorf("get git repo for %s: %w", tm.Name, err)
			}
			gitRepo = &domain.GitRepoRef{Name: repo.Name, URL: repo.URL}
		}

		// Build relations using TeamMember ID.
		relations := team.MemberRelations(tm.ID)

		memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, teamID, tm.ID)

		clierPrompt := buildClierPrompt(team.Name, tm.Name, relations, nameByID)
		userPrompt := joinPrompts(prompts)
		prompt := "---\n\n" + clierPrompt + "\n---\n\n" + userPrompt

		auth := setAuth(profile.Binary)

		files, err := buildFiles(profile.Binary, profile.DotConfig, PlaceholderMemberspace)
		if err != nil {
			return nil, fmt.Errorf("build files for %s: %w", tm.Name, err)
		}
		files = append(files, auth.Files...)

		cmd, err := buildCommand(
			profile.Binary, profile.Model, profile.SystemArgs, profile.CustomArgs,
			prompt, PlaceholderSessionID, tm.ID,
			auth.CommandEnvs, envs,
		)
		if err != nil {
			return nil, fmt.Errorf("build command for %s: %w", tm.Name, err)
		}

		launchPath := PlaceholderMemberspace + "/launch.sh"
		files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

		plans = append(plans, domain.MemberPlan{
			TeamMemberID: tm.ID,
			MemberName:   tm.Name,
			Terminal:     domain.TerminalPlan{Command: ". " + launchPath},
			Workspace: domain.WorkspacePlan{
				Memberspace: memberspace,
				Files:       files,
				GitRepo:     gitRepo,
			},
		})
	}

	team.Plan = plans
	if err := s.store.UpdateTeamPlan(ctx, &team); err != nil {
		return nil, fmt.Errorf("update team plan: %w", err)
	}
	return &team, nil
}

// buildFiles dispatches to the binary-specific config file builder.
func buildFiles(binary domain.CliBinary, dotConfig domain.DotConfig,
	memberspacePlaceholder string) ([]domain.FileEntry, error) {

	workDir := memberspacePlaceholder + "/project"

	switch binary {
	case domain.BinaryClaude:
		return buildClaudeFiles(dotConfig, workDir, memberspacePlaceholder)
	case domain.BinaryCodex:
		return buildCodexFiles(dotConfig, workDir, memberspacePlaceholder)
	default:
		return nil, fmt.Errorf("unknown binary: %s", binary)
	}
}
