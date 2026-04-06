package task

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// TaskStore persists Task lifecycle state and provides read access
// to team/member specs needed for plan building.
type TaskStore interface {
	// Task CRUD
	CreateTask(ctx context.Context, task *domain.Task) error
	GetTask(ctx context.Context, id string) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, task *domain.Task) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
	CreateNote(ctx context.Context, n *domain.Note) error

	// Team and member spec reads (used by plan building)
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetCliProfile(ctx context.Context, id string) (resource.CliProfile, error)
	GetSystemPrompt(ctx context.Context, id string) (resource.SystemPrompt, error)
	GetEnv(ctx context.Context, id string) (resource.Env, error)
	GetGitRepo(ctx context.Context, id string) (resource.GitRepo, error)
}

// Terminal launches and terminates member processes.
type Terminal interface {
	Launch(taskID, taskName string, members []domain.MemberPlan) error
	Terminate(taskID string) error
	Send(taskID, teamMemberID, text string) error
	Attach(taskID string, memberID *string) error
}

// Workspace prepares and cleans up member directories.
type Workspace interface {
	Prepare(ctx context.Context, members []domain.MemberPlan) error
	Cleanup(taskID string) error
}

// AuthChecker reads authentication credentials for the CLI agent.
type AuthChecker interface {
	ReadToken() (string, error)
}

// Service orchestrates task execution: build plan, prepare workspace,
// launch terminals, deliver messages.
type Service struct {
	store     TaskStore
	terminal  Terminal
	workspace Workspace
	base      string
	homeDir   string
}

// New creates a task Service.
func New(store TaskStore, term Terminal, ws Workspace, base, homeDir string) *Service {
	return &Service{store: store, terminal: term, workspace: ws, base: base, homeDir: homeDir}
}

// Start resolves the team, builds the execution plan, expands placeholders,
// prepares the workspace, and launches terminals for each member.
func (s *Service) Start(ctx context.Context, team domain.Team, auth AuthChecker) (*domain.Task, error) {
	// Resolve: ID references -> loaded domain objects
	resolved, err := s.resolveTeam(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("resolve team: %w", err)
	}

	taskID := uuid.NewString()
	taskName := domain.TaskName(team.Name, taskID)

	// Build: resolved objects -> execution plan (with placeholders)
	plan := buildPlans(resolved, taskID)

	t, err := domain.NewTask(taskID, taskName, team.ID)
	if err != nil {
		return nil, fmt.Errorf("new task: %w", err)
	}
	t.Plan = plan

	// Expand: placeholders -> concrete paths
	claudeToken := readAuth(auth)
	members := make([]domain.MemberPlan, 0, len(plan))
	for _, m := range plan {
		members = append(members, expandPlaceholders(m, s.base, s.homeDir, taskID, claudeToken))
	}

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(taskID)
		}
	}()

	// Start: prepare workspace + launch terminals
	if err := s.workspace.Prepare(ctx, members); err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	if err := s.store.CreateTask(ctx, t); err != nil {
		return nil, fmt.Errorf("save task: %w", err)
	}

	if err := s.terminal.Launch(taskID, taskName, members); err != nil {
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	success = true
	return t, nil
}

// Stop terminates a running execution, updates status, and cleans up workspace.
// Workspace cleanup is best-effort — status is updated even if cleanup fails.
func (s *Service) Stop(ctx context.Context, taskID string) error {
	t, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	if err := s.terminal.Terminate(taskID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	t.Stop()
	if err := s.store.UpdateTaskStatus(ctx, &t); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	// Allow OS to release file handles from terminated processes.
	time.Sleep(2 * time.Second)

	if err := s.workspace.Cleanup(taskID); err != nil {
		log.Printf("cleanup workspace %s: %v", taskID, err)
	}

	return nil
}

// Send delivers a message to the recipient's terminal, then persists it.
// Delivery happens first so that a bad recipient fails before anything is saved.
func (s *Service) Send(ctx context.Context, taskID, fromTeamMemberID, toTeamMemberID, content string) error {
	t, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	text := content
	if fromTeamMemberID != "" {
		senderName := fromTeamMemberID
		for _, m := range t.Plan {
			if m.TeamMemberID == fromTeamMemberID {
				senderName = m.MemberName
				break
			}
		}
		text = fmt.Sprintf("[Message from %s] %s", senderName, content)
	}

	if err := s.terminal.Send(taskID, toTeamMemberID, text); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}

	msg, err := domain.NewMessage(taskID, fromTeamMemberID, toTeamMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// Note persists a progress entry posted by a team member.
func (s *Service) Note(ctx context.Context, taskID, teamMemberID, content string) error {
	if _, err := s.store.GetTask(ctx, taskID); err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	n, err := domain.NewNote(taskID, teamMemberID, content)
	if err != nil {
		return fmt.Errorf("new note: %w", err)
	}
	if err := s.store.CreateNote(ctx, n); err != nil {
		return fmt.Errorf("save note: %w", err)
	}
	return nil
}

// readAuth reads the Claude auth token.
func readAuth(auth AuthChecker) string {
	token, err := auth.ReadToken()
	if err == nil && token != "" {
		return token
	}
	return ""
}
