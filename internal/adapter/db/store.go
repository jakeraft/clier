package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

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
	if _, err := qtx.CreateTeam(ctx, generated.CreateTeamParams{
		ID:               t.ID,
		Name:             t.Name,
		RootTeamMemberID: t.RootTeamMemberID,
		CreatedAt:        t.CreatedAt.Unix(),
		UpdatedAt:        t.UpdatedAt.Unix(),
	}); err != nil {
		return err
	}
	for _, tm := range t.TeamMembers {
		if _, err := qtx.AddTeamMember(ctx, generated.AddTeamMemberParams{
			ID: tm.ID, TeamID: t.ID, MemberID: tm.MemberID, Name: tm.Name,
		}); err != nil {
			return err
		}
	}
	for _, r := range t.Relations {
		if _, err := qtx.AddTeamRelation(ctx, generated.AddTeamRelationParams{
			TeamID: t.ID, FromTeamMemberID: r.From, ToTeamMemberID: r.To, Type: string(r.Type),
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
	tmRows, err := s.queries.ListTeamMembers(ctx, id)
	if err != nil {
		return domain.Team{}, err
	}
	teamMembers := make([]domain.TeamMember, 0, len(tmRows))
	for _, r := range tmRows {
		teamMembers = append(teamMembers, domain.TeamMember{ID: r.ID, MemberID: r.MemberID, Name: r.Name})
	}
	relRows, err := s.queries.ListTeamRelations(ctx, id)
	if err != nil {
		return domain.Team{}, err
	}
	relations := make([]domain.Relation, 0, len(relRows))
	for _, r := range relRows {
		relations = append(relations, domain.Relation{From: r.FromTeamMemberID, To: r.ToTeamMemberID, Type: domain.RelationType(r.Type)})
	}
	return domain.Team{
		ID:               row.ID,
		Name:             row.Name,
		RootTeamMemberID: row.RootTeamMemberID,
		TeamMembers:      teamMembers,
		Relations:        relations,
		CreatedAt:        time.Unix(row.CreatedAt, 0),
		UpdatedAt:        time.Unix(row.UpdatedAt, 0),
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
	_, err := s.queries.UpdateTeam(ctx, generated.UpdateTeamParams{
		Name:             t.Name,
		RootTeamMemberID: t.RootTeamMemberID,
		UpdatedAt:        t.UpdatedAt.Unix(),
		ID:               t.ID,
	})
	return err
}

// DeleteTeam deletes a team. CASCADE: team_members, team_relations.
func (s *Store) DeleteTeam(ctx context.Context, id string) error {
	result, err := s.queries.DeleteTeam(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("team not found: %s", id)
	}
	return nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID string, tm domain.TeamMember) error {
	_, err := s.queries.AddTeamMember(ctx, generated.AddTeamMemberParams{
		ID: tm.ID, TeamID: teamID, MemberID: tm.MemberID, Name: tm.Name,
	})
	return err
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID, teamMemberID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if _, err := qtx.RemoveTeamMemberRelations(ctx, generated.RemoveTeamMemberRelationsParams{
		TeamID: teamID, FromTeamMemberID: teamMemberID, ToTeamMemberID: teamMemberID,
	}); err != nil {
		return err
	}
	if _, err := qtx.RemoveTeamMember(ctx, teamMemberID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AddTeamRelation(ctx context.Context, teamID string, r domain.Relation) error {
	_, err := s.queries.AddTeamRelation(ctx, generated.AddTeamRelationParams{
		TeamID: teamID, FromTeamMemberID: r.From, ToTeamMemberID: r.To, Type: string(r.Type),
	})
	return err
}

func (s *Store) RemoveTeamRelation(ctx context.Context, teamID string, r domain.Relation) error {
	_, err := s.queries.RemoveTeamRelation(ctx, generated.RemoveTeamRelationParams{
		TeamID: teamID, FromTeamMemberID: r.From, ToTeamMemberID: r.To, Type: string(r.Type),
	})
	return err
}

func (s *Store) DeleteTeamMembers(ctx context.Context, teamID string) error {
	_, err := s.queries.DeleteTeamMembers(ctx, teamID)
	return err
}

func (s *Store) DeleteTeamRelations(ctx context.Context, teamID string) error {
	_, err := s.queries.DeleteTeamRelations(ctx, teamID)
	return err
}

// ReplaceTeamComposition atomically updates a team's basic info,
// clears all members and relations, and re-adds them.
func (s *Store) ReplaceTeamComposition(ctx context.Context, t *domain.Team) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if _, err := qtx.DeleteTeamRelations(ctx, t.ID); err != nil {
		return err
	}
	if _, err := qtx.DeleteTeamMembers(ctx, t.ID); err != nil {
		return err
	}
	if _, err := qtx.UpdateTeam(ctx, generated.UpdateTeamParams{
		Name: t.Name, RootTeamMemberID: t.RootTeamMemberID, UpdatedAt: t.UpdatedAt.Unix(), ID: t.ID,
	}); err != nil {
		return err
	}
	for _, tm := range t.TeamMembers {
		if _, err := qtx.AddTeamMember(ctx, generated.AddTeamMemberParams{
			ID: tm.ID, TeamID: t.ID, MemberID: tm.MemberID, Name: tm.Name,
		}); err != nil {
			return err
		}
	}
	for _, r := range t.Relations {
		if _, err := qtx.AddTeamRelation(ctx, generated.AddTeamRelationParams{
			TeamID: t.ID, FromTeamMemberID: r.From, ToTeamMemberID: r.To, Type: string(r.Type),
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Member

func (s *Store) CreateMember(ctx context.Context, m *domain.Member) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := generated.New(tx)
	if _, err := qtx.CreateMember(ctx, generated.CreateMemberParams{
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
		if _, err := qtx.AddMemberSystemPrompt(ctx, generated.AddMemberSystemPromptParams{
			MemberID: m.ID, SystemPromptID: promptID,
		}); err != nil {
			return err
		}
	}
	for _, envID := range m.EnvIDs {
		if _, err := qtx.AddMemberEnv(ctx, generated.AddMemberEnvParams{
			MemberID: m.ID, EnvID: envID,
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
	if promptIDs == nil {
		promptIDs = []string{}
	}
	envIDs, err := s.queries.ListMemberEnvIDs(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	if envIDs == nil {
		envIDs = []string{}
	}
	return domain.Member{
		ID:              row.ID,
		Name:            row.Name,
		CliProfileID:    row.CliProfileID,
		SystemPromptIDs: promptIDs,
		EnvIDs:          envIDs,
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
	if _, err := qtx.UpdateMember(ctx, generated.UpdateMemberParams{
		Name:         m.Name,
		CliProfileID: m.CliProfileID,
		GitRepoID:    sql.NullString{String: m.GitRepoID, Valid: m.GitRepoID != ""},
		UpdatedAt:    m.UpdatedAt.Unix(),
		ID:           m.ID,
	}); err != nil {
		return err
	}
	// Replace junction rows: delete all + re-insert.
	if _, err := qtx.DeleteMemberSystemPrompts(ctx, m.ID); err != nil {
		return err
	}
	for _, promptID := range m.SystemPromptIDs {
		if _, err := qtx.AddMemberSystemPrompt(ctx, generated.AddMemberSystemPromptParams{
			MemberID: m.ID, SystemPromptID: promptID,
		}); err != nil {
			return err
		}
	}
	if _, err := qtx.DeleteMemberEnvs(ctx, m.ID); err != nil {
		return err
	}
	for _, envID := range m.EnvIDs {
		if _, err := qtx.AddMemberEnv(ctx, generated.AddMemberEnvParams{
			MemberID: m.ID, EnvID: envID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteMember deletes a member. CASCADE: member_system_prompts, member_envs.
// RESTRICT: team_members.member_id — fails if member is referenced by a team.
func (s *Store) DeleteMember(ctx context.Context, id string) error {
	result, err := s.queries.DeleteMember(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("member not found: %s", id)
	}
	return nil
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
	_, err = s.queries.CreateCliProfile(ctx, generated.CreateCliProfileParams{
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
	return err
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
	if systemArgs == nil {
		systemArgs = []string{}
	}
	if customArgs == nil {
		customArgs = []string{}
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
	_, err = s.queries.UpdateCliProfile(ctx, generated.UpdateCliProfileParams{
		Name:       p.Name,
		Model:      p.Model,
		Binary:     string(p.Binary),
		SystemArgs: systemArgs,
		CustomArgs: customArgs,
		DotConfig:  dotConfig,
		UpdatedAt:  p.UpdatedAt.Unix(),
		ID:         p.ID,
	})
	return err
}

// DeleteCliProfile deletes a cli profile. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteCliProfile(ctx context.Context, id string) error {
	result, err := s.queries.DeleteCliProfile(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("cli profile not found: %s", id)
	}
	return nil
}

// SystemPrompt

func (s *Store) CreateSystemPrompt(ctx context.Context, sp *domain.SystemPrompt) error {
	_, err := s.queries.CreateSystemPrompt(ctx, generated.CreateSystemPromptParams{
		ID:        sp.ID,
		Name:      sp.Name,
		Prompt:    sp.Prompt,
		CreatedAt: sp.CreatedAt.Unix(),
		UpdatedAt: sp.UpdatedAt.Unix(),
	})
	return err
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
	_, err := s.queries.UpdateSystemPrompt(ctx, generated.UpdateSystemPromptParams{
		Name:      sp.Name,
		Prompt:    sp.Prompt,
		UpdatedAt: sp.UpdatedAt.Unix(),
		ID:        sp.ID,
	})
	return err
}

// DeleteSystemPrompt deletes a system prompt. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteSystemPrompt(ctx context.Context, id string) error {
	result, err := s.queries.DeleteSystemPrompt(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("system prompt not found: %s", id)
	}
	return nil
}

// Env

func (s *Store) CreateEnv(ctx context.Context, e *domain.Env) error {
	_, err := s.queries.CreateEnv(ctx, generated.CreateEnvParams{
		ID:        e.ID,
		Name:      e.Name,
		Key:       e.Key,
		Value:     e.Value,
		CreatedAt: e.CreatedAt.Unix(),
		UpdatedAt: e.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetEnv(ctx context.Context, id string) (domain.Env, error) {
	row, err := s.queries.GetEnv(ctx, id)
	if err != nil {
		return domain.Env{}, err
	}
	return domain.Env{
		ID:        row.ID,
		Name:      row.Name,
		Key:       row.Key,
		Value:     row.Value,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListEnvs(ctx context.Context) ([]domain.Env, error) {
	rows, err := s.queries.ListEnvs(ctx)
	if err != nil {
		return nil, err
	}
	envs := make([]domain.Env, 0, len(rows))
	for _, row := range rows {
		envs = append(envs, domain.Env{
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

func (s *Store) UpdateEnv(ctx context.Context, e *domain.Env) error {
	_, err := s.queries.UpdateEnv(ctx, generated.UpdateEnvParams{
		Name:      e.Name,
		Key:       e.Key,
		Value:     e.Value,
		UpdatedAt: e.UpdatedAt.Unix(),
		ID:        e.ID,
	})
	return err
}

// DeleteEnv deletes an env. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteEnv(ctx context.Context, id string) error {
	result, err := s.queries.DeleteEnv(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("env not found: %s", id)
	}
	return nil
}

// GitRepo

func (s *Store) CreateGitRepo(ctx context.Context, r *domain.GitRepo) error {
	_, err := s.queries.CreateGitRepo(ctx, generated.CreateGitRepoParams{
		ID:        r.ID,
		Name:      r.Name,
		Url:       r.URL,
		CreatedAt: r.CreatedAt.Unix(),
		UpdatedAt: r.UpdatedAt.Unix(),
	})
	return err
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
	_, err := s.queries.UpdateGitRepo(ctx, generated.UpdateGitRepoParams{
		Name:      r.Name,
		Url:       r.URL,
		UpdatedAt: r.UpdatedAt.Unix(),
		ID:        r.ID,
	})
	return err
}

// DeleteGitRepo deletes a git repo. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteGitRepo(ctx context.Context, id string) error {
	result, err := s.queries.DeleteGitRepo(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("git repo not found: %s", id)
	}
	return nil
}

// Session

func unmarshalSession(row generated.Session) (domain.Session, error) {
	var plan []domain.MemberPlan
	if err := json.Unmarshal([]byte(row.Plan), &plan); err != nil {
		return domain.Session{}, fmt.Errorf("unmarshal session plan: %w", err)
	}
	if plan == nil {
		plan = []domain.MemberPlan{}
	}
	s := domain.Session{
		ID:        row.ID,
		TeamID:    row.TeamID,
		Status:    domain.SessionStatus(row.Status),
		Plan:      plan,
		CreatedAt: time.Unix(row.CreatedAt, 0),
	}
	if row.StoppedAt.Valid {
		t := time.Unix(row.StoppedAt.Int64, 0)
		s.StoppedAt = &t
	}
	return s, nil
}

func (s *Store) CreateSession(ctx context.Context, session *domain.Session) error {
	planJSON, err := json.Marshal(session.Plan)
	if err != nil {
		return fmt.Errorf("marshal session plan: %w", err)
	}
	params := generated.CreateSessionParams{
		ID:        session.ID,
		TeamID:    session.TeamID,
		Status:    string(session.Status),
		Plan:      string(planJSON),
		CreatedAt: session.CreatedAt.Unix(),
	}
	if session.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: session.StoppedAt.Unix(), Valid: true}
	}
	_, err = s.queries.CreateSession(ctx, params)
	return err
}

func (s *Store) GetSession(ctx context.Context, id string) (domain.Session, error) {
	row, err := s.queries.GetSession(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	return unmarshalSession(row)
}

func (s *Store) ListSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := s.queries.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	sessions := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		sess, err := unmarshalSession(row)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *Store) UpdateSessionStatus(ctx context.Context, session *domain.Session) error {
	params := generated.UpdateSessionStatusParams{
		Status: string(session.Status),
		ID:     session.ID,
	}
	if session.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: session.StoppedAt.Unix(), Valid: true}
	}
	_, err := s.queries.UpdateSessionStatus(ctx, params)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	result, err := s.queries.DeleteSession(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// Message

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	_, err := s.queries.CreateMessage(ctx, generated.CreateMessageParams{
		ID:               msg.ID,
		SessionID:        msg.SessionID,
		FromTeamMemberID: toNullString(msg.FromTeamMemberID),
		ToTeamMemberID:   msg.ToTeamMemberID,
		Content:          msg.Content,
		CreatedAt:        msg.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListMessagesBySessionID(ctx context.Context, sessionID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			SessionID:        row.SessionID,
			FromTeamMemberID: row.FromTeamMemberID.String,
			ToTeamMemberID:   row.ToTeamMemberID,
			Content:          row.Content,
			CreatedAt:        time.Unix(row.CreatedAt, 0),
		})
	}
	return msgs, nil
}

func (s *Store) ListMessagesBySessionAndMember(ctx context.Context, sessionID, teamMemberID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesBySessionAndMember(ctx, generated.ListMessagesBySessionAndMemberParams{
		SessionID: sessionID, FromTeamMemberID: toNullString(teamMemberID), ToTeamMemberID: teamMemberID,
	})
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			SessionID:        row.SessionID,
			FromTeamMemberID: row.FromTeamMemberID.String,
			ToTeamMemberID:   row.ToTeamMemberID,
			Content:          row.Content,
			CreatedAt:        time.Unix(row.CreatedAt, 0),
		})
	}
	return msgs, nil
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// Log

func (s *Store) CreateLog(ctx context.Context, l *domain.Log) error {
	_, err := s.queries.CreateLog(ctx, generated.CreateLogParams{
		ID:           l.ID,
		SessionID:    l.SessionID,
		TeamMemberID: l.TeamMemberID,
		Content:      l.Content,
		CreatedAt:    l.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListLogsBySessionID(ctx context.Context, sessionID string) ([]domain.Log, error) {
	rows, err := s.queries.ListLogsBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	logs := make([]domain.Log, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, domain.Log{
			ID:           row.ID,
			SessionID:    row.SessionID,
			TeamMemberID: row.TeamMemberID,
			Content:      row.Content,
			CreatedAt:    time.Unix(row.CreatedAt, 0),
		})
	}
	return logs, nil
}

// SessionSurface (infra state for terminal adapter)

func (s *Store) SaveSessionSurface(ctx context.Context, sessionID, teamMemberID, workspaceRef, surfaceRef string) error {
	_, err := s.queries.SaveSessionSurface(ctx, generated.SaveSessionSurfaceParams{
		SessionID: sessionID, TeamMemberID: teamMemberID, WorkspaceRef: workspaceRef, SurfaceRef: surfaceRef,
	})
	return err
}

func (s *Store) GetSessionSurface(ctx context.Context, sessionID, teamMemberID string) (workspaceRef, surfaceRef string, err error) {
	row, err := s.queries.GetSessionSurface(ctx, generated.GetSessionSurfaceParams{
		SessionID: sessionID, TeamMemberID: teamMemberID,
	})
	if err != nil {
		return "", "", err
	}
	return row.WorkspaceRef, row.SurfaceRef, nil
}

func (s *Store) GetSessionWorkspaceRef(ctx context.Context, sessionID string) (string, error) {
	return s.queries.GetSessionWorkspaceRef(ctx, sessionID)
}

func (s *Store) DeleteSessionSurfaces(ctx context.Context, sessionID string) error {
	_, err := s.queries.DeleteSessionSurfaces(ctx, sessionID)
	return err
}
