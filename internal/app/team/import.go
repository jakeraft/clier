package team

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// Import creates or updates a complete team from a TeamExport, including all
// sub-resources (system prompts, git repos, cli profiles, members,
// relations). Resources whose ID already exists in the store are updated.
// Resources with duplicate names are created once and shared.
func (s *Service) Import(ctx context.Context, export domain.TeamExport) (*domain.Team, error) {
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
	teamID, err := s.importTeam(ctx, export, rootID, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("import team: %w", err)
	}

	complete, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return &complete, nil
}

// importTeam creates or updates the team, its member list, and relations.
func (s *Service) importTeam(ctx context.Context, export domain.TeamExport, rootID string, memberIDs map[string]string) (string, error) {
	// Build member ID list and relations from the export.
	allMemberIDs := make([]string, 0, len(export.Members))
	allMemberIDs = append(allMemberIDs, rootID) // root first
	for _, m := range export.Members {
		if m.Name == export.RootMemberName {
			continue
		}
		allMemberIDs = append(allMemberIDs, memberIDs[m.Name])
	}

	relations := make([]domain.Relation, 0, len(export.Relations))
	for _, r := range export.Relations {
		relations = append(relations, domain.Relation{
			From: memberIDs[r.From], To: memberIDs[r.To], Type: r.Type,
		})
	}

	// Update existing team atomically, or create a new one.
	if export.TeamID != "" {
		if existing, err := s.store.GetTeam(ctx, export.TeamID); err == nil {
			if err := existing.ReplaceComposition(export.TeamName, rootID, allMemberIDs, relations); err != nil {
				return "", fmt.Errorf("validate team composition: %w", err)
			}
			if err := s.store.ReplaceTeamComposition(ctx, &existing); err != nil {
				return "", fmt.Errorf("replace team: %w", err)
			}
			return existing.ID, nil
		}
	}

	// Create new team.
	team, err := domain.NewTeam(export.TeamName, rootID)
	if err != nil {
		return "", fmt.Errorf("new team: %w", err)
	}
	if export.TeamID != "" {
		team.ID = export.TeamID
	}
	// Add non-root members.
	for _, m := range export.Members {
		if m.Name == export.RootMemberName {
			continue
		}
		if err := team.AddMember(memberIDs[m.Name]); err != nil {
			return "", fmt.Errorf("add member %s: %w", m.Name, err)
		}
	}
	// Add relations.
	for _, r := range export.Relations {
		rel := domain.Relation{From: memberIDs[r.From], To: memberIDs[r.To], Type: r.Type}
		if err := team.AddRelation(rel); err != nil {
			return "", fmt.Errorf("add relation %s→%s: %w", r.From, r.To, err)
		}
	}
	if err := s.store.CreateTeam(ctx, team); err != nil {
		return "", fmt.Errorf("create team: %w", err)
	}
	return team.ID, nil
}

// importSystemPrompts deduplicates and creates/updates system prompts.
// Returns map[promptName]->promptID.
func (s *Service) importSystemPrompts(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		for _, p := range m.SystemPrompts {
			if _, exists := ids[p.Name]; exists {
				continue
			}
			if p.ID != "" {
				if existing, err := s.store.GetSystemPrompt(ctx, p.ID); err == nil {
					if err := existing.Update(&p.Name, &p.Prompt); err != nil {
						return nil, fmt.Errorf("update system prompt %s: %w", p.Name, err)
					}
					if err := s.store.UpdateSystemPrompt(ctx, &existing); err != nil {
						return nil, fmt.Errorf("update system prompt %s: %w", p.Name, err)
					}
					ids[p.Name] = p.ID
					continue
				}
			}
			sp, err := domain.NewSystemPrompt(p.Name, p.Prompt)
			if err != nil {
				return nil, fmt.Errorf("new system prompt %s: %w", p.Name, err)
			}
			if p.ID != "" {
				sp.ID = p.ID
			}
			if err := s.store.CreateSystemPrompt(ctx, sp); err != nil {
				return nil, fmt.Errorf("create system prompt %s: %w", p.Name, err)
			}
			ids[p.Name] = sp.ID
		}
	}
	return ids, nil
}

// importEnvs deduplicates and creates/updates envs.
// Returns map[envName]->envID.
func (s *Service) importEnvs(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		for _, e := range m.Envs {
			if _, exists := ids[e.Name]; exists {
				continue
			}
			if e.ID != "" {
				if existing, err := s.store.GetEnv(ctx, e.ID); err == nil {
					if err := existing.Update(&e.Name, &e.Key, &e.Value); err != nil {
						return nil, fmt.Errorf("update env %s: %w", e.Name, err)
					}
					if err := s.store.UpdateEnv(ctx, &existing); err != nil {
						return nil, fmt.Errorf("update env %s: %w", e.Name, err)
					}
					ids[e.Name] = e.ID
					continue
				}
			}
			env, err := domain.NewEnv(e.Name, e.Key, e.Value)
			if err != nil {
				return nil, fmt.Errorf("new env %s: %w", e.Name, err)
			}
			if e.ID != "" {
				env.ID = e.ID
			}
			if err := s.store.CreateEnv(ctx, env); err != nil {
				return nil, fmt.Errorf("create env %s: %w", e.Name, err)
			}
			ids[e.Name] = env.ID
		}
	}
	return ids, nil
}

// importGitRepos deduplicates and creates/updates git repos.
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
		if m.GitRepo.ID != "" {
			if existing, err := s.store.GetGitRepo(ctx, m.GitRepo.ID); err == nil {
				if err := existing.Update(&m.GitRepo.Name, &m.GitRepo.URL); err != nil {
					return nil, fmt.Errorf("update git repo %s: %w", m.GitRepo.Name, err)
				}
				if err := s.store.UpdateGitRepo(ctx, &existing); err != nil {
					return nil, fmt.Errorf("update git repo %s: %w", m.GitRepo.Name, err)
				}
				ids[m.GitRepo.Name] = m.GitRepo.ID
				continue
			}
		}
		repo, err := domain.NewGitRepo(m.GitRepo.Name, m.GitRepo.URL)
		if err != nil {
			return nil, fmt.Errorf("new git repo %s: %w", m.GitRepo.Name, err)
		}
		if m.GitRepo.ID != "" {
			repo.ID = m.GitRepo.ID
		}
		if err := s.store.CreateGitRepo(ctx, repo); err != nil {
			return nil, fmt.Errorf("create git repo %s: %w", m.GitRepo.Name, err)
		}
		ids[m.GitRepo.Name] = repo.ID
	}
	return ids, nil
}

// importCliProfiles deduplicates and creates/updates CLI profiles.
// Returns map[profileName]->profileID.
func (s *Service) importCliProfiles(ctx context.Context, export domain.TeamExport) (map[string]string, error) {
	ids := make(map[string]string)
	for _, m := range export.Members {
		p := m.CliProfile
		if _, exists := ids[p.Name]; exists {
			continue
		}
		if p.ID != "" {
			if existing, err := s.store.GetCliProfile(ctx, p.ID); err == nil {
				if err := existing.UpdateRaw(p.Name, p.Model, p.Binary, p.SystemArgs, p.CustomArgs, p.DotConfig); err != nil {
					return nil, fmt.Errorf("update cli profile %s: %w", p.Name, err)
				}
				if err := s.store.UpdateCliProfile(ctx, &existing); err != nil {
					return nil, fmt.Errorf("store update cli profile %s: %w", p.Name, err)
				}
				ids[p.Name] = p.ID
				continue
			}
		}
		profile, err := domain.NewCliProfileRaw(p.Name, p.Model, p.Binary, p.SystemArgs, p.CustomArgs, p.DotConfig)
		if err != nil {
			return nil, fmt.Errorf("new cli profile %s: %w", p.Name, err)
		}
		if p.ID != "" {
			profile.ID = p.ID
		}
		if err := s.store.CreateCliProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("create cli profile %s: %w", p.Name, err)
		}
		ids[p.Name] = profile.ID
	}
	return ids, nil
}

// importMembers creates/updates members, linking to already-created profiles/prompts/repos.
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

		if m.ID != "" {
			if existing, err := s.store.GetMember(ctx, m.ID); err == nil {
				if err := existing.Update(&m.Name, &profileID, &spIDs, &repoID, &eIDs); err != nil {
					return nil, fmt.Errorf("update member %s: %w", m.Name, err)
				}
				if err := s.store.UpdateMember(ctx, &existing); err != nil {
					return nil, fmt.Errorf("store update member %s: %w", m.Name, err)
				}
				ids[m.Name] = m.ID
				continue
			}
		}

		member, err := domain.NewMember(m.Name, profileID, spIDs, repoID, eIDs)
		if err != nil {
			return nil, fmt.Errorf("new member %s: %w", m.Name, err)
		}
		if m.ID != "" {
			member.ID = m.ID
		}
		if err := s.store.CreateMember(ctx, member); err != nil {
			return nil, fmt.Errorf("create member %s: %w", m.Name, err)
		}
		ids[m.Name] = member.ID
	}
	return ids, nil
}
