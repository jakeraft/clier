package run

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/domain/resource"
)

// RunStore persists Run lifecycle state and provides read access
// to team/member specs needed for plan building.
type RunStore interface {
	// Run CRUD
	CreateRun(ctx context.Context, run *domain.Run) error
	GetRun(ctx context.Context, id string) (domain.Run, error)
	UpdateRunStatus(ctx context.Context, run *domain.Run) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
	CreateNote(ctx context.Context, n *domain.Note) error

	// Team and member spec reads (used by plan building)
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetClaudeMd(ctx context.Context, id string) (resource.ClaudeMd, error)
	GetSkill(ctx context.Context, id string) (resource.Skill, error)
	GetClaudeSettings(ctx context.Context, id string) (resource.ClaudeSettings, error)
}

// Terminal launches and terminates member processes.
type Terminal interface {
	Launch(runID, runName string, members []domain.MemberPlan) error
	Terminate(runID string) error
	Send(runID, teamMemberID, text string) error
	Attach(runID string, memberID *string) error
}

// Workspace prepares and cleans up member directories.
type Workspace interface {
	Prepare(ctx context.Context, members []domain.MemberPlan) error
	Cleanup(runID string) error
}

// Service orchestrates run execution: build plan, prepare workspace,
// launch terminals, deliver messages.
type Service struct {
	store     RunStore
	terminal  Terminal
	workspace Workspace
	base      string
	runtimes  map[string]AgentRuntime
}

// New creates a run Service.
func New(store RunStore, term Terminal, ws Workspace, base string, runtimes map[string]AgentRuntime) *Service {
	return &Service{store: store, terminal: term, workspace: ws, base: base, runtimes: runtimes}
}

// Start resolves the team, builds the execution plan,
// prepares the workspace, and launches terminals for each member.
func (s *Service) Start(ctx context.Context, team domain.Team) (*domain.Run, error) {
	// Resolve: ID references -> loaded domain objects
	resolved, err := s.resolveTeam(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("resolve team: %w", err)
	}

	runID := uuid.NewString()
	runName := domain.RunName(team.Name, runID)

	// Build: resolved objects -> execution plan with concrete paths
	members := buildPlans(resolved, s.base, runID, s.runtimes)

	r, err := domain.NewRun(runID, runName, team.ID)
	if err != nil {
		return nil, fmt.Errorf("new run: %w", err)
	}
	r.Plan = members

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(runID)
		}
	}()

	// Start: prepare workspace + launch terminals
	if err := s.workspace.Prepare(ctx, members); err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	if err := s.store.CreateRun(ctx, r); err != nil {
		return nil, fmt.Errorf("save run: %w", err)
	}

	if err := s.terminal.Launch(runID, runName, members); err != nil {
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	success = true
	return r, nil
}

// Stop terminates a running execution, updates status, and cleans up workspace.
// Workspace cleanup is best-effort — status is updated even if cleanup fails.
func (s *Service) Stop(ctx context.Context, runID string) error {
	r, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	if err := s.terminal.Terminate(runID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	r.Stop()
	if err := s.store.UpdateRunStatus(ctx, &r); err != nil {
		return fmt.Errorf("update run status: %w", err)
	}

	// Allow OS to release file handles from terminated processes.
	time.Sleep(2 * time.Second)

	if err := s.workspace.Cleanup(runID); err != nil {
		log.Printf("cleanup workspace %s: %v", runID, err)
	}

	return nil
}

// Send delivers a message to the recipient's terminal, then persists it.
// Delivery happens first so that a bad recipient fails before anything is saved.
func (s *Service) Send(ctx context.Context, runID, fromTeamMemberID, toTeamMemberID, content string) error {
	r, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	text := content
	if fromTeamMemberID != "" {
		senderName := fromTeamMemberID
		for _, m := range r.Plan {
			if m.TeamMemberID == fromTeamMemberID {
				senderName = m.MemberName
				break
			}
		}
		text = fmt.Sprintf("[Message from %s] %s", senderName, content)
	}

	if err := s.terminal.Send(runID, toTeamMemberID, text); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}

	msg, err := domain.NewMessage(runID, fromTeamMemberID, toTeamMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// Note persists a progress entry posted by a team member.
func (s *Service) Note(ctx context.Context, runID, teamMemberID, content string) error {
	if _, err := s.store.GetRun(ctx, runID); err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	n, err := domain.NewNote(runID, teamMemberID, content)
	if err != nil {
		return fmt.Errorf("new note: %w", err)
	}
	if err := s.store.CreateNote(ctx, n); err != nil {
		return fmt.Errorf("save note: %w", err)
	}
	return nil
}

