package session

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
)

// buildPlan computes the execution plan from current team state.
// For each TeamMember, loads the member spec, profile, prompts, envs, and repo,
// then builds a MemberPlan directly.
func (s *Service) buildPlan(ctx context.Context, team domain.Team) ([]domain.MemberPlan, error) {
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

		memberspace := fmt.Sprintf("%s/%s/%s", PlaceholderBase, PlaceholderSessionID, tm.ID)

		clierPrompt := buildClierPrompt(team.Name, tm.Name, relations, nameByID)
		userPrompt := joinPrompts(prompts)
		prompt := "---\n\n" + clierPrompt + "\n---\n\n" + userPrompt

		authEnvs := setAuth()

		files, err := buildClaudeFiles(profile.DotConfig, PlaceholderMemberspace+"/project", PlaceholderMemberspace)
		if err != nil {
			return nil, fmt.Errorf("build files for %s: %w", tm.Name, err)
		}

		cmd := buildCommand(
			profile.Model, profile.SystemArgs, profile.CustomArgs,
			prompt, PlaceholderSessionID, tm.ID,
			authEnvs, envs,
		)

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

	return plans, nil
}

