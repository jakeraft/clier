package sprint

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// Store defines the DB operations needed by the sprint engine.
type Store interface {
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetCliProfile(ctx context.Context, id string) (domain.CliProfile, error)
	GetSystemPrompt(ctx context.Context, id string) (domain.SystemPrompt, error)
	GetEnvironment(ctx context.Context, id string) (domain.Environment, error)
	GetGitRepo(ctx context.Context, id string) (domain.GitRepo, error)
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
	snapshot, err := s.buildSnapshot(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("build snapshot: %w", err)
	}

	sprint, err := domain.NewSprint(snapshot)
	if err != nil {
		return nil, fmt.Errorf("new sprint: %w", err)
	}

	members, tempFiles, err := s.prepareMembers(ctx, sprint.ID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare members: %w", err)
	}

	if err := s.store.CreateSprint(ctx, sprint); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	if err := s.terminal.Launch(sprint.ID, sprint.Name, members); err != nil {
		s.failSprint(ctx, sprint.ID, err.Error())
		cleanupTempFiles(tempFiles)
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
func (s *Service) prepareMembers(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) ([]MemberSpec, []string, error) {
	dirs, err := s.workspace.Prepare(ctx, sprintID, snapshot)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare workspace: %w", err)
	}

	var members []MemberSpec
	var tempFiles []string

	for _, m := range snapshot.Members {
		dir := dirs[m.MemberID]
		prompt, err := BuildMemberPrompt(snapshot, m.MemberID)
		if err != nil {
			return nil, nil, fmt.Errorf("build prompt for %s: %w", m.MemberName, err)
		}
		env := BuildEnv(m, sprintID, dir.Home)
		cmd, tf, err := BuildCommand(m, prompt, dir.WorkDir, env)
		if err != nil {
			return nil, nil, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}
		tempFiles = append(tempFiles, tf...)

		members = append(members, MemberSpec{
			ID:      m.MemberID,
			Name:    m.MemberName,
			Command: cmd,
		})
	}

	return members, tempFiles, nil
}

// buildSnapshot loads all team data from DB and creates a TeamSnapshot.
func (s *Service) buildSnapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error) {
	team, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("get team: %w", err)
	}

	snapshots := make([]domain.MemberSnapshot, 0, len(team.MemberIDs))
	for _, id := range team.MemberIDs {
		ms, err := s.loadMemberSnapshot(ctx, id)
		if err != nil {
			return domain.TeamSnapshot{}, fmt.Errorf("load member %s: %w", id, err)
		}
		ms.Relations = team.MemberRelations(id)
		snapshots = append(snapshots, ms)
	}

	return domain.TeamSnapshot{
		TeamName:     team.Name,
		RootMemberID: team.RootMemberID,
		Members:      snapshots,
	}, nil
}

func (s *Service) loadMemberSnapshot(ctx context.Context, memberID string) (domain.MemberSnapshot, error) {
	member, err := s.store.GetMember(ctx, memberID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get member: %w", err)
	}

	profile, err := s.store.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get cli profile: %w", err)
	}

	prompts := make([]domain.SnapshotPrompt, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := s.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, domain.SnapshotPrompt{Name: sp.Name, Prompt: sp.Prompt})
	}

	envs := make([]domain.SnapshotEnvironment, 0, len(member.EnvironmentIDs))
	for _, id := range member.EnvironmentIDs {
		env, err := s.store.GetEnvironment(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get environment %s: %w", id, err)
		}
		envs = append(envs, domain.SnapshotEnvironment{Name: env.Name, Key: env.Key, Value: env.Value})
	}

	var gitRepo *domain.SnapshotGitRepo
	if member.GitRepoID != "" {
		repo, err := s.store.GetGitRepo(ctx, member.GitRepoID)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get git repo: %w", err)
		}
		gitRepo = &domain.SnapshotGitRepo{Name: repo.Name, URL: repo.URL}
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

func (s *Service) failSprint(ctx context.Context, sprintID, errMsg string) {
	_ = s.store.UpdateSprintState(ctx, sprintID, domain.SprintErrored, errMsg)
}
