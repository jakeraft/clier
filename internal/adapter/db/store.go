package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jakeraft/clier/internal/adapter/db/generated"
	"github.com/jakeraft/clier/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{
		db:      db,
		queries: generated.New(db),
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Team

func (s *Store) CreateTeam(ctx context.Context, t *domain.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if err := qtx.CreateTeam(ctx, generated.CreateTeamParams{
		ID:           t.ID,
		Name:         t.Name,
		RootMemberID: t.RootMemberID,
		CreatedAt:    t.CreatedAt.Unix(),
		UpdatedAt:    t.UpdatedAt.Unix(),
	}); err != nil {
		return err
	}
	for _, memberID := range t.MemberIDs {
		if err := qtx.AddTeamMember(ctx, generated.AddTeamMemberParams{
			TeamID: t.ID, MemberID: memberID,
		}); err != nil {
			return err
		}
	}
	for _, r := range t.Relations {
		if err := qtx.AddTeamRelation(ctx, generated.AddTeamRelationParams{
			TeamID: t.ID, FromMemberID: r.From, ToMemberID: r.To, Type: string(r.Type),
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetTeam(ctx context.Context, id string) (domain.Team, error) {
	row, err := s.queries.GetTeam(ctx, id)
	if err != nil {
		return domain.Team{}, err
	}
	memberIDs, err := s.queries.ListTeamMemberIDs(ctx, id)
	if err != nil {
		return domain.Team{}, err
	}
	relRows, err := s.queries.ListTeamRelations(ctx, id)
	if err != nil {
		return domain.Team{}, err
	}
	relations := make([]domain.Relation, len(relRows))
	for i, r := range relRows {
		relations[i] = domain.Relation{From: r.FromMemberID, To: r.ToMemberID, Type: domain.RelationType(r.Type)}
	}
	return domain.Team{
		ID:           row.ID,
		Name:         row.Name,
		RootMemberID: row.RootMemberID,
		MemberIDs:    memberIDs,
		Relations:    relations,
		CreatedAt:    time.Unix(row.CreatedAt, 0),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListTeams(ctx context.Context) ([]domain.Team, error) {
	rows, err := s.queries.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	teams := make([]domain.Team, 0, len(rows))
	for _, row := range rows {
		t, err := s.GetTeam(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (s *Store) UpdateTeam(ctx context.Context, t *domain.Team) error {
	return s.queries.UpdateTeam(ctx, generated.UpdateTeamParams{
		Name:         t.Name,
		RootMemberID: t.RootMemberID,
		UpdatedAt:    t.UpdatedAt.Unix(),
		ID:           t.ID,
	})
}

// DeleteTeam deletes a team. CASCADE: team_members, team_relations.
func (s *Store) DeleteTeam(ctx context.Context, id string) error {
	return s.queries.DeleteTeam(ctx, id)
}

func (s *Store) AddTeamMember(ctx context.Context, teamID, memberID string) error {
	return s.queries.AddTeamMember(ctx, generated.AddTeamMemberParams{
		TeamID: teamID, MemberID: memberID,
	})
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, memberID string) error {
	return s.queries.RemoveTeamMember(ctx, generated.RemoveTeamMemberParams{
		TeamID: teamID, MemberID: memberID,
	})
}

func (s *Store) AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error {
	return s.queries.AddTeamRelation(ctx, generated.AddTeamRelationParams{
		TeamID: teamID, FromMemberID: r.From, ToMemberID: r.To, Type: string(r.Type),
	})
}

func (s *Store) RemoveTeamRelation(ctx context.Context, teamID string, r domain.Relation) error {
	return s.queries.RemoveTeamRelation(ctx, generated.RemoveTeamRelationParams{
		TeamID: teamID, FromMemberID: r.From, ToMemberID: r.To, Type: string(r.Type),
	})
}

// Member

func (s *Store) CreateMember(ctx context.Context, m *domain.Member) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if err := qtx.CreateMember(ctx, generated.CreateMemberParams{
		ID:           m.ID,
		Name:         m.Name,
		CliProfileID: m.CliProfileID,
		GitRepoID:    sql.NullString{String: m.GitRepoID, Valid: m.GitRepoID != ""},
		CreatedAt:    m.CreatedAt.Unix(),
		UpdatedAt:    m.UpdatedAt.Unix(),
	}); err != nil {
		return err
	}
	for _, promptID := range m.SystemPromptIDs {
		if err := qtx.AddMemberSystemPrompt(ctx, generated.AddMemberSystemPromptParams{
			MemberID: m.ID, SystemPromptID: promptID,
		}); err != nil {
			return err
		}
	}
	for _, envID := range m.EnvironmentIDs {
		if err := qtx.AddMemberEnvironment(ctx, generated.AddMemberEnvironmentParams{
			MemberID: m.ID, EnvironmentID: envID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetMember(ctx context.Context, id string) (domain.Member, error) {
	row, err := s.queries.GetMember(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	promptIDs, err := s.queries.ListMemberSystemPromptIDs(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	envIDs, err := s.queries.ListMemberEnvironmentIDs(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	return domain.Member{
		ID:              row.ID,
		Name:            row.Name,
		CliProfileID:    row.CliProfileID,
		SystemPromptIDs: promptIDs,
		EnvironmentIDs:  envIDs,
		GitRepoID:       row.GitRepoID.String,
		CreatedAt:       time.Unix(row.CreatedAt, 0),
		UpdatedAt:       time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListMembers(ctx context.Context) ([]domain.Member, error) {
	rows, err := s.queries.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	members := make([]domain.Member, 0, len(rows))
	for _, row := range rows {
		m, err := s.GetMember(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (s *Store) UpdateMember(ctx context.Context, m *domain.Member) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if err := qtx.UpdateMember(ctx, generated.UpdateMemberParams{
		Name:         m.Name,
		CliProfileID: m.CliProfileID,
		GitRepoID:    sql.NullString{String: m.GitRepoID, Valid: m.GitRepoID != ""},
		UpdatedAt:    m.UpdatedAt.Unix(),
		ID:           m.ID,
	}); err != nil {
		return err
	}
	// Replace junction rows: delete all + re-insert.
	if err := qtx.DeleteMemberSystemPrompts(ctx, m.ID); err != nil {
		return err
	}
	for _, promptID := range m.SystemPromptIDs {
		if err := qtx.AddMemberSystemPrompt(ctx, generated.AddMemberSystemPromptParams{
			MemberID: m.ID, SystemPromptID: promptID,
		}); err != nil {
			return err
		}
	}
	if err := qtx.DeleteMemberEnvironments(ctx, m.ID); err != nil {
		return err
	}
	for _, envID := range m.EnvironmentIDs {
		if err := qtx.AddMemberEnvironment(ctx, generated.AddMemberEnvironmentParams{
			MemberID: m.ID, EnvironmentID: envID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteMember deletes a member. CASCADE: member_system_prompts, member_environments.
// RESTRICT: teams.root_member_id — fails if member is a team's root.
func (s *Store) DeleteMember(ctx context.Context, id string) error {
	return s.queries.DeleteMember(ctx, id)
}

// CliProfile

func marshalCliProfileJSON(p *domain.CliProfile) (systemArgs, customArgs, dotConfig string, err error) {
	sa, err := json.Marshal(p.SystemArgs)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal system_args: %w", err)
	}
	ca, err := json.Marshal(p.CustomArgs)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal custom_args: %w", err)
	}
	dc, err := json.Marshal(p.DotConfig)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal dot_config: %w", err)
	}
	return string(sa), string(ca), string(dc), nil
}

func (s *Store) CreateCliProfile(ctx context.Context, p *domain.CliProfile) error {
	systemArgs, customArgs, dotConfig, err := marshalCliProfileJSON(p)
	if err != nil {
		return err
	}
	return s.queries.CreateCliProfile(ctx, generated.CreateCliProfileParams{
		ID:         p.ID,
		Name:       p.Name,
		Model:      p.Model,
		Binary:     string(p.Binary),
		SystemArgs: systemArgs,
		CustomArgs: customArgs,
		DotConfig:  dotConfig,
		CreatedAt:  p.CreatedAt.Unix(),
		UpdatedAt:  p.UpdatedAt.Unix(),
	})
}

func unmarshalCliProfile(row generated.CliProfile) (domain.CliProfile, error) {
	var systemArgs, customArgs []string
	if err := json.Unmarshal([]byte(row.SystemArgs), &systemArgs); err != nil {
		return domain.CliProfile{}, fmt.Errorf("unmarshal system_args: %w", err)
	}
	if err := json.Unmarshal([]byte(row.CustomArgs), &customArgs); err != nil {
		return domain.CliProfile{}, fmt.Errorf("unmarshal custom_args: %w", err)
	}
	var dotConfig domain.DotConfig
	if err := json.Unmarshal([]byte(row.DotConfig), &dotConfig); err != nil {
		return domain.CliProfile{}, fmt.Errorf("unmarshal dot_config: %w", err)
	}
	return domain.CliProfile{
		ID:         row.ID,
		Name:       row.Name,
		Model:      row.Model,
		Binary:     domain.CliBinary(row.Binary),
		SystemArgs: systemArgs,
		CustomArgs: customArgs,
		DotConfig:  dotConfig,
		CreatedAt:  time.Unix(row.CreatedAt, 0),
		UpdatedAt:  time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) GetCliProfile(ctx context.Context, id string) (domain.CliProfile, error) {
	row, err := s.queries.GetCliProfile(ctx, id)
	if err != nil {
		return domain.CliProfile{}, err
	}
	return unmarshalCliProfile(row)
}

func (s *Store) ListCliProfiles(ctx context.Context) ([]domain.CliProfile, error) {
	rows, err := s.queries.ListCliProfiles(ctx)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.CliProfile, 0, len(rows))
	for _, row := range rows {
		p, err := unmarshalCliProfile(row)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (s *Store) UpdateCliProfile(ctx context.Context, p *domain.CliProfile) error {
	systemArgs, customArgs, dotConfig, err := marshalCliProfileJSON(p)
	if err != nil {
		return err
	}
	return s.queries.UpdateCliProfile(ctx, generated.UpdateCliProfileParams{
		Name:       p.Name,
		Model:      p.Model,
		Binary:     string(p.Binary),
		SystemArgs: systemArgs,
		CustomArgs: customArgs,
		DotConfig:  dotConfig,
		UpdatedAt:  p.UpdatedAt.Unix(),
		ID:         p.ID,
	})
}

// DeleteCliProfile deletes a cli profile. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteCliProfile(ctx context.Context, id string) error {
	return s.queries.DeleteCliProfile(ctx, id)
}

// SystemPrompt

func (s *Store) CreateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error {
	return s.queries.CreateSystemPrompt(ctx, generated.CreateSystemPromptParams{
		ID:        sp.ID,
		Name:      sp.Name,
		Prompt:    sp.Prompt,
		CreatedAt: sp.CreatedAt.Unix(),
		UpdatedAt: sp.UpdatedAt.Unix(),
	})
}

func (s *Store) GetSystemPrompt(ctx context.Context, id string) (domain.SystemPrompt, error) {
	row, err := s.queries.GetSystemPrompt(ctx, id)
	if err != nil {
		return domain.SystemPrompt{}, err
	}
	return domain.SystemPrompt{
		ID:        row.ID,
		Name:      row.Name,
		Prompt:    row.Prompt,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListSystemPrompts(ctx context.Context) ([]domain.SystemPrompt, error) {
	rows, err := s.queries.ListSystemPrompts(ctx)
	if err != nil {
		return nil, err
	}
	prompts := make([]domain.SystemPrompt, 0, len(rows))
	for _, row := range rows {
		prompts = append(prompts, domain.SystemPrompt{
			ID:        row.ID,
			Name:      row.Name,
			Prompt:    row.Prompt,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return prompts, nil
}

func (s *Store) UpdateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error {
	return s.queries.UpdateSystemPrompt(ctx, generated.UpdateSystemPromptParams{
		Name:      sp.Name,
		Prompt:    sp.Prompt,
		UpdatedAt: sp.UpdatedAt.Unix(),
		ID:        sp.ID,
	})
}

// DeleteSystemPrompt deletes a system prompt. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteSystemPrompt(ctx context.Context, id string) error {
	return s.queries.DeleteSystemPrompt(ctx, id)
}

// Environment

func (s *Store) CreateEnvironment(ctx context.Context, e *domain.Environment) error {
	return s.queries.CreateEnvironment(ctx, generated.CreateEnvironmentParams{
		ID:        e.ID,
		Name:      e.Name,
		Key:       e.Key,
		Value:     e.Value,
		CreatedAt: e.CreatedAt.Unix(),
		UpdatedAt: e.UpdatedAt.Unix(),
	})
}

func (s *Store) GetEnvironment(ctx context.Context, id string) (domain.Environment, error) {
	row, err := s.queries.GetEnvironment(ctx, id)
	if err != nil {
		return domain.Environment{}, err
	}
	return domain.Environment{
		ID:        row.ID,
		Name:      row.Name,
		Key:       row.Key,
		Value:     row.Value,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListEnvironments(ctx context.Context) ([]domain.Environment, error) {
	rows, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		return nil, err
	}
	envs := make([]domain.Environment, 0, len(rows))
	for _, row := range rows {
		envs = append(envs, domain.Environment{
			ID:        row.ID,
			Name:      row.Name,
			Key:       row.Key,
			Value:     row.Value,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return envs, nil
}

func (s *Store) UpdateEnvironment(ctx context.Context, e *domain.Environment) error {
	return s.queries.UpdateEnvironment(ctx, generated.UpdateEnvironmentParams{
		Name:      e.Name,
		Key:       e.Key,
		Value:     e.Value,
		UpdatedAt: e.UpdatedAt.Unix(),
		ID:        e.ID,
	})
}

// DeleteEnvironment deletes an environment. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteEnvironment(ctx context.Context, id string) error {
	return s.queries.DeleteEnvironment(ctx, id)
}

// GitRepo

func (s *Store) CreateGitRepo(ctx context.Context, r *domain.GitRepo) error {
	return s.queries.CreateGitRepo(ctx, generated.CreateGitRepoParams{
		ID:        r.ID,
		Name:      r.Name,
		Url:       r.URL,
		CreatedAt: r.CreatedAt.Unix(),
		UpdatedAt: r.UpdatedAt.Unix(),
	})
}

func (s *Store) GetGitRepo(ctx context.Context, id string) (domain.GitRepo, error) {
	row, err := s.queries.GetGitRepo(ctx, id)
	if err != nil {
		return domain.GitRepo{}, err
	}
	return domain.GitRepo{
		ID:        row.ID,
		Name:      row.Name,
		URL:       row.Url,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListGitRepos(ctx context.Context) ([]domain.GitRepo, error) {
	rows, err := s.queries.ListGitRepos(ctx)
	if err != nil {
		return nil, err
	}
	repos := make([]domain.GitRepo, 0, len(rows))
	for _, row := range rows {
		repos = append(repos, domain.GitRepo{
			ID:        row.ID,
			Name:      row.Name,
			URL:       row.Url,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return repos, nil
}

func (s *Store) UpdateGitRepo(ctx context.Context, r *domain.GitRepo) error {
	return s.queries.UpdateGitRepo(ctx, generated.UpdateGitRepoParams{
		Name:      r.Name,
		Url:       r.URL,
		UpdatedAt: r.UpdatedAt.Unix(),
		ID:        r.ID,
	})
}

// DeleteGitRepo deletes a git repo. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteGitRepo(ctx context.Context, id string) error {
	return s.queries.DeleteGitRepo(ctx, id)
}

// TeamSnapshot (aggregate)

func (s *Store) GetTeamSnapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error) {
	team, err := s.GetTeam(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("get team: %w", err)
	}

	members := make([]domain.MemberSnapshot, 0, len(team.MemberIDs))
	for _, id := range team.MemberIDs {
		ms, err := s.getMemberSnapshot(ctx, id)
		if err != nil {
			return domain.TeamSnapshot{}, fmt.Errorf("load member %s: %w", id, err)
		}
		ms.Relations = team.MemberRelations(id)
		members = append(members, ms)
	}

	return domain.TeamSnapshot{
		TeamName:     team.Name,
		RootMemberID: team.RootMemberID,
		Members:      members,
	}, nil
}

func (s *Store) getMemberSnapshot(ctx context.Context, memberID string) (domain.MemberSnapshot, error) {
	member, err := s.GetMember(ctx, memberID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get member: %w", err)
	}

	profile, err := s.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get cli profile: %w", err)
	}

	prompts := make([]domain.PromptSnapshot, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := s.GetSystemPrompt(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, domain.PromptSnapshot{Name: sp.Name, Prompt: sp.Prompt})
	}

	envs := make([]domain.EnvironmentSnapshot, 0, len(member.EnvironmentIDs))
	for _, id := range member.EnvironmentIDs {
		env, err := s.GetEnvironment(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get environment %s: %w", id, err)
		}
		envs = append(envs, domain.EnvironmentSnapshot{Name: env.Name, Key: env.Key, Value: env.Value})
	}

	var gitRepo *domain.GitRepoSnapshot
	if member.GitRepoID != "" {
		repo, err := s.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get git repo: %w", err)
		}
		gitRepo = &domain.GitRepoSnapshot{Name: repo.Name, URL: repo.URL}
	}

	return domain.MemberSnapshot{
		MemberID:       memberID,
		MemberName:     member.Name,
		Binary:         profile.Binary,
		Model:          profile.Model,
		CliProfileName: profile.Name,
		SystemArgs:     profile.SystemArgs,
		CustomArgs:     profile.CustomArgs,
		DotConfig:      profile.DotConfig,
		SystemPrompts:  prompts,
		Environments:   envs,
		GitRepo:        gitRepo,
	}, nil
}

// Sprint

func unmarshalSprint(row generated.Sprint) (domain.Sprint, error) {
	var snapshot domain.TeamSnapshot
	if err := json.Unmarshal([]byte(row.TeamSnapshot), &snapshot); err != nil {
		return domain.Sprint{}, fmt.Errorf("unmarshal team_snapshot: %w", err)
	}
	return domain.Sprint{
		ID:           row.ID,
		Name:         row.Name,
		TeamSnapshot: snapshot,
		State:        domain.SprintState(row.State),
		Error:        row.Error,
		CreatedAt:    time.Unix(row.CreatedAt, 0),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) GetSprint(ctx context.Context, id string) (domain.Sprint, error) {
	row, err := s.queries.GetSprint(ctx, id)
	if err != nil {
		return domain.Sprint{}, err
	}
	return unmarshalSprint(row)
}

func (s *Store) CreateSprint(ctx context.Context, sprint *domain.Sprint) error {
	snapshotJSON, err := json.Marshal(sprint.TeamSnapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return s.queries.CreateSprint(ctx, generated.CreateSprintParams{
		ID:           sprint.ID,
		Name:         sprint.Name,
		TeamSnapshot: string(snapshotJSON),
		State:        string(sprint.State),
		Error:        sprint.Error,
		CreatedAt:    sprint.CreatedAt.Unix(),
		UpdatedAt:    sprint.UpdatedAt.Unix(),
	})
}

func (s *Store) UpdateSprintState(ctx context.Context, sprintID string, state domain.SprintState, sprintErr string) error {
	return s.queries.UpdateSprintState(ctx, generated.UpdateSprintStateParams{
		State:     string(state),
		Error:     sprintErr,
		UpdatedAt: time.Now().Unix(),
		ID:        sprintID,
	})
}

func (s *Store) ListSprints(ctx context.Context) ([]domain.Sprint, error) {
	rows, err := s.queries.ListSprints(ctx)
	if err != nil {
		return nil, err
	}
	sprints := make([]domain.Sprint, 0, len(rows))
	for _, row := range rows {
		sp, err := unmarshalSprint(row)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, sp)
	}
	return sprints, nil
}

// DeleteSprint deletes a sprint. CASCADE: sprint_surfaces, messages.
func (s *Store) DeleteSprint(ctx context.Context, id string) error {
	return s.queries.DeleteSprint(ctx, id)
}

// Message

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	return s.queries.CreateMessage(ctx, generated.CreateMessageParams{
		ID:           msg.ID,
		SprintID:     msg.SprintID,
		FromMemberID: msg.FromMemberID,
		ToMemberID:   msg.ToMemberID,
		Content:      msg.Content,
		CreatedAt:    msg.CreatedAt.Unix(),
	})
}

func (s *Store) ListMessagesBySprintID(ctx context.Context, sprintID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesBySprintID(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:           row.ID,
			SprintID:     row.SprintID,
			FromMemberID: row.FromMemberID,
			ToMemberID:   row.ToMemberID,
			Content:      row.Content,
			CreatedAt:    time.Unix(row.CreatedAt, 0),
		})
	}
	return msgs, nil
}
