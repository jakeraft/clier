package sprint

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/settings"
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
	CreateMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error
}

// Terminal defines the terminal operations needed by the sprint engine.
type Terminal interface {
	Launch(workspaceName string, specs []terminal.SurfaceSpec) (*terminal.LaunchResult, error)
	Terminate(workspaceRef string) error
	Send(surfaceRef, text string) error
}

// Engine orchestrates sprint lifecycle.
type Engine struct {
	store    Store
	terminal Terminal
	settings *settings.Settings
}

func NewEngine(store Store, term Terminal, s *settings.Settings) *Engine {
	return &Engine{store: store, terminal: term, settings: s}
}

func (e *Engine) Start(ctx context.Context, teamID string) (*domain.Sprint, error) {
	snapshot, err := e.buildSnapshot(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("build snapshot: %w", err)
	}

	sprint := domain.NewSprint(snapshot)

	specs, tempFiles, err := e.prepareMembers(ctx, sprint.ID, snapshot)
	if err != nil {
		return nil, fmt.Errorf("prepare members: %w", err)
	}

	if err := e.store.CreateSprint(ctx, sprint); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	result, err := e.terminal.Launch(sprint.Name, specs)
	if err != nil {
		e.failSprint(ctx, sprint.ID, err.Error())
		cleanupTempFiles(tempFiles)
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	if err := saveSurfaces(e.settings.SprintsDir(), sprint.ID, snapshot, result); err != nil {
		return nil, fmt.Errorf("save surfaces: %w", err)
	}

	return sprint, nil
}

func (e *Engine) Stop(ctx context.Context, sprintID string) error {
	surfaces, err := loadSurfaces(e.settings.SprintsDir(), sprintID)
	if err != nil {
		return fmt.Errorf("load surfaces: %w", err)
	}

	if err := e.terminal.Terminate(surfaces.WorkspaceRef); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := e.store.UpdateSprintState(ctx, sprintID, domain.SprintCompleted, ""); err != nil {
		return fmt.Errorf("update sprint state: %w", err)
	}

	sprintDir := filepath.Join(e.settings.SprintsDir(), sprintID)
	_ = os.RemoveAll(sprintDir)

	return nil
}

// prepareMembers creates workspace directories, copies auth, writes configs,
// sets up git repos, and builds launch specs for all members.
func (e *Engine) prepareMembers(ctx context.Context, sprintID string, snapshot domain.TeamSnapshot) ([]terminal.SurfaceSpec, []string, error) {
	var specs []terminal.SurfaceSpec
	var tempFiles []string

	for _, m := range snapshot.Members {
		memberHome := filepath.Join(e.settings.SprintsDir(), sprintID, m.MemberID)
		workDir := filepath.Join(memberHome, "project")

		if err := os.MkdirAll(workDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("create workspace for %s: %w", m.MemberName, err)
		}

		if err := e.settings.CopyAuthTo(m.Binary, memberHome); err != nil {
			return nil, nil, fmt.Errorf("copy auth for %s: %w", m.MemberName, err)
		}

		if err := WriteConfigs(m, memberHome, workDir); err != nil {
			return nil, nil, fmt.Errorf("write configs for %s: %w", m.MemberName, err)
		}

		if err := e.setupGit(ctx, m, workDir); err != nil {
			return nil, nil, fmt.Errorf("setup git for %s: %w", m.MemberName, err)
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

func (e *Engine) setupGit(ctx context.Context, m domain.MemberSnapshot, workDir string) error {
	if m.GitRepo == nil {
		return exec.CommandContext(ctx, "git", "init", workDir).Run()
	}

	cloneURL := m.GitRepo.URL
	if host := extractHost(cloneURL); host != "" {
		if token, err := e.settings.GetCredential(host); err == nil {
			cloneURL = injectCredential(cloneURL, token)
		}
	}

	if err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", cloneURL, workDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", m.GitRepo.URL, err)
	}
	return nil
}

// buildSnapshot loads all team data from DB and creates a TeamSnapshot.
func (e *Engine) buildSnapshot(ctx context.Context, teamID string) (domain.TeamSnapshot, error) {
	team, err := e.store.GetTeam(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("get team: %w", err)
	}

	snapshots := make([]domain.MemberSnapshot, 0, len(team.MemberIDs))
	for _, id := range team.MemberIDs {
		ms, err := e.loadMemberSnapshot(ctx, id)
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

func (e *Engine) loadMemberSnapshot(ctx context.Context, memberID string) (domain.MemberSnapshot, error) {
	member, err := e.store.GetMember(ctx, memberID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get member: %w", err)
	}

	profile, err := e.store.GetCliProfile(ctx, member.CliProfileID)
	if err != nil {
		return domain.MemberSnapshot{}, fmt.Errorf("get cli profile: %w", err)
	}

	prompts := make([]domain.SnapshotPrompt, 0, len(member.SystemPromptIDs))
	for _, id := range member.SystemPromptIDs {
		sp, err := e.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, domain.SnapshotPrompt{Name: sp.Name, Prompt: sp.Prompt})
	}

	envs := make([]domain.SnapshotEnvironment, 0, len(member.EnvironmentIDs))
	for _, id := range member.EnvironmentIDs {
		env, err := e.store.GetEnvironment(ctx, id)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get environment %s: %w", id, err)
		}
		envs = append(envs, domain.SnapshotEnvironment{Name: env.Name, Key: env.Key, Value: env.Value})
	}

	var gitRepo *domain.SnapshotGitRepo
	if member.GitRepoID != "" {
		repo, err := e.store.GetGitRepo(ctx, member.GitRepoID)
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

func (e *Engine) failSprint(ctx context.Context, sprintID, errMsg string) {
	_ = e.store.UpdateSprintState(ctx, sprintID, domain.SprintErrored, errMsg)
}

// helpers

func extractHost(gitURL string) string {
	u, err := url.Parse(gitURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

func injectCredential(gitURL, token string) string {
	u, err := url.Parse(gitURL)
	if err != nil {
		return gitURL
	}
	u.User = url.UserPassword("x-access-token", token)
	return u.String()
}

func cleanupTempFiles(files []string) {
	for _, f := range files {
		_ = os.Remove(f)
	}
}
