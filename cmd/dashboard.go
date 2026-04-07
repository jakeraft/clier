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
	"github.com/jakeraft/clier/internal/domain/resource"
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
	agentDotMds, err := store.ListAgentDotMds(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	skills, err := store.ListSkills(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	claudeSettingsList, err := store.ListClaudeSettings(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	claudeJsons, err := store.ListClaudeJsons(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	repos, err := store.ListGitRepos(ctx)
	if err != nil {
		return dashboardData{}, err
	}
	tasks, err := store.ListTasks(ctx)
	if err != nil {
		return dashboardData{}, err
	}

	agentDotMdNames := nameMap(agentDotMds, func(c resource.AgentDotMd) (string, string) { return c.ID, c.Name })
	skillNames := nameMap(skills, func(s resource.Skill) (string, string) { return s.ID, s.Name })
	claudeSettingsNames := nameMap(claudeSettingsList, func(s resource.ClaudeSettings) (string, string) { return s.ID, s.Name })
	claudeJsonNames := nameMap(claudeJsons, func(c resource.ClaudeJson) (string, string) { return c.ID, c.Name })
	repoNames := nameMap(repos, func(r resource.GitRepo) (string, string) { return r.ID, r.Name })
	teamNames := nameMap(teams, func(t domain.Team) (string, string) { return t.ID, t.Name })

	taskViews, err := convertTasks(ctx, store, tasks, teamNames)
	if err != nil {
		return dashboardData{}, err
	}

	return dashboardData{
		Teams:          convertTeams(teams),
		Members:        convertMembers(members, agentDotMdNames, skillNames, claudeSettingsNames, claudeJsonNames, repoNames),
		AgentDotMds:    convertAgentDotMds(agentDotMds),
		Skills:         convertSkills(skills),
		ClaudeSettings: convertClaudeSettings(claudeSettingsList),
		ClaudeJsons:    convertClaudeJsons(claudeJsons),
		GitRepos:       convertGitRepos(repos),
		Tasks:          taskViews,
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

// --- domain -> view conversions ---

func convertTeams(teams []domain.Team) []teamView {
	views := make([]teamView, 0, len(teams))
	for _, t := range teams {
		names := make([]string, 0, len(t.TeamMembers))
		for _, tm := range t.TeamMembers {
			names = append(names, tm.Name)
		}
		relations := make([]relationView, 0, len(t.Relations))
		for _, r := range t.Relations {
			relations = append(relations, relationView{From: r.From, To: r.To})
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

func convertMembers(members []domain.Member, agentDotMdNames, skillNames, claudeSettingsNames, claudeJsonNames, repoNames map[string]string) []memberView {
	views := make([]memberView, 0, len(members))
	for _, m := range members {
		skNames := make([]string, 0, len(m.SkillIDs))
		for _, id := range m.SkillIDs {
			skNames = append(skNames, skillNames[id])
		}

		mv := memberView{
			ID:        m.ID,
			Name:      m.Name,
			AgentType: m.AgentType,
			Model:     m.Model,
			Args:      m.Args,
			SkillIDs:  m.SkillIDs,
			SkillNames: skNames,
			CreatedAt:  m.CreatedAt,
			UpdatedAt:  m.UpdatedAt,
		}

		if m.AgentDotMdID != "" {
			mv.AgentDotMdID = &m.AgentDotMdID
			name := agentDotMdNames[m.AgentDotMdID]
			mv.AgentDotMdName = &name
		}
		if m.ClaudeSettingsID != "" {
			mv.ClaudeSettingsID = &m.ClaudeSettingsID
			name := claudeSettingsNames[m.ClaudeSettingsID]
			mv.ClaudeSettingsName = &name
		}
		if m.ClaudeJsonID != "" {
			mv.ClaudeJsonID = &m.ClaudeJsonID
			name := claudeJsonNames[m.ClaudeJsonID]
			mv.ClaudeJsonName = &name
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

func convertAgentDotMds(items []resource.AgentDotMd) []agentDotMdView {
	views := make([]agentDotMdView, 0, len(items))
	for _, c := range items {
		views = append(views, agentDotMdView{
			ID:        c.ID,
			Name:      c.Name,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return views
}

func convertSkills(items []resource.Skill) []skillView {
	views := make([]skillView, 0, len(items))
	for _, s := range items {
		views = append(views, skillView{
			ID:        s.ID,
			Name:      s.Name,
			Content:   s.Content,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}
	return views
}

func convertClaudeSettings(items []resource.ClaudeSettings) []claudeSettingsView {
	views := make([]claudeSettingsView, 0, len(items))
	for _, s := range items {
		views = append(views, claudeSettingsView{
			ID:        s.ID,
			Name:      s.Name,
			Content:   s.Content,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		})
	}
	return views
}

func convertClaudeJsons(items []resource.ClaudeJson) []claudeJsonView {
	views := make([]claudeJsonView, 0, len(items))
	for _, c := range items {
		views = append(views, claudeJsonView{
			ID:        c.ID,
			Name:      c.Name,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return views
}

func convertGitRepos(repos []resource.GitRepo) []gitRepoView {
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

// --- view types (JSON serialization for the frontend) ---

type dashboardData struct {
	Teams          []teamView           `json:"teams"`
	Members        []memberView         `json:"members"`
	AgentDotMds    []agentDotMdView     `json:"agentDotMds"`
	Skills         []skillView          `json:"skills"`
	ClaudeSettings []claudeSettingsView `json:"claudeSettings"`
	ClaudeJsons    []claudeJsonView     `json:"claudeJsons"`
	GitRepos       []gitRepoView        `json:"gitRepos"`
	Tasks          []taskView           `json:"tasks"`
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
}

type memberView struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	AgentType           string    `json:"agentType"`
	Model               string    `json:"model"`
	Args                []string  `json:"args"`
	AgentDotMdID        *string   `json:"agentDotMdId"`
	SkillIDs            []string  `json:"skillIds"`
	ClaudeSettingsID    *string   `json:"claudeSettingsId"`
	ClaudeJsonID        *string   `json:"claudeJsonId"`
	GitRepoID           *string   `json:"gitRepoId"`
	AgentDotMdName      *string   `json:"agentDotMdName"`
	SkillNames          []string  `json:"skillNames"`
	ClaudeSettingsName  *string   `json:"claudeSettingsName"`
	ClaudeJsonName      *string   `json:"claudeJsonName"`
	GitRepoName         *string   `json:"gitRepoName"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type agentDotMdView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type skillView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type claudeSettingsView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type claudeJsonView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
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

type taskView struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	TeamID    string           `json:"teamId"`
	TeamName  string           `json:"teamName"`
	Status    string           `json:"status"`
	Plan      []memberPlanView `json:"plan"`
	Notes     []noteView       `json:"notes"`
	Messages  []messageView    `json:"messages"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type memberPlanView struct {
	TeamMemberID string                `json:"teamMemberId"`
	MemberName   string                `json:"memberName"`
	Memberspace  string                `json:"memberspace"`
	Command      string                `json:"command"`
	GitRepo      *memberPlanGitRepoRef `json:"gitRepo"`
	Files        []memberPlanFileEntry `json:"files"`
}

type memberPlanGitRepoRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type memberPlanFileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type noteView struct {
	ID           string    `json:"id"`
	TeamMemberID string    `json:"teamMemberId"`
	MemberName   string    `json:"memberName"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

type messageView struct {
	ID               string    `json:"id"`
	FromTeamMemberID string    `json:"fromTeamMemberId"`
	FromMemberName   string    `json:"fromMemberName"`
	ToTeamMemberID   string    `json:"toTeamMemberId"`
	ToMemberName     string    `json:"toMemberName"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"createdAt"`
}

func convertTasks(ctx context.Context, store *db.Store, tasks []domain.Task, teamNames map[string]string) ([]taskView, error) {
	views := make([]taskView, 0, len(tasks))
	for _, t := range tasks {
		notes, err := store.ListNotesByTaskID(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		msgs, err := store.ListMessagesByTaskID(ctx, t.ID)
		if err != nil {
			return nil, err
		}

		// Build teamMemberID -> memberName map from plan
		nameOf := make(map[string]string, len(t.Plan))
		for _, mp := range t.Plan {
			nameOf[mp.TeamMemberID] = mp.MemberName
		}

		planViews := make([]memberPlanView, 0, len(t.Plan))
		for _, mp := range t.Plan {
			var gitRepo *memberPlanGitRepoRef
			if mp.Workspace.GitRepo != nil {
				gitRepo = &memberPlanGitRepoRef{Name: mp.Workspace.GitRepo.Name, URL: mp.Workspace.GitRepo.URL}
			}
			files := make([]memberPlanFileEntry, 0, len(mp.Workspace.Files))
			for _, f := range mp.Workspace.Files {
				files = append(files, memberPlanFileEntry{Path: f.Path, Content: f.Content})
			}
			planViews = append(planViews, memberPlanView{
				TeamMemberID: mp.TeamMemberID,
				MemberName:   mp.MemberName,
				Memberspace:  mp.Workspace.Memberspace,
				Command:      mp.Terminal.Command,
				GitRepo:      gitRepo,
				Files:        files,
			})
		}

		noteViews := make([]noteView, 0, len(notes))
		for _, n := range notes {
			noteViews = append(noteViews, noteView{
				ID:           n.ID,
				TeamMemberID: n.TeamMemberID,
				MemberName:   nameOf[n.TeamMemberID],
				Content:      n.Content,
				CreatedAt:    n.CreatedAt,
			})
		}

		msgViews := make([]messageView, 0, len(msgs))
		for _, m := range msgs {
			msgViews = append(msgViews, messageView{
				ID:               m.ID,
				FromTeamMemberID: m.FromTeamMemberID,
				FromMemberName:   nameOf[m.FromTeamMemberID],
				ToTeamMemberID:   m.ToTeamMemberID,
				ToMemberName:     nameOf[m.ToTeamMemberID],
				Content:          m.Content,
				CreatedAt:        m.CreatedAt,
			})
		}

		updatedAt := t.CreatedAt
		if t.StoppedAt != nil {
			updatedAt = *t.StoppedAt
		}

		views = append(views, taskView{
			ID:        t.ID,
			Name:      t.Name,
			TeamID:    t.TeamID,
			TeamName:  teamNames[t.TeamID],
			Status:    string(t.Status),
			Plan:      planViews,
			Notes:     noteViews,
			Messages:  msgViews,
			CreatedAt: t.CreatedAt,
			UpdatedAt: updatedAt,
		})
	}
	return views, nil
}
