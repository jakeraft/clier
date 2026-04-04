package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakeraft/clier/internal/adapter/db"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDashboardCmd())
}

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open a read-only dashboard snapshot in the browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newSettings()
			if err != nil {
				return err
			}
			store, err := newStore(cfg)
			if err != nil {
				return err
			}
			defer store.Close()

			outPath, err := generateDashboard(cmd.Context(), store, cfg.Paths.Dashboard())
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dashboard:", outPath)
			return exec.Command("open", outPath).Run() // macOS only
		},
	}
}

const jsonPlaceholder = "/* JSON_DATA */"

// generateDashboard collects all entities, injects them as JSON into index.html, and writes the result.
func generateDashboard(ctx context.Context, store *db.Store, outPath string) (string, error) {
	data, err := collectDashboardData(ctx, store)
	if err != nil {
		return "", fmt.Errorf("collect data: %w", err)
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal data: %w", err)
	}

	indexBytes, err := ui.DistFS.ReadFile(filepath.Join(ui.DistRoot, "index.html"))
	if err != nil {
		return "", fmt.Errorf("read embedded index.html: %w", err)
	}

	original := string(indexBytes)
	injected := strings.Replace(original, jsonPlaceholder, string(jsonBytes), 1)
	if injected == original {
		return "", fmt.Errorf("placeholder %q not found in index.html", jsonPlaceholder)
	}

	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	if err := os.WriteFile(outPath, []byte(injected), 0644); err != nil {
		return "", fmt.Errorf("write dashboard.html: %w", err)
	}

	return outPath, nil
}

func collectDashboardData(ctx context.Context, store *db.Store) (dashboardData, error) {
	teams, err := store.ListTeams(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	members, err := store.ListMembers(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	profiles, err := store.ListCliProfiles(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	prompts, err := store.ListSystemPrompts(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	repos, err := store.ListGitRepos(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	envs, err := store.ListEnvs(ctx)
	if err != nil {
		return dashboardData{}, err
	}

	profileNames := nameMap(profiles, func(p domain.CliProfile) (string, string) { return p.ID, p.Name })
	promptNames := nameMap(prompts, func(p domain.SystemPrompt) (string, string) { return p.ID, p.Name })
	repoNames := nameMap(repos, func(r domain.GitRepo) (string, string) { return r.ID, r.Name })
	envNames := nameMap(envs, func(e domain.Env) (string, string) { return e.ID, e.Name })

	return dashboardData{
		Teams:         convertTeams(teams),
		Members:       convertMembers(members, profileNames, promptNames, repoNames, envNames),
		CliProfiles:   convertCliProfiles(profiles),
		SystemPrompts: convertSystemPrompts(prompts),
		GitRepos:      convertGitRepos(repos),
		Envs:          convertEnvs(envs),
	}, nil
}

func nameMap[T any](items []T, fn func(T) (string, string)) map[string]string {
	m := make(map[string]string, len(items))
	for _, item := range items {
		k, v := fn(item)
		m[k] = v
	}
	return m
}

// --- domain → view conversions ---

func convertTeams(teams []domain.Team) []teamView {
	views := make([]teamView, 0, len(teams))
	for _, t := range teams {
		names := make([]string, 0, len(t.TeamMembers))
		for _, tm := range t.TeamMembers {
			names = append(names, tm.Name)
		}
		relations := make([]relationView, 0, len(t.Relations))
		for _, r := range t.Relations {
			relations = append(relations, relationView{From: r.From, To: r.To, Type: string(r.Type)})
		}
		teamMemberIDs := make([]string, 0, len(t.TeamMembers))
		teamMemberViews := make([]teamMemberView, 0, len(t.TeamMembers))
		for _, tm := range t.TeamMembers {
			teamMemberIDs = append(teamMemberIDs, tm.ID)
			teamMemberViews = append(teamMemberViews, teamMemberView{
				ID:       tm.ID,
				MemberID: tm.MemberID,
				Name:     tm.Name,
			})
		}

		rootMemberName := ""
		for _, tm := range t.TeamMembers {
			if tm.ID == t.RootTeamMemberID {
				rootMemberName = tm.Name
				break
			}
		}

		views = append(views, teamView{
			ID:               t.ID,
			Name:             t.Name,
			RootTeamMemberID: t.RootTeamMemberID,
			TeamMemberIDs:    teamMemberIDs,
			TeamMembers:      teamMemberViews,
			Relations:        relations,
			RootMemberName:   rootMemberName,
			MemberNames:      names,
			CreatedAt:        t.CreatedAt,
			UpdatedAt:        t.UpdatedAt,
		})
	}
	return views
}

func convertMembers(members []domain.Member, profileNames, promptNames, repoNames, envNames map[string]string) []memberView {
	views := make([]memberView, 0, len(members))
	for _, m := range members {
		spNames := make([]string, 0, len(m.SystemPromptIDs))
		for _, id := range m.SystemPromptIDs {
			spNames = append(spNames, promptNames[id])
		}
		eNames := make([]string, 0, len(m.EnvIDs))
		for _, id := range m.EnvIDs {
			eNames = append(eNames, envNames[id])
		}
		mv := memberView{
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

func convertCliProfiles(profiles []domain.CliProfile) []cliProfileView {
	views := make([]cliProfileView, 0, len(profiles))
	for _, p := range profiles {
		views = append(views, cliProfileView{
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

func convertSystemPrompts(prompts []domain.SystemPrompt) []systemPromptView {
	views := make([]systemPromptView, 0, len(prompts))
	for _, p := range prompts {
		views = append(views, systemPromptView{
			ID:        p.ID,
			Name:      p.Name,
			Prompt:    p.Prompt,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}
	return views
}

func convertGitRepos(repos []domain.GitRepo) []gitRepoView {
	views := make([]gitRepoView, 0, len(repos))
	for _, r := range repos {
		views = append(views, gitRepoView{
			ID:        r.ID,
			Name:      r.Name,
			URL:       r.URL,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return views
}

func convertEnvs(envs []domain.Env) []envView {
	views := make([]envView, 0, len(envs))
	for _, e := range envs {
		views = append(views, envView{
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

// --- view types (JSON serialization for the frontend) ---

type dashboardData struct {
	Teams         []teamView         `json:"teams"`
	Members       []memberView       `json:"members"`
	CliProfiles   []cliProfileView   `json:"cliProfiles"`
	SystemPrompts []systemPromptView `json:"systemPrompts"`
	GitRepos      []gitRepoView      `json:"gitRepos"`
	Envs          []envView          `json:"envs"`
}

type teamMemberView struct {
	ID       string `json:"id"`
	MemberID string `json:"memberId"`
	Name     string `json:"name"`
}

type teamView struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	RootTeamMemberID string           `json:"rootTeamMemberId"`
	TeamMemberIDs    []string         `json:"teamMemberIds"`
	TeamMembers      []teamMemberView `json:"teamMembers"`
	Relations        []relationView   `json:"relations"`
	RootMemberName   string           `json:"rootMemberName"`
	MemberNames      []string         `json:"memberNames"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type relationView struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type memberView struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	CliProfileID      string    `json:"cliProfileId"`
	SystemPromptIDs   []string  `json:"systemPromptIds"`
	EnvIDs            []string  `json:"envIds"`
	GitRepoID         *string   `json:"gitRepoId"`
	CliProfileName    string    `json:"cliProfileName"`
	SystemPromptNames []string  `json:"systemPromptNames"`
	EnvNames          []string  `json:"envNames"`
	GitRepoName       *string   `json:"gitRepoName"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type cliProfileView struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Model      string         `json:"model"`
	Binary     string         `json:"binary"`
	SystemArgs []string       `json:"systemArgs"`
	CustomArgs []string       `json:"customArgs"`
	DotConfig  map[string]any `json:"dotConfig"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type systemPromptView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prompt    string    `json:"prompt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type gitRepoView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type envView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
