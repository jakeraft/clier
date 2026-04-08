package run

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jakeraft/clier/internal/domain"
)

// RunStore persists Run lifecycle state.
type RunStore interface {
	// Run CRUD
	CreateRun(ctx context.Context, run *domain.Run) error
	GetRun(ctx context.Context, id int64) (domain.Run, error)
	UpdateRunStatus(ctx context.Context, run *domain.Run) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
	CreateNote(ctx context.Context, n *domain.Note) error
}

// Terminal launches and terminates member processes.
type Terminal interface {
	Launch(runID, runName string, members []domain.MemberPlan) error
	Terminate(runID string) error
	Send(runID, teamMemberID, text string) error
	Attach(runID string, memberID *string) error
}

// Service orchestrates run messaging and lifecycle.
type Service struct {
	store    RunStore
	terminal Terminal
}

// New creates a run Service.
func New(store RunStore, term Terminal) *Service {
	return &Service{store: store, terminal: term}
}

// Stop terminates a running execution and updates status.
func (s *Service) Stop(ctx context.Context, runID int64) error {
	r, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	runIDStr := strconv.FormatInt(runID, 10)
	if err := s.terminal.Terminate(runIDStr); err != nil {
		log.Printf("terminate terminal %s: %v", runIDStr, err)
	}

	r.Stop()
	if err := s.store.UpdateRunStatus(ctx, &r); err != nil {
		return fmt.Errorf("update run status: %w", err)
	}

	// Allow OS to release file handles from terminated processes.
	time.Sleep(2 * time.Second)

	return nil
}

// Send delivers a message to the recipient's terminal, then persists it.
// Delivery happens first so that a bad recipient fails before anything is saved.
func (s *Service) Send(ctx context.Context, runID int64, fromTeamMemberID, toTeamMemberID *int64, content string) error {
	if _, err := s.store.GetRun(ctx, runID); err != nil {
		return fmt.Errorf("get run: %w", err)
	}

	text := content
	if fromTeamMemberID != nil {
		text = fmt.Sprintf("[Message from %s] %s", strconv.FormatInt(*fromTeamMemberID, 10), content)
	}

	toStr := ""
	if toTeamMemberID != nil {
		toStr = strconv.FormatInt(*toTeamMemberID, 10)
	}
	runIDStr := strconv.FormatInt(runID, 10)
	if err := s.terminal.Send(runIDStr, toStr, text); err != nil {
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
func (s *Service) Note(ctx context.Context, runID int64, teamMemberID *int64, content string) error {
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
