package run

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Terminal launches and terminates member processes.
type Terminal interface {
	Terminate(plan *RunPlan) error
	Send(plan *RunPlan, teamMemberID int64, text string) error
}

// Service orchestrates run messaging and lifecycle.
type Service struct {
	terminal Terminal
	sleep    func(time.Duration)
}

// New creates a run Service.
func New(term Terminal) *Service {
	return &Service{
		terminal: term,
		sleep:    time.Sleep,
	}
}

// Stop terminates a running execution.
func (s *Service) Stop(plan *RunPlan) error {
	if err := s.terminal.Terminate(plan); err != nil {
		return fmt.Errorf("terminate terminal %s: %w", plan.RunID, err)
	}

	// Allow OS to release file handles from terminated processes.
	s.sleep(2 * time.Second)

	return nil
}

// Send delivers a message to the recipient's terminal.
func (s *Service) Send(plan *RunPlan, fromMemberID, toMemberID *int64, content string) error {
	if toMemberID == nil {
		return errors.New("recipient member id is required")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("message content must not be empty")
	}

	text := content
	if fromMemberID != nil {
		text = fmt.Sprintf("[Message from %s] %s", strconv.FormatInt(*fromMemberID, 10), content)
	}

	if err := s.terminal.Send(plan, *toMemberID, text); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}
	return nil
}

// Note validates a progress entry posted by a member.
func (s *Service) Note(memberID *int64, content string) error {
	_ = memberID
	if strings.TrimSpace(content) == "" {
		return errors.New("note content must not be empty")
	}
	return nil
}
