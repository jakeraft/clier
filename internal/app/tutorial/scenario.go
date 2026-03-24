package tutorial

import (
	"context"
	"fmt"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// Store defines the DB operations needed by the tutorial engine.
type Store interface {
	CreateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error
	CreateEnvironment(ctx context.Context, e *domain.Environment) error
	CreateGitRepo(ctx context.Context, r *domain.GitRepo) error
	CreateCliProfile(ctx context.Context, p *domain.CliProfile) error
	CreateMember(ctx context.Context, m *domain.Member) error
	CreateTeam(ctx context.Context, t *domain.Team) error
	AddTeamMember(ctx context.Context, teamID, memberID string) error
	AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error

	ListTeams(ctx context.Context) ([]domain.Team, error)
	ListMembers(ctx context.Context) ([]domain.Member, error)
	ListCliProfiles(ctx context.Context) ([]domain.CliProfile, error)
	ListSystemPrompts(ctx context.Context) ([]domain.SystemPrompt, error)
	ListEnvironments(ctx context.Context) ([]domain.Environment, error)
	ListGitRepos(ctx context.Context) ([]domain.GitRepo, error)

	DeleteTeam(ctx context.Context, id string) error
	DeleteMember(ctx context.Context, id string) error
	DeleteCliProfile(ctx context.Context, id string) error
	DeleteSystemPrompt(ctx context.Context, id string) error
	DeleteEnvironment(ctx context.Context, id string) error
	DeleteGitRepo(ctx context.Context, id string) error
}

type SystemPromptDef struct {
	Name   string
	Prompt string
}

type EnvironmentDef struct {
	Name  string
	Key   string
	Value string
}

type GitRepoDef struct {
	Name string
	URL  string
}

type CliProfileDef struct {
	Name      string
	PresetKey string
}

type MemberDef struct {
	Name              string
	CliProfileName    string
	SystemPromptNames []string
	EnvNames          []string
	GitRepoName       string // empty = none
}

type TeamDef struct {
	Name           string
	RootMemberName string
}

type RelationDef struct {
	From string
	To   string
	Type domain.RelationType
}

type Scenario struct {
	Name          string
	Description   string
	Prefix        string
	SystemPrompts []SystemPromptDef
	Environments  []EnvironmentDef
	GitRepos      []GitRepoDef
	CliProfiles   []CliProfileDef
	Members       []MemberDef
	Team          TeamDef
	Relations     []RelationDef
}

// Registry

var registry = map[string]*Scenario{}

func Register(s *Scenario) {
	registry[s.Name] = s
}

func Get(name string) (*Scenario, error) {
	s, ok := registry[name]
	if !ok {
		names := make([]string, 0, len(registry))
		for n := range registry {
			names = append(names, n)
		}
		return nil, fmt.Errorf("unknown scenario: %s (available: %s)", name, strings.Join(names, ", "))
	}
	return s, nil
}

func List() []*Scenario {
	scenarios := make([]*Scenario, 0, len(registry))
	for _, s := range registry {
		scenarios = append(scenarios, s)
	}
	return scenarios
}

// Run creates all resources for a scenario (clean first).
func Run(ctx context.Context, store Store, scenario *Scenario) error {
	if err := Clean(ctx, store, scenario); err != nil {
		return fmt.Errorf("clean: %w", err)
	}

	// Separate ID maps per entity type to prevent name collisions.
	systemPromptIDs := make(map[string]string)
	environmentIDs := make(map[string]string)
	gitRepoIDs := make(map[string]string)
	cliProfileIDs := make(map[string]string)
	memberIDs := make(map[string]string)

	// Phase 1: Independent resources (system prompts, environments, git repos, cli profiles)
	for _, def := range scenario.SystemPrompts {
		sp, err := domain.NewSystemPrompt(def.Name, def.Prompt)
		if err != nil {
			return fmt.Errorf("new system prompt %s: %w", def.Name, err)
		}
		if err := store.CreateSystemPrompt(ctx, sp); err != nil {
			return fmt.Errorf("create system prompt %s: %w", def.Name, err)
		}
		systemPromptIDs[def.Name] = sp.ID
	}

	for _, def := range scenario.Environments {
		env, err := domain.NewEnvironment(def.Name, def.Key, def.Value)
		if err != nil {
			return fmt.Errorf("new environment %s: %w", def.Name, err)
		}
		if err := store.CreateEnvironment(ctx, env); err != nil {
			return fmt.Errorf("create environment %s: %w", def.Name, err)
		}
		environmentIDs[def.Name] = env.ID
	}

	for _, def := range scenario.GitRepos {
		repo, err := domain.NewGitRepo(def.Name, def.URL)
		if err != nil {
			return fmt.Errorf("new git repo %s: %w", def.Name, err)
		}
		if err := store.CreateGitRepo(ctx, repo); err != nil {
			return fmt.Errorf("create git repo %s: %w", def.Name, err)
		}
		gitRepoIDs[def.Name] = repo.ID
	}

	for _, def := range scenario.CliProfiles {
		profile, err := domain.NewCliProfile(def.Name, def.PresetKey, nil)
		if err != nil {
			return fmt.Errorf("new cli profile %s: %w", def.Name, err)
		}
		if err := store.CreateCliProfile(ctx, profile); err != nil {
			return fmt.Errorf("create cli profile %s: %w", def.Name, err)
		}
		cliProfileIDs[def.Name] = profile.ID
	}

	// Phase 2: Members (depend on cli profile, system prompts, environments, git repos)
	for _, def := range scenario.Members {
		profileID, ok := cliProfileIDs[def.CliProfileName]
		if !ok {
			return fmt.Errorf("member %s: unknown cli profile %s", def.Name, def.CliProfileName)
		}

		spIDs := make([]string, len(def.SystemPromptNames))
		for i, name := range def.SystemPromptNames {
			id, ok := systemPromptIDs[name]
			if !ok {
				return fmt.Errorf("member %s: unknown system prompt %s", def.Name, name)
			}
			spIDs[i] = id
		}

		envIDs := make([]string, len(def.EnvNames))
		for i, name := range def.EnvNames {
			id, ok := environmentIDs[name]
			if !ok {
				return fmt.Errorf("member %s: unknown environment %s", def.Name, name)
			}
			envIDs[i] = id
		}

		var gitRepoID string
		if def.GitRepoName != "" {
			id, ok := gitRepoIDs[def.GitRepoName]
			if !ok {
				return fmt.Errorf("member %s: unknown git repo %s", def.Name, def.GitRepoName)
			}
			gitRepoID = id
		}

		member, err := domain.NewMember(def.Name, profileID, spIDs, envIDs, gitRepoID)
		if err != nil {
			return fmt.Errorf("new member %s: %w", def.Name, err)
		}
		if err := store.CreateMember(ctx, member); err != nil {
			return fmt.Errorf("create member %s: %w", def.Name, err)
		}
		memberIDs[def.Name] = member.ID
	}

	// Phase 3: Team (depends on root member)
	rootMemberID, ok := memberIDs[scenario.Team.RootMemberName]
	if !ok {
		return fmt.Errorf("team: unknown root member %s", scenario.Team.RootMemberName)
	}

	team, err := domain.NewTeam(scenario.Team.Name, rootMemberID)
	if err != nil {
		return fmt.Errorf("new team: %w", err)
	}
	if err := store.CreateTeam(ctx, team); err != nil {
		return fmt.Errorf("create team: %w", err)
	}

	// Phase 4: Add non-root members + relations (store-only, no in-memory tracking needed)
	for _, def := range scenario.Members {
		if def.Name == scenario.Team.RootMemberName {
			continue
		}
		if err := store.AddTeamMember(ctx, team.ID, memberIDs[def.Name]); err != nil {
			return fmt.Errorf("add member %s to team: %w", def.Name, err)
		}
	}

	for _, def := range scenario.Relations {
		fromID, ok := memberIDs[def.From]
		if !ok {
			return fmt.Errorf("relation: unknown from member %s", def.From)
		}
		toID, ok := memberIDs[def.To]
		if !ok {
			return fmt.Errorf("relation: unknown to member %s", def.To)
		}
		rel := domain.Relation{From: fromID, To: toID, Type: def.Type}
		if err := store.AddTeamRelation(ctx, team.ID, rel); err != nil {
			return fmt.Errorf("add relation %s → %s: %w", def.From, def.To, err)
		}
	}

	return nil
}

// Clean deletes all resources matching the scenario's prefix.
// Deletion order: teams → members → (cli profiles, system prompts, environments, git repos).
func Clean(ctx context.Context, store Store, scenario *Scenario) error {
	prefix := scenario.Prefix

	// 1. Delete teams (CASCADE: team_members, team_relations)
	teams, err := store.ListTeams(ctx)
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}
	for _, t := range teams {
		if strings.HasPrefix(t.Name, prefix) {
			if err := store.DeleteTeam(ctx, t.ID); err != nil {
				return fmt.Errorf("delete team %s: %w", t.Name, err)
			}
		}
	}

	// 2. Delete members (CASCADE: member_system_prompts, member_environments)
	members, err := store.ListMembers(ctx)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	for _, m := range members {
		if strings.HasPrefix(m.Name, prefix) {
			if err := store.DeleteMember(ctx, m.ID); err != nil {
				return fmt.Errorf("delete member %s: %w", m.Name, err)
			}
		}
	}

	// 3. Delete remaining resources (independent after members are gone)
	profiles, err := store.ListCliProfiles(ctx)
	if err != nil {
		return fmt.Errorf("list cli profiles: %w", err)
	}
	for _, p := range profiles {
		if strings.HasPrefix(p.Name, prefix) {
			if err := store.DeleteCliProfile(ctx, p.ID); err != nil {
				return fmt.Errorf("delete cli profile %s: %w", p.Name, err)
			}
		}
	}

	prompts, err := store.ListSystemPrompts(ctx)
	if err != nil {
		return fmt.Errorf("list system prompts: %w", err)
	}
	for _, sp := range prompts {
		if strings.HasPrefix(sp.Name, prefix) {
			if err := store.DeleteSystemPrompt(ctx, sp.ID); err != nil {
				return fmt.Errorf("delete system prompt %s: %w", sp.Name, err)
			}
		}
	}

	envs, err := store.ListEnvironments(ctx)
	if err != nil {
		return fmt.Errorf("list environments: %w", err)
	}
	for _, e := range envs {
		if strings.HasPrefix(e.Name, prefix) {
			if err := store.DeleteEnvironment(ctx, e.ID); err != nil {
				return fmt.Errorf("delete environment %s: %w", e.Name, err)
			}
		}
	}

	repos, err := store.ListGitRepos(ctx)
	if err != nil {
		return fmt.Errorf("list git repos: %w", err)
	}
	for _, r := range repos {
		if strings.HasPrefix(r.Name, prefix) {
			if err := store.DeleteGitRepo(ctx, r.ID); err != nil {
				return fmt.Errorf("delete git repo %s: %w", r.Name, err)
			}
		}
	}

	return nil
}
