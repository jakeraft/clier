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
	sprints, err := store.ListSprints(ctx)
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

	return DashboardData{
		Teams:         convertTeams(teams, memberNames),
		Members:       convertMembers(members, profileNames, promptNames, repoNames),
		Sprints:       convertSprints(sprints),
		CliProfiles:   convertCliProfiles(profiles),
		SystemPrompts: convertSystemPrompts(prompts),
		GitRepos:      convertGitRepos(repos),
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
		views = append(views, TeamView{
			ID:             t.ID,
			Name:           t.Name,
			RootMemberID:   t.RootMemberID,
			MemberIDs:      t.MemberIDs,
			Relations:      relations,
			RootMemberName: memberNames[t.RootMemberID],
			MemberNames:    names,
			CreatedAt:      t.CreatedAt,
			UpdatedAt:      t.UpdatedAt,
		})
	}
	return views
}

func convertMembers(members []domain.Member, profileNames, promptNames, repoNames map[string]string) []MemberView {
	views := make([]MemberView, 0, len(members))
	for _, m := range members {
		spNames := make([]string, 0, len(m.SystemPromptIDs))
		for _, id := range m.SystemPromptIDs {
			spNames = append(spNames, promptNames[id])
		}
		mv := MemberView{
			ID:                m.ID,
			Name:              m.Name,
			CliProfileID:      m.CliProfileID,
			SystemPromptIDs:   m.SystemPromptIDs,
			CliProfileName:    profileNames[m.CliProfileID],
			SystemPromptNames: spNames,
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

func convertSprints(sprints []domain.Sprint) []SprintView {
	views := make([]SprintView, 0, len(sprints))
	for _, s := range sprints {
		sv := SprintView{
			ID:        s.ID,
			Name:      s.Name,
			State:     string(s.State),
			TeamName:  s.TeamSnapshot.TeamName,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			TeamSnapshot: TeamSnapshotView{
				TeamName:     s.TeamSnapshot.TeamName,
				RootMemberID: s.TeamSnapshot.RootMemberID,
				Members:      convertMemberSnapshots(s.TeamSnapshot.Members),
			},
		}
		if s.Error != "" {
			sv.Error = &s.Error
		}
		views = append(views, sv)
	}
	return views
}

func convertMemberSnapshots(members []domain.MemberSnapshot) []MemberSnapshotView {
	views := make([]MemberSnapshotView, 0, len(members))
	for _, m := range members {
		prompts := make([]PromptSnapshotView, 0, len(m.SystemPrompts))
		for _, p := range m.SystemPrompts {
			prompts = append(prompts, PromptSnapshotView{Name: p.Name, Prompt: p.Prompt})
		}
		mv := MemberSnapshotView{
			MemberID:       m.MemberID,
			MemberName:     m.MemberName,
			Binary:         string(m.Binary),
			Model:          m.Model,
			CliProfileName: m.CliProfileName,
			SystemArgs:     m.SystemArgs,
			CustomArgs:     m.CustomArgs,
			DotConfig:      m.DotConfig,
			SystemPrompts:  prompts,
			Relations: RelationsView{
				Leaders: emptyIfNil(m.Relations.Leaders),
				Workers: emptyIfNil(m.Relations.Workers),
				Peers:   emptyIfNil(m.Relations.Peers),
			},
			Protocol: m.Protocol,
		}
		if m.GitRepo != nil {
			mv.GitRepo = &GitRepoSnapshotView{Name: m.GitRepo.Name, URL: m.GitRepo.URL}
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

func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
