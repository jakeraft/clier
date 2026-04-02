package sprint

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

// Store defines the DB operations needed by the sprint engine.
type Store interface {
	GetSprint(ctx context.Context, id string) (domain.Sprint, error)
	CreateSprint(ctx context.Context, sprint *domain.Sprint) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
}

// Terminal defines the terminal operations needed by the sprint engine.
type Terminal interface {
	Launch(sprintID, sprintName string, snapshot domain.SprintSnapshot) error
	Send(sprintID, memberID, text string) error
	Terminate(sprintID string) error
}

// Workspace defines the filesystem operations for sprint environments.
type Workspace interface {
	Prepare(ctx context.Context, sprintID string, snapshot domain.SprintSnapshot) error
	Cleanup(sprintID string) error
}

// TeamSnapshotter aggregates a team's complete state from normalised entities.
type TeamSnapshotter interface {
	Snapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error)
}

// Service orchestrates sprint lifecycle.
type Service struct {
	team      TeamSnapshotter
	store     Store
	terminal  Terminal
	workspace Workspace
	baseDir   string
}

func New(teamSvc TeamSnapshotter, store Store, term Terminal, ws Workspace, baseDir string) *Service {
	return &Service{team: teamSvc, store: store, terminal: term, workspace: ws, baseDir: baseDir}
}

func (s *Service) Whoami(ctx context.Context, sprintID, memberID string) (SprintPosition, error) {
	sp, err := s.store.GetSprint(ctx, sprintID)
	if err != nil {
		return SprintPosition{}, fmt.Errorf("get sprint: %w", err)
	}
	return BuildPosition(sp.Snapshot, sprintID, memberID)
}

func (s *Service) Start(ctx context.Context, teamID string) (*domain.Sprint, error) {
	teamSnap, err := s.team.Snapshot(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team snapshot: %w", err)
	}

	sprintID := uuid.NewString()

	snapshot, err := BuildSprintSnapshot(sprintID, s.baseDir, teamSnap)
	if err != nil {
		return nil, fmt.Errorf("build sprint snapshot: %w", err)
	}

	sp, err := domain.NewSprint(sprintID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("new sprint: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(sprintID)
		}
	}()

	if err := s.workspace.Prepare(ctx, sprintID, snapshot); err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	if err := s.terminal.Launch(sp.ID, sp.Name, snapshot); err != nil {
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	if err := s.store.CreateSprint(ctx, sp); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	success = true
	return sp, nil
}

func (s *Service) Stop(ctx context.Context, sprintID string) error {
	if _, err := s.store.GetSprint(ctx, sprintID); err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}

	if err := s.terminal.Terminate(sprintID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := s.workspace.Cleanup(sprintID); err != nil {
		return fmt.Errorf("cleanup workspace: %w", err)
	}

	return nil
}
