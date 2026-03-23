package sprint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jakeraft/clier/internal/db/generated"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/settings"
	"github.com/jakeraft/clier/internal/terminal"
)

// Store defines the DB operations needed by the sprint engine.
type Store interface {
	GetTeam(ctx context.Context, id string) (generated.Team, error)
	ListTeamMemberIDs(ctx context.Context, teamID string) ([]string, error)
	ListTeamRelations(ctx context.Context, teamID string) ([]generated.ListTeamRelationsRow, error)
	GetMember(ctx context.Context, id string) (generated.Member, error)
	GetCliProfile(ctx context.Context, id string) (generated.CliProfile, error)
	ListMemberSystemPromptIDs(ctx context.Context, memberID string) ([]string, error)
	GetSystemPrompt(ctx context.Context, id string) (generated.SystemPrompt, error)
	ListMemberEnvironmentIDs(ctx context.Context, memberID string) ([]string, error)
	GetEnvironment(ctx context.Context, id string) (generated.Environment, error)
	GetGitRepo(ctx context.Context, id string) (generated.GitRepo, error)
	CreateSprint(ctx context.Context, arg generated.CreateSprintParams) error
	GetSprint(ctx context.Context, id string) (generated.Sprint, error)
	UpdateSprintState(ctx context.Context, arg generated.UpdateSprintStateParams) error
	CreateMessage(ctx context.Context, arg generated.CreateMessageParams) error
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

	specs, tempFiles, err := e.prepareMembers(ctx, sprint.ID, snapshot.Members)
	if err != nil {
		return nil, fmt.Errorf("prepare members: %w", err)
	}

	if err := e.saveSprint(ctx, sprint, snapshot); err != nil {
		return nil, fmt.Errorf("save sprint: %w", err)
	}

	result, err := e.terminal.Launch(sprint.Name, specs)
	if err != nil {
		e.failSprint(ctx, sprint.ID, err.Error())
		cleanupTempFiles(tempFiles)
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	if err := saveSurfaces(e.settings.SprintsDir(), sprint.ID, snapshot.Members, result); err != nil {
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

	if err := e.store.UpdateSprintState(ctx, generated.UpdateSprintStateParams{
		State:     string(domain.SprintCompleted),
		Error:     "",
		UpdatedAt: time.Now().Unix(),
		ID:        sprintID,
	}); err != nil {
		return fmt.Errorf("update sprint state: %w", err)
	}

	sprintDir := filepath.Join(e.settings.SprintsDir(), sprintID)
	_ = os.RemoveAll(sprintDir)

	return nil
}

// prepareMembers creates workspace directories, copies auth, writes configs,
// sets up git repos, and builds launch specs for all members.
func (e *Engine) prepareMembers(ctx context.Context, sprintID string, members []domain.MemberSnapshot) ([]terminal.SurfaceSpec, []string, error) {
	var specs []terminal.SurfaceSpec
	var tempFiles []string

	for _, m := range members {
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

		env := BuildEnv(m, sprintID, memberHome)
		cmd, tf, err := BuildCommand(m, workDir, env)
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

	memberIDs, err := e.store.ListTeamMemberIDs(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("list members: %w", err)
	}

	relRows, err := e.store.ListTeamRelations(ctx, teamID)
	if err != nil {
		return domain.TeamSnapshot{}, fmt.Errorf("list relations: %w", err)
	}

	memberNames := make(map[string]string, len(memberIDs))
	snapshots := make([]domain.MemberSnapshot, 0, len(memberIDs))

	// Load all members
	for _, id := range memberIDs {
		ms, err := e.loadMemberSnapshot(ctx, id)
		if err != nil {
			return domain.TeamSnapshot{}, fmt.Errorf("load member %s: %w", id, err)
		}
		memberNames[id] = ms.MemberName
		snapshots = append(snapshots, ms)
	}

	// Convert DB relations to domain relations
	relations := make([]domain.Relation, len(relRows))
	for i, r := range relRows {
		relations[i] = domain.Relation{From: r.FromMemberID, To: r.ToMemberID, Type: domain.RelationType(r.Type)}
	}

	// Build relations and compose prompts
	for i := range snapshots {
		ms := &snapshots[i]
		ms.Relations = domain.ClassifyRelations(ms.MemberID, relations)

		isRoot := ms.MemberID == team.RootMemberID
		protocol := BuildProtocol(ms.MemberName, team.Name, isRoot, ms.Relations, memberNames)
		ms.ComposedPrompt = ComposePrompt(ms.SystemPrompts, protocol)
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

	prompts, err := e.loadPrompts(ctx, memberID)
	if err != nil {
		return domain.MemberSnapshot{}, err
	}

	envs, err := e.loadEnvironments(ctx, memberID)
	if err != nil {
		return domain.MemberSnapshot{}, err
	}

	var gitRepo *domain.SnapshotGitRepo
	if member.GitRepoID.Valid && member.GitRepoID.String != "" {
		repo, err := e.store.GetGitRepo(ctx, member.GitRepoID.String)
		if err != nil {
			return domain.MemberSnapshot{}, fmt.Errorf("get git repo: %w", err)
		}
		gitRepo = &domain.SnapshotGitRepo{Name: repo.Name, URL: repo.Url}
	}

	var systemArgs, customArgs []string
	_ = json.Unmarshal([]byte(profile.SystemArgs), &systemArgs)
	_ = json.Unmarshal([]byte(profile.CustomArgs), &customArgs)
	var dotConfig domain.DotConfig
	_ = json.Unmarshal([]byte(profile.DotConfig), &dotConfig)

	return domain.MemberSnapshot{
		MemberID:       memberID,
		MemberName:     member.Name,
		Binary:         domain.CliBinary(profile.Binary),
		Model:          profile.Model,
		CliProfileName: profile.Name,
		SystemArgs:     systemArgs,
		CustomArgs:     customArgs,
		DotConfig:      dotConfig,
		SystemPrompts:  prompts,
		Environments:   envs,
		GitRepo:        gitRepo,
	}, nil
}

func (e *Engine) loadPrompts(ctx context.Context, memberID string) ([]domain.SnapshotPrompt, error) {
	ids, err := e.store.ListMemberSystemPromptIDs(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	prompts := make([]domain.SnapshotPrompt, 0, len(ids))
	for _, id := range ids {
		sp, err := e.store.GetSystemPrompt(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get prompt %s: %w", id, err)
		}
		prompts = append(prompts, domain.SnapshotPrompt{Name: sp.Name, Prompt: sp.Prompt})
	}
	return prompts, nil
}

func (e *Engine) loadEnvironments(ctx context.Context, memberID string) ([]domain.SnapshotEnvironment, error) {
	ids, err := e.store.ListMemberEnvironmentIDs(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	envs := make([]domain.SnapshotEnvironment, 0, len(ids))
	for _, id := range ids {
		env, err := e.store.GetEnvironment(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get environment %s: %w", id, err)
		}
		envs = append(envs, domain.SnapshotEnvironment{Name: env.Name, Key: env.Key, Value: env.Value})
	}
	return envs, nil
}

func (e *Engine) saveSprint(ctx context.Context, sprint *domain.Sprint, snapshot domain.TeamSnapshot) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	now := time.Now().Unix()
	return e.store.CreateSprint(ctx, generated.CreateSprintParams{
		ID:           sprint.ID,
		Name:         sprint.Name,
		TeamSnapshot: string(snapshotJSON),
		State:        string(sprint.State),
		Error:        sprint.Error,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (e *Engine) failSprint(ctx context.Context, sprintID, errMsg string) {
	_ = e.store.UpdateSprintState(ctx, generated.UpdateSprintStateParams{
		State:     string(domain.SprintErrored),
		Error:     errMsg,
		UpdatedAt: time.Now().Unix(),
		ID:        sprintID,
	})
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
