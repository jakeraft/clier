package team

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// Import creates a complete team from a TeamExport, including all
// sub-resources (system prompts, git repos, cli profiles, members,
// relations). Resources with duplicate names are created once and shared.
func (s *Service) Import(ctx context.Context, export domain.TeamExport) (*domain.Team, error) {
	if err := export.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	promptIDs, err := s.importSystemPrompts(ctx, export)
	if err != nil {
		return nil, fmt.Errorf("import system prompts: %w", err)
	}

	envIDs, err := s.importEnvs(ctx, export)
	if err != nil {
		return nil, fmt.Errorf("import envs: %w", err)
	}

	repoIDs, err := s.importGitRepos(ctx, export)
	if err != nil {
		return nil, fmt.Errorf("import git repos: %w", err)
	}

	profileIDs, err := s.importCliProfiles(ctx, export)
	if err != nil {
		return nil, fmt.Errorf("import cli profiles: %w", err)
	}

	memberIDs, err := s.importMembers(ctx, export, profileIDs, promptIDs, envIDs, repoIDs)
	if err != nil {
		return nil, fmt.Errorf("import members: %w", err)
	}

	rootID := memberIDs[export.RootMemberName]

	team, err := domain.NewTeam(export.TeamName, rootID)
	if err != nil {
		return nil, fmt.Errorf("new team: %w", err)
	}
	if err := s.store.CreateTeam(ctx, team); err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	for _, m := range export.Members {
		if m.Name == export.RootMemberName {
			continue
		}
		if err := s.store.AddTeamMember(ctx, team.ID, memberIDs[m.Name]); err != nil {
			return nil, fmt.Errorf("add member %s: %w", m.Name, err)
		}
	}

	for _, r := range export.Relations {
		fromID := memberIDs[r.From]
		toID := memberIDs[r.To]
		rel := domain.Relation{From: fromID, To: toID, Type: r.Type}
		if err := s.store.AddTeamRelation(ctx, team.ID, rel); err != nil {
			return nil, fmt.Errorf("add relation %s→%s: %w", r.From, r.To, err)
		}
	}

	// Re-read team to return complete state with all members and relations
	complete, err := s.store.GetTeam(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return &complete, nil
}

// importSystemPrompts deduplicates and creates system prompts.
// Returns map[promptName]->promptID.
func (s *Service) importSystemPrompts(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		for _, p := range m.SystemPrompts {
			if _, exists := ids[p.Name]; exists {
				continue
			}
			sp, err := domain.NewSystemPrompt(p.Name, p.Prompt)
			if err != nil {
				return nil, fmt.Errorf("new system prompt %s: %w", p.Name, err)
			}
			if err := s.store.CreateSystemPrompt(ctx, sp); err != nil {
				return nil, fmt.Errorf("create system prompt %s: %w", p.Name, err)
			}
			ids[p.Name] = sp.ID
		}
	}
	return ids, nil
}

// importEnvs deduplicates and creates envs.
// Returns map[envName]->envID.
func (s *Service) importEnvs(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		for _, e := range m.Envs {
			if _, exists := ids[e.Name]; exists {
				continue
			}
			env, err := domain.NewEnv(e.Name, e.Key, e.Value)
			if err != nil {
				return nil, fmt.Errorf("new env %s: %w", e.Name, err)
			}
			if err := s.store.CreateEnv(ctx, env); err != nil {
				return nil, fmt.Errorf("create env %s: %w", e.Name, err)
			}
			ids[e.Name] = env.ID
		}
	}
	return ids, nil
}

// importGitRepos deduplicates and creates git repos.
// Returns map[repoName]->repoID.
func (s *Service) importGitRepos(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		if m.GitRepo == nil {
			continue
		}
		if _, exists := ids[m.GitRepo.Name]; exists {
			continue
		}
		repo, err := domain.NewGitRepo(m.GitRepo.Name, m.GitRepo.URL)
		if err != nil {
			return nil, fmt.Errorf("new git repo %s: %w", m.GitRepo.Name, err)
		}
		if err := s.store.CreateGitRepo(ctx, repo); err != nil {
			return nil, fmt.Errorf("create git repo %s: %w", m.GitRepo.Name, err)
		}
		ids[m.GitRepo.Name] = repo.ID
	}
	return ids, nil
}

// importCliProfiles deduplicates and creates CLI profiles.
// Returns map[profileName]->profileID.
func (s *Service) importCliProfiles(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		p := m.CliProfile
		if _, exists := ids[p.Name]; exists {
			continue
		}
		profile, err := domain.NewCliProfileRaw(p.Name, p.Model, p.Binary, p.SystemArgs, p.CustomArgs, p.DotConfig)
		if err != nil {
			return nil, fmt.Errorf("new cli profile %s: %w", p.Name, err)
		}
		if err := s.store.CreateCliProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("create cli profile %s: %w", p.Name, err)
		}
		ids[p.Name] = profile.ID
	}
	return ids, nil
}

// importMembers creates members, linking to already-created profiles/prompts/repos.
// Returns map[memberName]->memberID.
func (s *Service) importMembers(ctx context.Context, export domain.TeamExport, profileIDs, promptIDs, envIDs, repoIDs map[string]string) (map[string]string, error) {
	ids := make(map[string]string, len(export.Members))
	for _, m := range export.Members {
		profileID := profileIDs[m.CliProfile.Name]

		spIDs := make([]string, 0, len(m.SystemPrompts))
		for _, p := range m.SystemPrompts {
			spIDs = append(spIDs, promptIDs[p.Name])
		}

		eIDs := make([]string, 0, len(m.Envs))
		for _, e := range m.Envs {
			eIDs = append(eIDs, envIDs[e.Name])
		}

		var repoID string
		if m.GitRepo != nil {
			repoID = repoIDs[m.GitRepo.Name]
		}

		member, err := domain.NewMember(m.Name, profileID, spIDs, repoID, eIDs)
		if err != nil {
			return nil, fmt.Errorf("new member %s: %w", m.Name, err)
		}
		if err := s.store.CreateMember(ctx, member); err != nil {
			return nil, fmt.Errorf("create member %s: %w", m.Name, err)
		}
		ids[m.Name] = member.ID
	}
	return ids, nil
}
