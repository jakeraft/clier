package task

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

const (
	PlaceholderBase        = "{{CLIER_BASE}}"
	PlaceholderMemberspace = "{{CLIER_MEMBERSPACE}}"
	PlaceholderTaskID      = "{{CLIER_TASK_ID}}"
	PlaceholderAuthClaude  = "{{CLIER_AUTH_CLAUDE}}"
)

// resolveTeam loads all referenced resources for every team member.
// This is the resolve phase: ID strings -> actual domain objects.
func (s *Service) resolveTeam(ctx context.Context, team domain.Team) (*domain.ResolvedTeam, error) {
	members := make([]domain.ResolvedMember, 0, len(team.TeamMembers))
	for _, tm := range team.TeamMembers {
		rm, err := s.resolveMember(ctx, &team, tm)
		if err != nil {
			return nil, err
		}
		members = append(members, *rm)
	}
	return &domain.ResolvedTeam{Team: team, Members: members}, nil
}

// resolveMember loads the member spec and all its referenced resources.
func (s *Service) resolveMember(ctx context.Context, team *domain.Team, tm domain.TeamMember) (*domain.ResolvedMember, error) {
	member, err := s.store.GetMember(ctx, tm.MemberID)
	if err != nil {
		return nil, fmt.Errorf("get member %s: %w", tm.MemberID, err)
	}

	profile, err := s.store.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return nil, fmt.Errorf("get cli profile for %s: %w", tm.Name, err)
	}

	prompts := make([]resource.SystemPrompt, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := s.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, sp)
	}

	envs := make([]resource.Env, 0, len(member.EnvIDs))
	for _, id := range member.EnvIDs {
		env, err := s.store.GetEnv(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get env %s: %w", id, err)
		}
		envs = append(envs, env)
	}

	var repo *resource.GitRepo
	if member.GitRepoID != "" {
		r, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return nil, fmt.Errorf("get git repo for %s: %w", tm.Name, err)
		}
		repo = &r
	}

	relations := team.MemberRelations(tm.ID)

	return &domain.ResolvedMember{
		TeamMemberID: tm.ID,
		Name:         tm.Name,
		Profile:      profile,
		Prompts:      prompts,
		Envs:         envs,
		Repo:         repo,
		Relations:    relations,
	}, nil
}

// buildPlans constructs MemberPlans from a resolved team.
// This is the build phase: resolved objects -> execution plan with placeholders.
func buildPlans(resolved *domain.ResolvedTeam, taskID string) ([]domain.MemberPlan, error) {
	nameByID := make(map[string]string, len(resolved.Members))
	for _, rm := range resolved.Members {
		nameByID[rm.TeamMemberID] = rm.Name
	}

	plans := make([]domain.MemberPlan, 0, len(resolved.Members))
	for _, rm := range resolved.Members {
		plan := buildMemberPlan(&rm, nameByID, resolved.Name, taskID)
		plans = append(plans, plan)
	}
	return plans, nil
}

// buildMemberPlan constructs a single MemberPlan from a resolved member.
func buildMemberPlan(rm *domain.ResolvedMember, nameByID map[string]string, teamName, taskID string) domain.MemberPlan {
	memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, PlaceholderTaskID, rm.TeamMemberID)

	clierPrompt := buildClierPrompt(teamName, rm.Name, rm.Relations, nameByID)
	userPrompt := joinPrompts(rm.Prompts)
	prompt := "---\n\n" + clierPrompt + "\n---\n\n" + userPrompt

	files := buildClaudeFiles(rm.Profile.SettingsJSON, rm.Profile.ClaudeJSON, PlaceholderMemberspace)

	cmd := buildCommand(rm.Profile, prompt, teamName, rm.Name, taskID, rm.TeamMemberID, rm.Envs)

	launchPath := PlaceholderMemberspace + "/launch.sh"
	files = append(files, domain.FileEntry{Path: launchPath, Content: cmd})

	var gitRepo *domain.GitRepoRef
	if rm.Repo != nil {
		gitRepo = &domain.GitRepoRef{Name: rm.Repo.Name, URL: rm.Repo.URL}
	}

	return domain.MemberPlan{
		TeamMemberID: rm.TeamMemberID,
		MemberName:   rm.Name,
		Terminal:     domain.TerminalPlan{Command: ". " + launchPath},
		Workspace: domain.WorkspacePlan{
			Memberspace: memberspace,
			Files:       files,
			GitRepo:     gitRepo,
		},
	}
}
