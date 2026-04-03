package dashboard

import (
	"context"

	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/domain"
)

func Collect(ctx context.Context, store *db.Store) (DashboardData, error) {
	teams, err := store.ListTeams(ctx)
	if err != nil {
		return DashboardData{}, err
	}
	members, err := store.ListMembers(ctx)
	if err != nil {
		return DashboardData{}, err
	}
	profiles, err := store.ListCliProfiles(ctx)
	if err != nil {
		return DashboardData{}, err
	}
	prompts, err := store.ListSystemPrompts(ctx)
	if err != nil {
		return DashboardData{}, err
	}
	repos, err := store.ListGitRepos(ctx)
	if err != nil {
		return DashboardData{}, err
	}
	envs, err := store.ListEnvs(ctx)
	if err != nil {
		return DashboardData{}, err
	}

	memberNames := make(map[string]string, len(members))
	for _, m := range members {
		memberNames[m.ID] = m.Name
	}
	profileNames := make(map[string]string, len(profiles))
	for _, p := range profiles {
		profileNames[p.ID] = p.Name
	}
	promptNames := make(map[string]string, len(prompts))
	for _, p := range prompts {
		promptNames[p.ID] = p.Name
	}
	repoNames := make(map[string]string, len(repos))
	for _, r := range repos {
		repoNames[r.ID] = r.Name
	}
	envNames := make(map[string]string, len(envs))
	for _, e := range envs {
		envNames[e.ID] = e.Name
	}

	return DashboardData{
		Teams:         convertTeams(teams, memberNames),
		Members:       convertMembers(members, profileNames, promptNames, repoNames, envNames),
		CliProfiles:   convertCliProfiles(profiles),
		SystemPrompts: convertSystemPrompts(prompts),
		GitRepos:      convertGitRepos(repos),
		Envs:          convertEnvs(envs),
	}, nil
}

func convertTeams(teams []domain.Team, memberNames map[string]string) []TeamView {
	views := make([]TeamView, 0, len(teams))
	for _, t := range teams {
		names := make([]string, 0, len(t.MemberIDs))
		for _, id := range t.MemberIDs {
			names = append(names, memberNames[id])
		}
		relations := make([]RelationView, 0, len(t.Relations))
		for _, r := range t.Relations {
			relations = append(relations, RelationView{From: r.From, To: r.To, Type: string(r.Type)})
		}
		plan := make([]MemberSessionPlanView, 0, len(t.Plan))
		for _, m := range t.Plan {
			files := make([]FileEntryView, 0, len(m.Workspace.Files))
			for _, f := range m.Workspace.Files {
				files = append(files, FileEntryView{Path: f.Path, Content: f.Content})
			}
			mv := MemberSessionPlanView{
				MemberID:    m.MemberID,
				MemberName:  m.MemberName,
				Memberspace: m.Workspace.Memberspace,
				Command:     m.Terminal.Command,
				Files:       files,
			}
			if m.Workspace.GitRepo != nil {
				mv.GitRepo = &GitRepoRef{Name: m.Workspace.GitRepo.Name, URL: m.Workspace.GitRepo.URL}
			}
			plan = append(plan, mv)
		}

		views = append(views, TeamView{
			ID:             t.ID,
			Name:           t.Name,
			RootMemberID:   t.RootMemberID,
			MemberIDs:      t.MemberIDs,
			Relations:      relations,
			Plan:           plan,
			RootMemberName: memberNames[t.RootMemberID],
			MemberNames:    names,
			CreatedAt:      t.CreatedAt,
			UpdatedAt:      t.UpdatedAt,
		})
	}
	return views
}

func convertMembers(members []domain.Member, profileNames, promptNames, repoNames, envNames map[string]string) []MemberView {
	views := make([]MemberView, 0, len(members))
	for _, m := range members {
		spNames := make([]string, 0, len(m.SystemPromptIDs))
		for _, id := range m.SystemPromptIDs {
			spNames = append(spNames, promptNames[id])
		}
		eNames := make([]string, 0, len(m.EnvIDs))
		for _, id := range m.EnvIDs {
			eNames = append(eNames, envNames[id])
		}
		mv := MemberView{
			ID:                m.ID,
			Name:              m.Name,
			CliProfileID:      m.CliProfileID,
			SystemPromptIDs:   m.SystemPromptIDs,
			EnvIDs:            m.EnvIDs,
			CliProfileName:    profileNames[m.CliProfileID],
			SystemPromptNames: spNames,
			EnvNames:          eNames,
			CreatedAt:         m.CreatedAt,
			UpdatedAt:         m.UpdatedAt,
		}
		if m.GitRepoID != "" {
			mv.GitRepoID = &m.GitRepoID
			name := repoNames[m.GitRepoID]
			mv.GitRepoName = &name
		}
		views = append(views, mv)
	}
	return views
}

func convertCliProfiles(profiles []domain.CliProfile) []CliProfileView {
	views := make([]CliProfileView, 0, len(profiles))
	for _, p := range profiles {
		views = append(views, CliProfileView{
			ID:         p.ID,
			Name:       p.Name,
			Model:      p.Model,
			Binary:     string(p.Binary),
			SystemArgs: p.SystemArgs,
			CustomArgs: p.CustomArgs,
			DotConfig:  p.DotConfig,
			CreatedAt:  p.CreatedAt,
			UpdatedAt:  p.UpdatedAt,
		})
	}
	return views
}

func convertSystemPrompts(prompts []domain.SystemPrompt) []SystemPromptView {
	views := make([]SystemPromptView, 0, len(prompts))
	for _, p := range prompts {
		views = append(views, SystemPromptView{
			ID:        p.ID,
			Name:      p.Name,
			Prompt:    p.Prompt,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return views
}

func convertGitRepos(repos []domain.GitRepo) []GitRepoView {
	views := make([]GitRepoView, 0, len(repos))
	for _, r := range repos {
		views = append(views, GitRepoView{
			ID:        r.ID,
			Name:      r.Name,
			URL:       r.URL,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return views
}

func convertEnvs(envs []domain.Env) []EnvView {
	views := make([]EnvView, 0, len(envs))
	for _, e := range envs {
		views = append(views, EnvView{
			ID:        e.ID,
			Name:      e.Name,
			Key:       e.Key,
			Value:     e.Value,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}
	return views
}
