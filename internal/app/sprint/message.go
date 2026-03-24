package sprint

import (
	"context"
	"fmt"
	"slices"

	"github.com/jakeraft/clier/internal/domain"
)

// DeliverMessage validates the relation, persists the message, and delivers it to the recipient's terminal.
func (s *Service) DeliverMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error {
	// Load sprint and validate state
	sprint, err := s.store.GetSprint(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}
	if sprint.State != domain.SprintRunning {
		return fmt.Errorf("sprint is not running (state: %s)", sprint.State)
	}

	snapshot := sprint.TeamSnapshot

	fromName, err := validateMessageRoute(snapshot, fromMemberID, toMemberID)
	if err != nil {
		return err
	}

	// Persist message
	msg, err := domain.NewMessage(sprintID, fromMemberID, toMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	text := fmt.Sprintf("[Message from %s] %s", fromName, content)
	return s.terminal.Send(sprintID, toMemberID, text)
}

func validateMessageRoute(snapshot domain.TeamSnapshot, fromID, toID string) (string, error) {
	var fromName string
	var fromRelations domain.MemberRelations
	found := false

	for _, m := range snapshot.Members {
		if m.MemberID == fromID {
			fromName = m.MemberName
			fromRelations = m.Relations
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("sender not found: %s", fromID)
	}

	// Check recipient exists
	recipientExists := false
	for _, m := range snapshot.Members {
		if m.MemberID == toID {
			recipientExists = true
			break
		}
	}
	if !recipientExists {
		return "", fmt.Errorf("recipient not found: %s", toID)
	}

	// Validate relation: sender can message leaders, workers, or peers
	if slices.Contains(fromRelations.Leaders, toID) ||
		slices.Contains(fromRelations.Workers, toID) ||
		slices.Contains(fromRelations.Peers, toID) {
		return fromName, nil
	}

	return "", fmt.Errorf("no relation from %s to %s", fromID, toID)
}
