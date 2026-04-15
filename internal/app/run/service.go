package run

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Terminal launches and terminates member processes.
type Terminal interface {
	Terminate(plan *RunPlan) error
	Send(plan *RunPlan, memberName string, text string) error
}

// PlanStore persists and retrieves run plans.
type PlanStore interface {
	Save(plan *RunPlan) error
}

// Service orchestrates run messaging and lifecycle.
type Service struct {
	terminal Terminal
	store    PlanStore
	sleep    func(time.Duration)
}

// New creates a run Service.
func New(term Terminal, store PlanStore) *Service {
	return &Service{
		terminal: term,
		store:    store,
		sleep:    time.Sleep,
	}
}

// Stop terminates a running execution and persists the stopped state.
func (s *Service) Stop(plan *RunPlan) error {
	if err := s.terminal.Terminate(plan); err != nil {
		return fmt.Errorf("terminate terminal %s: %w", plan.RunID, err)
	}

	// Allow OS to release file handles from terminated processes.
	s.sleep(2 * time.Second)

	plan.MarkStopped()
	if err := s.store.Save(plan); err != nil {
		return fmt.Errorf("save stopped plan %s: %w", plan.RunID, err)
	}
	return nil
}

// Send delivers a message to the recipient's terminal and records it in the plan.
func (s *Service) Send(plan *RunPlan, fromMember, toMember *string, content string) error {
	if toMember == nil {
		return errors.New("recipient member name is required")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("message content must not be empty")
	}

	text := content
	if fromMember != nil {
		text = fmt.Sprintf("[Message from %s] %s", *fromMember, content)
	}

	if err := s.terminal.Send(plan, *toMember, text); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}

	if err := plan.AddMessage(fromMember, toMember, content); err != nil {
		return fmt.Errorf("record message: %w", err)
	}
	if err := s.store.Save(plan); err != nil {
		return fmt.Errorf("save plan %s: %w", plan.RunID, err)
	}
	return nil
}

// Note records a progress entry posted by a member.
func (s *Service) Note(plan *RunPlan, member *string, content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("note content must not be empty")
	}

	if err := plan.AddNote(member, content); err != nil {
		return fmt.Errorf("record note: %w", err)
	}
	if err := s.store.Save(plan); err != nil {
		return fmt.Errorf("save plan %s: %w", plan.RunID, err)
	}
	return nil
}
