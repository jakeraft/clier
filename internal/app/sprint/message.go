package sprint

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// DeliverMessage validates the relation, persists the message, and delivers it to the recipient's terminal.
func (s *Service) DeliverMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error {
	sprint, err := s.store.GetSprint(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}
	if sprint.State != domain.SprintRunning {
		return fmt.Errorf("sprint is not running (state: %s)", sprint.State)
	}

	if _, ok := findMember(sprint.TeamSnapshot.Members, toMemberID); !ok {
		return fmt.Errorf("recipient not found: %s", toMemberID)
	}

	senderName := "user"
	if fromMemberID != "" {
		from, ok := findMember(sprint.TeamSnapshot.Members, fromMemberID)
		if !ok {
			return fmt.Errorf("sender not found: %s", fromMemberID)
		}
		if !from.Relations.IsConnectedTo(toMemberID) {
			return fmt.Errorf("no relation from %s to %s", fromMemberID, toMemberID)
		}
		senderName = from.MemberName
	}

	msg, err := domain.NewMessage(sprintID, fromMemberID, toMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	text := fmt.Sprintf("[Message from %s] %s", senderName, content)
	return s.terminal.Send(sprintID, toMemberID, text)
}
