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

	qtx := generated.New(tx)
	if _, err := qtx.CreateMember(ctx, generated.CreateMemberParams{
		ID:               m.ID,
		Name:             m.Name,
		Command:          m.Command,
		ClaudeMdID:       sql.NullString{String: m.ClaudeMdID, Valid: m.ClaudeMdID != ""},
		ClaudeSettingsID: sql.NullString{String: m.ClaudeSettingsID, Valid: m.ClaudeSettingsID != ""},
		GitRepoUrl:       m.GitRepoURL,
		CreatedAt:        m.CreatedAt.Unix(),
		UpdatedAt:        m.UpdatedAt.Unix(),
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
	skillIDs, err := s.queries.ListMemberSkillIDs(ctx, id)
	if err != nil {
		return domain.Member{}, err
	}
	if skillIDs == nil {
		skillIDs = []string{}
	}
	return domain.Member{
		ID:               row.ID,
		Name:             row.Name,
		Command:          row.Command,
		ClaudeMdID:       row.ClaudeMdID.String,
		SkillIDs:         skillIDs,
		ClaudeSettingsID: row.ClaudeSettingsID.String,
		GitRepoURL:       row.GitRepoUrl,
		CreatedAt:        time.Unix(row.CreatedAt, 0),
		UpdatedAt:        time.Unix(row.UpdatedAt, 0),
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
		Name:             m.Name,
		Command:          m.Command,
		ClaudeMdID:       sql.NullString{String: m.ClaudeMdID, Valid: m.ClaudeMdID != ""},
		ClaudeSettingsID: sql.NullString{String: m.ClaudeSettingsID, Valid: m.ClaudeSettingsID != ""},
		GitRepoUrl:       m.GitRepoURL,
		UpdatedAt:        m.UpdatedAt.Unix(),
		ID:               m.ID,
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

// ClaudeSettings

func (s *Store) CreateClaudeSettings(ctx context.Context, st *resource.ClaudeSettings) error {
	_, err := s.queries.CreateClaudeSettings(ctx, generated.CreateClaudeSettingsParams{
		ID:        st.ID,
		Name:      st.Name,
		Content:   st.Content,
		CreatedAt: st.CreatedAt.Unix(),
		UpdatedAt: st.UpdatedAt.Unix(),
	})
	return err
}

func (s *Store) GetClaudeSettings(ctx context.Context, id string) (resource.ClaudeSettings, error) {
	row, err := s.queries.GetClaudeSettings(ctx, id)
	if err != nil {
		return resource.ClaudeSettings{}, err
	}
	return resource.ClaudeSettings{
		ID:        row.ID,
		Name:      row.Name,
		Content:   row.Content,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}, nil
}

func (s *Store) ListClaudeSettings(ctx context.Context) ([]resource.ClaudeSettings, error) {
	rows, err := s.queries.ListClaudeSettings(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]resource.ClaudeSettings, 0, len(rows))
	for _, row := range rows {
		items = append(items, resource.ClaudeSettings{
			ID:        row.ID,
			Name:      row.Name,
			Content:   row.Content,
			CreatedAt: time.Unix(row.CreatedAt, 0),
			UpdatedAt: time.Unix(row.UpdatedAt, 0),
		})
	}
	return items, nil
}

func (s *Store) UpdateClaudeSettings(ctx context.Context, st *resource.ClaudeSettings) error {
	_, err := s.queries.UpdateClaudeSettings(ctx, generated.UpdateClaudeSettingsParams{
		Name:      st.Name,
		Content:   st.Content,
		UpdatedAt: st.UpdatedAt.Unix(),
		ID:        st.ID,
	})
	return err
}

// DeleteClaudeSettings deletes a claude settings. RESTRICT: fails if referenced by a member.
func (s *Store) DeleteClaudeSettings(ctx context.Context, id string) error {
	result, err := s.queries.DeleteClaudeSettings(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("claude settings not found: %s", id)
	}
	return nil
}

// Run

func unmarshalRun(row generated.Run) (domain.Run, error) {
	var plan []domain.MemberPlan
	if err := json.Unmarshal([]byte(row.Plan), &plan); err != nil {
		return domain.Run{}, fmt.Errorf("unmarshal run plan: %w", err)
	}
	if plan == nil {
		plan = []domain.MemberPlan{}
	}
	r := domain.Run{
		ID:        row.ID,
		Name:      row.Name,
		TeamID:    row.TeamID,
		Status:    domain.RunStatus(row.Status),
		Plan:      plan,
		StartedAt: time.Unix(row.StartedAt, 0),
	}
	if row.StoppedAt.Valid {
		ts := time.Unix(row.StoppedAt.Int64, 0)
		r.StoppedAt = &ts
	}
	return r, nil
}

func (s *Store) CreateRun(ctx context.Context, run *domain.Run) error {
	planJSON, err := json.Marshal(run.Plan)
	if err != nil {
		return fmt.Errorf("marshal run plan: %w", err)
	}
	params := generated.CreateRunParams{
		ID:        run.ID,
		Name:      run.Name,
		TeamID:    run.TeamID,
		Status:    string(run.Status),
		Plan:      string(planJSON),
		StartedAt: run.StartedAt.Unix(),
	}
	if run.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: run.StoppedAt.Unix(), Valid: true}
	}
	_, err = s.queries.CreateRun(ctx, params)
	return err
}

func (s *Store) GetRun(ctx context.Context, id string) (domain.Run, error) {
	row, err := s.queries.GetRun(ctx, id)
	if err != nil {
		return domain.Run{}, err
	}
	return unmarshalRun(row)
}

func (s *Store) ListRuns(ctx context.Context) ([]domain.Run, error) {
	rows, err := s.queries.ListRuns(ctx)
	if err != nil {
		return nil, err
	}
	runs := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		r, err := unmarshalRun(row)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, nil
}

func (s *Store) UpdateRunStatus(ctx context.Context, run *domain.Run) error {
	params := generated.UpdateRunStatusParams{
		Status: string(run.Status),
		ID:     run.ID,
	}
	if run.StoppedAt != nil {
		params.StoppedAt = sql.NullInt64{Int64: run.StoppedAt.Unix(), Valid: true}
	}
	_, err := s.queries.UpdateRunStatus(ctx, params)
	return err
}

func (s *Store) DeleteRun(ctx context.Context, id string) error {
	result, err := s.queries.DeleteRun(ctx, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("run not found: %s", id)
	}
	return nil
}

// Message

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	_, err := s.queries.CreateMessage(ctx, generated.CreateMessageParams{
		ID:               msg.ID,
		RunID:            msg.RunID,
		FromTeamMemberID: toNullString(msg.FromTeamMemberID),
		ToTeamMemberID:   msg.ToTeamMemberID,
		Content:          msg.Content,
		CreatedAt:        msg.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListMessagesByRunID(ctx context.Context, runID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			RunID:            row.RunID,
			FromTeamMemberID: row.FromTeamMemberID.String,
			ToTeamMemberID:   row.ToTeamMemberID,
			Content:          row.Content,
			CreatedAt:        time.Unix(row.CreatedAt, 0),
		})
	}
	return msgs, nil
}

func (s *Store) ListMessagesByRunAndMember(ctx context.Context, runID, teamMemberID string) ([]domain.Message, error) {
	rows, err := s.queries.ListMessagesByRunAndMember(ctx, generated.ListMessagesByRunAndMemberParams{
		RunID: runID, FromTeamMemberID: toNullString(teamMemberID), ToTeamMemberID: teamMemberID,
	})
	if err != nil {
		return nil, err
	}
	msgs := make([]domain.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, domain.Message{
			ID:               row.ID,
			RunID:            row.RunID,
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
		RunID:        n.RunID,
		TeamMemberID: n.TeamMemberID,
		Content:      n.Content,
		CreatedAt:    n.CreatedAt.Unix(),
	})
	return err
}

func (s *Store) ListNotesByRunID(ctx context.Context, runID string) ([]domain.Note, error) {
	rows, err := s.queries.ListNotesByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	notes := make([]domain.Note, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, domain.Note{
			ID:           row.ID,
			RunID:        row.RunID,
			TeamMemberID: row.TeamMemberID,
			Content:      row.Content,
			CreatedAt:    time.Unix(row.CreatedAt, 0),
		})
	}
	return notes, nil
}

// TerminalRefs (infra state for terminal adapter)

func (s *Store) SaveRefs(ctx context.Context, runID, memberID string, refs map[string]string) error {
	data, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("marshal refs: %w", err)
	}
	_, err = s.queries.SaveTerminalRefs(ctx, generated.SaveTerminalRefsParams{
		RunID: runID, TeamMemberID: memberID, Refs: string(data),
	})
	return err
}

func (s *Store) GetRefs(ctx context.Context, runID, memberID string) (map[string]string, error) {
	raw, err := s.queries.GetTerminalRefs(ctx, generated.GetTerminalRefsParams{
		RunID: runID, TeamMemberID: memberID,
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

func (s *Store) GetRunRefs(ctx context.Context, runID string) (map[string]string, error) {
	raw, err := s.queries.GetRunTerminalRefs(ctx, runID)
	if err != nil {
		return nil, err
	}
	var refs map[string]string
	if err := json.Unmarshal([]byte(raw), &refs); err != nil {
		return nil, fmt.Errorf("unmarshal refs: %w", err)
	}
	return refs, nil
}

func (s *Store) DeleteRefs(ctx context.Context, runID string) error {
	_, err := s.queries.DeleteTerminalRefs(ctx, runID)
	return err
}
