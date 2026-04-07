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
	"github.com/jakeraft/clier/internal/domain/resource"
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
			TeamID: t.ID, FromTeamMemberID: r.From, ToTeamMemberID: r.To,
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
		relations = append(relations, domain.Relation{From: r.FromTeamMemberID, To: r.ToTeamMemberID})
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
		TeamID: teamID, FromTeamMemberID: r.From, ToTeamMemberID: r.To,
	})
	return err
}

func (s *Store) RemoveTeamRelation(ctx context.Context, teamID string, r domain.Relation) error {
	_, err := s.queries.RemoveTeamRelation(ctx, generated.RemoveTeamRelationParams{
		TeamID: teamID, FromTeamMemberID: r.From, ToTeamMemberID: r.To,
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
			TeamID: t.ID, FromTeamMemberID: r.From, ToTeamMemberID: r.To,
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

	argsJSON, err := json.Marshal(m.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}

	qtx := generated.New(tx)
	if _, err := qtx.CreateMember(ctx, generated.CreateMemberParams{
		ID:           m.ID,
		Name:         m.Name,
		Model:        m.Model,
		Args:         string(argsJSON),
		ClaudeMdID:   sql.NullString{String: m.ClaudeMdID, Valid: m.ClaudeMdID != ""},
		SettingsID:   sql.NullString{String: m.SettingsID, Valid: m.SettingsID != ""},
		ClaudeJsonID: sql.NullString{String: m.ClaudeJsonID, Valid: m.ClaudeJsonID != ""},
		GitRepoUrl:   m.GitRepoURL,
		CreatedAt:    m.CreatedAt.Unix(),
		UpdatedAt:    m.UpdatedAt.Unix(),
	}); err != nil {
		return err
	}
	for _, skillID := range m.SkillIDs {
		if _, err := qtx.AddMemberSkill(ctx, generated.AddMemberSkillParams{
			MemberID: m.ID, SkillID: skillID,
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
	var args []string
	if err := json.Unmarshal([]byte(row.Args), &args); err != nil {
		return domain.Member{}, fmt.Errorf("unmarshal args: %w", err)
	}
	if args == nil {
		args = []string{}
	}
	skillIDs, err := s.queries.ListMemberSkillIDs(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	if skillIDs == nil {
		skillIDs = []string{}
	}
	return domain.Member{
		ID:           row.ID,
		Name:         row.Name,
		Model:        row.Model,
		Args:         args,
		ClaudeMdID:   row.ClaudeMdID.String,
		SkillIDs:     skillIDs,
		SettingsID:   row.SettingsID.String,
		ClaudeJsonID: row.ClaudeJsonID.String,
		GitRepoURL:   row.GitRepoUrl,
		CreatedAt:    time.Unix(row.CreatedAt, 0),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0),
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

	argsJSON, err := json.Marshal(m.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}

	qtx := generated.New(tx)
	if _, err := qtx.UpdateMember(ctx, generated.UpdateMemberParams{
		Name:         m.Name,
		Model:        m.Model,
		Args:         string(argsJSON),
		ClaudeMdID:   sql.NullString{String: m.ClaudeMdID, Valid: m.ClaudeMdID != ""},
		SettingsID:   sql.NullString{String: m.SettingsID, Valid: m.SettingsID != ""},
		ClaudeJsonID: sql.NullString{String: m.ClaudeJsonID, Valid: m.ClaudeJsonID != ""},
		GitRepoUrl:   m.GitRepoURL,
		UpdatedAt:    m.UpdatedAt.Unix(),
		ID:           m.ID,
	}); err != nil {
		return err
	}
	// Replace skill junction rows
	if _, err := qtx.DeleteMemberSkills(ctx, m.ID); err != nil {
		return err
	}
	for _, skillID := range m.SkillIDs {
		if _, err := qtx.AddMemberSkill(ctx, generated.AddMemberSkillParams{
			MemberID: m.ID, SkillID: skillID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteMember deletes a member. CASCADE: member_skills.
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

// ClaudeMd

func (s *Store) CreateClaudeMd(ctx context.Context, c *resource.ClaudeMd) error {
	_, err := s.queries.CreateClaudeMd(ctx, generated.CreateClaudeMdParams{
		ID:        c.ID,
		Name:      c.Name,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Unix(),
		UpdatedAt: c.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetClaudeMd(ctx context.Context, id string) (resource.ClaudeMd, error) {
	row, err := s.queries.GetClaudeMd(ctx, id)
	if err != nil {
		return resource.ClaudeMd{}, err
	}
	return resource.ClaudeMd{
		ID:        row.ID,
		Name:      row.Name,
		Content:   row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListClaudeMds(ctx context.Context) ([]resource.ClaudeMd, error) {
	rows, err := s.queries.ListClaudeMds(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]resource.ClaudeMd, 0, len(rows))
	for _, row := range rows {
		items = append(items, resource.ClaudeMd{
			ID:        row.ID,
			Name:      row.Name,
			Content:   row.Content,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return items, nil
}

func (s *Store) UpdateClaudeMd(ctx context.Context, c *resource.ClaudeMd) error {
	_, err := s.queries.UpdateClaudeMd(ctx, generated.UpdateClaudeMdParams{
		Name:      c.Name,
		Content:   c.Content,
		UpdatedAt: c.UpdatedAt.Unix(),
		ID:        c.ID,
	})
	return err
}

// DeleteClaudeMd deletes a claude md. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteClaudeMd(ctx context.Context, id string) error {
	result, err := s.queries.DeleteClaudeMd(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("claude md not found: %s", id)
	}
	return nil
}

// Skill

func (s *Store) CreateSkill(ctx context.Context, sk *resource.Skill) error {
	_, err := s.queries.CreateSkill(ctx, generated.CreateSkillParams{
		ID:        sk.ID,
		Name:      sk.Name,
		Content:   sk.Content,
		CreatedAt: sk.CreatedAt.Unix(),
		UpdatedAt: sk.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetSkill(ctx context.Context, id string) (resource.Skill, error) {
	row, err := s.queries.GetSkill(ctx, id)
	if err != nil {
		return resource.Skill{}, err
	}
	return resource.Skill{
		ID:        row.ID,
		Name:      row.Name,
		Content:   row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListSkills(ctx context.Context) ([]resource.Skill, error) {
	rows, err := s.queries.ListSkills(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]resource.Skill, 0, len(rows))
	for _, row := range rows {
		items = append(items, resource.Skill{
			ID:        row.ID,
			Name:      row.Name,
			Content:   row.Content,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return items, nil
}

func (s *Store) UpdateSkill(ctx context.Context, sk *resource.Skill) error {
	_, err := s.queries.UpdateSkill(ctx, generated.UpdateSkillParams{
		Name:      sk.Name,
		Content:   sk.Content,
		UpdatedAt: sk.UpdatedAt.Unix(),
		ID:        sk.ID,
	})
	return err
}

// DeleteSkill deletes a skill. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	result, err := s.queries.DeleteSkill(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("skill not found: %s", id)
	}
	return nil
}

// Settings

func (s *Store) CreateSettings(ctx context.Context, st *resource.Settings) error {
	_, err := s.queries.CreateSettings(ctx, generated.CreateSettingsParams{
		ID:        st.ID,
		Name:      st.Name,
		Content:   st.Content,
		CreatedAt: st.CreatedAt.Unix(),
		UpdatedAt: st.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetSettings(ctx context.Context, id string) (resource.Settings, error) {
	row, err := s.queries.GetSettings(ctx, id)
	if err != nil {
		return resource.Settings{}, err
	}
	return resource.Settings{
		ID:        row.ID,
		Name:      row.Name,
		Content:   row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListSettings(ctx context.Context) ([]resource.Settings, error) {
	rows, err := s.queries.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]resource.Settings, 0, len(rows))
	for _, row := range rows {
		items = append(items, resource.Settings{
			ID:        row.ID,
			Name:      row.Name,
			Content:   row.Content,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return items, nil
}

func (s *Store) UpdateSettings(ctx context.Context, st *resource.Settings) error {
	_, err := s.queries.UpdateSettings(ctx, generated.UpdateSettingsParams{
		Name:      st.Name,
		Content:   st.Content,
		UpdatedAt: st.UpdatedAt.Unix(),
		ID:        st.ID,
	})
	return err
}

// DeleteSettings deletes a settings. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteSettings(ctx context.Context, id string) error {
	result, err := s.queries.DeleteSettings(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("settings not found: %s", id)
	}
	return nil
}

// ClaudeJson

func (s *Store) CreateClaudeJson(ctx context.Context, c *resource.ClaudeJson) error {
	_, err := s.queries.CreateClaudeJson(ctx, generated.CreateClaudeJsonParams{
		ID:        c.ID,
		Name:      c.Name,
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Unix(),
		UpdatedAt: c.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetClaudeJson(ctx context.Context, id string) (resource.ClaudeJson, error) {
	row, err := s.queries.GetClaudeJson(ctx, id)
	if err != nil {
		return resource.ClaudeJson{}, err
	}
	return resource.ClaudeJson{
		ID:        row.ID,
		Name:      row.Name,
		Content:   row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListClaudeJsons(ctx context.Context) ([]resource.ClaudeJson, error) {
	rows, err := s.queries.ListClaudeJsons(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]resource.ClaudeJson, 0, len(rows))
	for _, row := range rows {
		items = append(items, resource.ClaudeJson{
			ID:        row.ID,
			Name:      row.Name,
			Content:   row.Content,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return items, nil
}

func (s *Store) UpdateClaudeJson(ctx context.Context, c *resource.ClaudeJson) error {
	_, err := s.queries.UpdateClaudeJson(ctx, generated.UpdateClaudeJsonParams{
		Name:      c.Name,
		Content:   c.Content,
		UpdatedAt: c.UpdatedAt.Unix(),
		ID:        c.ID,
	})
	return err
}

// DeleteClaudeJson deletes a claude json. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteClaudeJson(ctx context.Context, id string) error {
	result, err := s.queries.DeleteClaudeJson(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("claude json not found: %s", id)
	}
	return nil
}

// Task

func unmarshalTask(row generated.Task) (domain.Task, error) {
	var plan []domain.MemberPlan
	if err := json.Unmarshal([]byte(row.Plan), &plan); err != nil {
		return domain.Task{}, fmt.Errorf("unmarshal task plan: %w", err)
	}
	if plan == nil {
		plan = []domain.MemberPlan{}
	}
	t := domain.Task{
		ID:        row.ID,
		Name:      row.Name,
		TeamID:    row.TeamID,
		Status:    domain.TaskStatus(row.Status),
		Plan:      plan,
		CreatedAt: time.Unix(row.CreatedAt, 0),
	}
	if row.StoppedAt.Valid {
		ts := time.Unix(row.StoppedAt.Int64, 0)
		t.StoppedAt = &ts
	}
	return t, nil
}

func (s *Store) CreateTask(ctx context.Context, task *domain.Task) error {
	planJSON, err := json.Marshal(task.Plan)
	if err != nil {
		return fmt.Errorf("marshal task plan: %w", err)
	}
	params := generated.CreateTaskParams{
		ID:        task.ID,
		Name:      task.Name,
		TeamID:    task.TeamID,
		Status:    string(task.Status),
		Plan:      string(planJSON),
		CreatedAt: task.CreatedAt.Unix(),
	}
	if task.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: task.StoppedAt.Unix(), Valid: true}
	}
	_, err = s.queries.CreateTask(ctx, params)
	return err
}

func (s *Store) GetTask(ctx context.Context, id string) (domain.Task, error) {
	row, err := s.queries.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	return unmarshalTask(row)
}

func (s *Store) ListTasks(ctx context.Context) ([]domain.Task, error) {
	rows, err := s.queries.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		t, err := unmarshalTask(row)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *Store) UpdateTaskStatus(ctx context.Context, task *domain.Task) error {
	params := generated.UpdateTaskStatusParams{
		Status: string(task.Status),
		ID:     task.ID,
	}
	if task.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: task.StoppedAt.Unix(), Valid: true}
	}
	_, err := s.queries.UpdateTaskStatus(ctx, params)
	return err
}

func (s *Store) DeleteTask(ctx context.Context, id string) error {
	result, err := s.queries.DeleteTask(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", id)
	}
	return nil
}

// Message

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	_, err := s.queries.CreateMessage(ctx, generated.CreateMessageParams{
		ID:               msg.ID,
		TaskID:           msg.TaskID,
		FromTeamMemberID: toNullString(msg.FromTeamMemberID),
		ToTeamMemberID:   msg.ToTeamMemberID,
		Content:          msg.Content,
		CreatedAt:        msg.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListMessagesByTaskID(ctx context.Context, taskID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			TaskID:           row.TaskID,
			FromTeamMemberID: row.FromTeamMemberID.String,
			ToTeamMemberID:   row.ToTeamMemberID,
			Content:          row.Content,
			CreatedAt:        time.Unix(row.CreatedAt, 0),
		})
	}
	return msgs, nil
}

func (s *Store) ListMessagesByTaskAndMember(ctx context.Context, taskID, teamMemberID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesByTaskAndMember(ctx, generated.ListMessagesByTaskAndMemberParams{
		TaskID: taskID, FromTeamMemberID: toNullString(teamMemberID), ToTeamMemberID: teamMemberID,
	})
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			TaskID:           row.TaskID,
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

// Note

func (s *Store) CreateNote(ctx context.Context, n *domain.Note) error {
	_, err := s.queries.CreateNote(ctx, generated.CreateNoteParams{
		ID:           n.ID,
		TaskID:       n.TaskID,
		TeamMemberID: n.TeamMemberID,
		Content:      n.Content,
		CreatedAt:    n.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListNotesByTaskID(ctx context.Context, taskID string) ([]domain.Note, error) {
	rows, err := s.queries.ListNotesByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	notes := make([]domain.Note, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, domain.Note{
			ID:           row.ID,
			TaskID:       row.TaskID,
			TeamMemberID: row.TeamMemberID,
			Content:      row.Content,
			CreatedAt:    time.Unix(row.CreatedAt, 0),
		})
	}
	return notes, nil
}

// TerminalRefs (infra state for terminal adapter)

func (s *Store) SaveRefs(ctx context.Context, taskID, memberID string, refs map[string]string) error {
	data, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("marshal refs: %w", err)
	}
	_, err = s.queries.SaveTerminalRefs(ctx, generated.SaveTerminalRefsParams{
		TaskID: taskID, TeamMemberID: memberID, Refs: string(data),
	})
	return err
}

func (s *Store) GetRefs(ctx context.Context, taskID, memberID string) (map[string]string, error) {
	raw, err := s.queries.GetTerminalRefs(ctx, generated.GetTerminalRefsParams{
		TaskID: taskID, TeamMemberID: memberID,
	})
	if err != nil {
		return nil, err
	}
	var refs map[string]string
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil, fmt.Errorf("unmarshal refs: %w", err)
	}
	return refs, nil
}

func (s *Store) GetTaskRefs(ctx context.Context, taskID string) (map[string]string, error) {
	raw, err := s.queries.GetTaskTerminalRefs(ctx, taskID)
	if err != nil {
		return nil, err
	}
	var refs map[string]string
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil, fmt.Errorf("unmarshal refs: %w", err)
	}
	return refs, nil
}

func (s *Store) DeleteRefs(ctx context.Context, taskID string) error {
	_, err := s.queries.DeleteTerminalRefs(ctx, taskID)
	return err
}
