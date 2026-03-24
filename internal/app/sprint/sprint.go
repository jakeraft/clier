package sprint

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// Store defines the DB operations needed by the sprint engine.
type Store interface {
	GetTeamSnapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error)
	GetSprint(ctx context.Context, id string) (domain.Sprint, error)
	CreateSprint(ctx context.Context, sprint *domain.Sprint) error
	UpdateSprintState(ctx context.Context, sprintID string, state domain.SprintState, sprintErr string) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
}

// MemberSpec describes what to run for a member in a terminal session.
type MemberSpec struct {
	ID      string
	Name    string
	Command string
}

// Terminal defines the terminal operations needed by the sprint engine.
type Terminal interface {
	Launch(sprintID, sprintName string, members []MemberSpec) error
	Send(sprintID, memberID, text string) error
	Terminate(sprintID string) error
}

// MemberDir holds the prepared directory paths for a member.
type MemberDir struct {
	Home    string
	WorkDir string
}

// Workspace defines the filesystem operations for sprint environments.
type Workspace interface {
	Prepare(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) (map[string]MemberDir, error)
	Cleanup(sprintID string) error
}

// Service orchestrates sprint lifecycle.
type Service struct {
	store     Store
	terminal  Terminal
	workspace Workspace
}

func New(store Store, term Terminal, ws Workspace) *Service {
	return &Service{store: store, terminal: term, workspace: ws}
}

func (s *Service) Start(ctx context.Context, teamID string) (*domain.Sprint, error) {
	snapshot, err := s.store.GetTeamSnapshot(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team snapshot: %w", err)
	}

	sprint, err := domain.NewSprint(snapshot)
	if err != nil {
		return nil, fmt.Errorf("new sprint: %w", err)
	}

	members, err := s.prepareMembers(ctx, sprint.ID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare members: %w", err)
	}

	if err := s.store.CreateSprint(ctx, sprint); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	if err := s.terminal.Launch(sprint.ID, sprint.Name, members); err != nil {
		s.failSprint(ctx, sprint.ID, err.Error())
		_ = s.workspace.Cleanup(sprint.ID)
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	return sprint, nil
}

func (s *Service) Stop(ctx context.Context, sprintID string) error {
	if err := s.terminal.Terminate(sprintID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := s.store.UpdateSprintState(ctx, sprintID, domain.SprintCompleted, ""); err != nil {
		return fmt.Errorf("update sprint state: %w", err)
	}

	_ = s.workspace.Cleanup(sprintID)

	return nil
}

// prepareMembers sets up the workspace and builds launch commands for all members.
func (s *Service) prepareMembers(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) ([]MemberSpec, error) {
	dirs, err := s.workspace.Prepare(ctx, sprintID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	var members []MemberSpec

	for _, m := range snapshot.Members {
		dir := dirs[m.MemberID]
		prompt, err := BuildMemberPrompt(snapshot, m.MemberID)
		if err != nil {
			return nil, fmt.Errorf("build prompt for %s: %w", m.MemberName, err)
		}
		cmd, err := BuildCommand(m, prompt, dir.WorkDir, sprintID, dir.Home)
		if err != nil {
			return nil, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}

		members = append(members, MemberSpec{
			ID:      m.MemberID,
			Name:    m.MemberName,
			Command: cmd,
		})
	}

	return members, nil
}

func (s *Service) failSprint(ctx context.Context, sprintID, errMsg string) {
	_ = s.store.UpdateSprintState(ctx, sprintID, domain.SprintErrored, errMsg)
}
