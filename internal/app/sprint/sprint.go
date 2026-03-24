package sprint

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/adapter/terminal"
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
	SaveSurfaces(ctx context.Context, sprintID, workspaceRef string, surfaces map[string]string) error
	GetSurfaceRef(ctx context.Context, sprintID, memberID string) (string, error)
	GetWorkspaceRef(ctx context.Context, sprintID string) (string, error)
	DeleteSurfaces(ctx context.Context, sprintID string) error
}

// Terminal defines the terminal operations needed by the sprint engine.
type Terminal interface {
	Launch(workspaceName string, specs []terminal.SurfaceSpec) (*terminal.LaunchResult, error)
	Terminate(workspaceRef string) error
	Send(surfaceRef, text string) error
}

// Workspace defines the filesystem operations for sprint member environments.
type Workspace interface {
	PrepareMember(ctx context.Context, sprintID string, m domain.MemberSnapshot) (memberHome, workDir string, err error)
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

	specs, tempFiles, err := s.prepareMembers(ctx, sprint.ID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare members: %w", err)
	}

	if err := s.store.CreateSprint(ctx, sprint); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	result, err := s.terminal.Launch(sprint.Name, specs)
	if err != nil {
		s.failSprint(ctx, sprint.ID, err.Error())
		cleanupTempFiles(tempFiles)
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	surfaces := make(map[string]string, len(snapshot.Members))
	for i, m := range snapshot.Members {
		surfaces[m.MemberID] = result.Surfaces[i]
	}
	if err := s.store.SaveSurfaces(ctx, sprint.ID, result.WorkspaceRef, surfaces); err != nil {
		return nil, fmt.Errorf("save surfaces: %w", err)
	}

	return sprint, nil
}

func (s *Service) Stop(ctx context.Context, sprintID string) error {
	workspaceRef, err := s.store.GetWorkspaceRef(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get workspace ref: %w", err)
	}

	if err := s.terminal.Terminate(workspaceRef); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := s.store.UpdateSprintState(ctx, sprintID, domain.SprintCompleted, ""); err != nil {
		return fmt.Errorf("update sprint state: %w", err)
	}

	if err := s.store.DeleteSurfaces(ctx, sprintID); err != nil {
		return fmt.Errorf("delete surfaces: %w", err)
	}

	_ = s.workspace.Cleanup(sprintID)

	return nil
}

// prepareMembers prepares isolated workspaces and builds launch specs for all members.
func (s *Service) prepareMembers(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) ([]terminal.SurfaceSpec, []string, error) {
	var specs []terminal.SurfaceSpec
	var tempFiles []string

	for _, m := range snapshot.Members {
		memberHome, workDir, err := s.workspace.PrepareMember(ctx, sprintID, m)
		if err != nil {
			return nil, nil, fmt.Errorf("prepare member %s: %w", m.MemberName, err)
		}

		prompt := ComposePrompt(m.SystemPrompts, BuildProtocol(snapshot, m))
		env := BuildEnv(m, sprintID, memberHome)
		cmd, tf, err := BuildCommand(m, prompt, workDir, env)
		if err != nil {
			return nil, nil, fmt.Errorf("build command for %s: %w", m.MemberName, err)
		}
		tempFiles = append(tempFiles, tf...)

		specs = append(specs, terminal.SurfaceSpec{
			Name:    m.MemberName,
			Command: cmd,
		})
	}

	return specs, tempFiles, nil
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
		ms.Relations = team.GetMemberRelations(id)
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

