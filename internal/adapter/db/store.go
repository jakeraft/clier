package db

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// Member

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

// CliProfile

func (s *Store) GetCliProfile(ctx context.Context, id string) (domain.CliProfile, error) {
	row, err := s.queries.GetCliProfile(ctx, id)
	if err != nil {
		return domain.CliProfile{}, err
	}
	var systemArgs, customArgs []string
	_ = json.Unmarshal([]byte(row.SystemArgs), &systemArgs)
	_ = json.Unmarshal([]byte(row.CustomArgs), &customArgs)
	var dotConfig domain.DotConfig
	_ = json.Unmarshal([]byte(row.DotConfig), &dotConfig)
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

// SystemPrompt

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

// Environment

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

// GitRepo

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

// Sprint

func (s *Store) GetSprint(ctx context.Context, id string) (domain.Sprint, error) {
	row, err := s.queries.GetSprint(ctx, id)
	if err != nil {
		return domain.Sprint{}, err
	}
	var snapshot domain.TeamSnapshot
	_ = json.Unmarshal([]byte(row.TeamSnapshot), &snapshot)
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

func (s *Store) CreateSprint(ctx context.Context, sprint *domain.Sprint) error {
	snapshotJSON, err := json.Marshal(sprint.TeamSnapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	now := time.Now().Unix()
	return s.queries.CreateSprint(ctx, generated.CreateSprintParams{
		ID:           sprint.ID,
		Name:         sprint.Name,
		TeamSnapshot: string(snapshotJSON),
		State:        string(sprint.State),
		Error:        sprint.Error,
		CreatedAt:    now,
		UpdatedAt:    now,
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

// Message

func (s *Store) CreateMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error {
	return s.queries.CreateMessage(ctx, generated.CreateMessageParams{
		ID:           uuid.NewString(),
		SprintID:     sprintID,
		FromMemberID: fromMemberID,
		ToMemberID:   toMemberID,
		Content:      content,
		CreatedAt:    time.Now().Unix(),
	})
}
