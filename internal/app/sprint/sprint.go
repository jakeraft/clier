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
	dataDir   string
}

func New(store Store, term Terminal, ws Workspace, dataDir string) *Service {
	return &Service{store: store, terminal: term, workspace: ws, dataDir: dataDir}
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

	dirs, err := s.workspace.Prepare(ctx, sprint.ID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(sprint.ID)
		}
	}()

	members, err := buildMemberSpecs(sprint.ID, snapshot, dirs, s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("build member specs: %w", err)
	}

	if err := s.store.CreateSprint(ctx, sprint); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	if err := s.terminal.Launch(sprint.ID, sprint.Name, members); err != nil {
		_ = sprint.Fail(err.Error())
		_ = s.store.UpdateSprintState(ctx, sprint.ID, sprint.State, sprint.Error)
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	success = true
	return sprint, nil
}

func (s *Service) Stop(ctx context.Context, sprintID string) error {
	sp, err := s.store.GetSprint(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}
	if err := sp.Complete(); err != nil {
		return err
	}

	if err := s.terminal.Terminate(sprintID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := s.store.UpdateSprintState(ctx, sprintID, sp.State, sp.Error); err != nil {
		return fmt.Errorf("update sprint state: %w", err)
	}

	// Cleanup may leave empty .cache/starship dirs behind.
	// Cause: cmux surfaces start zsh with the user's .zshrc (starship init)
	// before HOME is overridden via sendAndEnter. The dying starship process
	// can recreate $HOME/.cache/starship after RemoveAll. These are 0-byte
	// empty dirs with no functional impact.
	if err := s.workspace.Cleanup(sprintID); err != nil {
		return fmt.Errorf("cleanup workspace: %w", err)
	}

	return nil
}

func buildMemberSpecs(sprintID string, snapshot domain.TeamSnapshot, dirs map[string]MemberDir, dataDir string) ([]MemberSpec, error) {
	var members []MemberSpec

	for _, m := range snapshot.Members {
		dir := dirs[m.MemberID]
		prompt, err := BuildMemberPrompt(snapshot, m.MemberID)
		if err != nil {
			return nil, fmt.Errorf("build prompt for %s: %w", m.MemberName, err)
		}
		cmd, err := BuildCommand(m, prompt, dir.WorkDir, sprintID, dir.Home, dataDir)
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

