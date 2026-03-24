package sprint

import (
	"context"
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

// DeliverMessage validates the relation, persists the message, and delivers it to the recipient's terminal.
func (s *Service) DeliverMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error {
	sp, err := s.store.GetSprint(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}
	if sp.State != domain.SprintRunning {
		return fmt.Errorf("sprint is not running (state: %s)", sp.State)
	}

	senderName := resolveSender(sp.TeamSnapshot.Members, fromMemberID)
	if err := validateDelivery(sp.TeamSnapshot.Members, fromMemberID, toMemberID); err != nil {
		return err
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

func resolveSender(members []domain.MemberSnapshot, fromMemberID string) string {
	if from, ok := findMember(members, fromMemberID); ok {
		return from.MemberName
	}
	return domain.UserMemberID
}

func validateDelivery(members []domain.MemberSnapshot, fromMemberID, toMemberID string) error {
	isUserSender := fromMemberID == "" || fromMemberID == domain.UserMemberID
	isUserRecipient := toMemberID == domain.UserMemberID

	// User can message any member; any member can message user.
	if isUserSender || isUserRecipient {
		if !isUserRecipient {
			if _, ok := findMember(members, toMemberID); !ok {
				return fmt.Errorf("recipient not found: %s", toMemberID)
			}
		}
		return nil
	}

	// Between team members: both must exist and be connected.
	from, ok := findMember(members, fromMemberID)
	if !ok {
		return fmt.Errorf("sender not found: %s", fromMemberID)
	}
	if _, ok := findMember(members, toMemberID); !ok {
		return fmt.Errorf("recipient not found: %s", toMemberID)
	}
	if !from.Relations.IsConnectedTo(toMemberID) {
		return fmt.Errorf("no relation from %s to %s", fromMemberID, toMemberID)
	}
	return nil
}
