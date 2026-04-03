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

	if err := validateDelivery(sp.TeamSnapshot, fromMemberID, toMemberID); err != nil {
		return err
	}

	senderName := sp.TeamSnapshot.MemberName(fromMemberID)
	if senderName == "" {
		senderName = "user"
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

func validateDelivery(team domain.TeamSnapshot, fromMemberID, toMemberID string) error {
	isUserSender := fromMemberID == domain.UserMemberID
	isUserRecipient := toMemberID == domain.UserMemberID

	if isUserSender || isUserRecipient {
		if !isUserRecipient {
			if _, ok := team.FindMember(toMemberID); !ok {
				return fmt.Errorf("recipient not found: %s", toMemberID)
			}
		}
		return nil
	}

	if _, ok := team.FindMember(fromMemberID); !ok {
		return fmt.Errorf("sender not found: %s", fromMemberID)
	}
	if _, ok := team.FindMember(toMemberID); !ok {
		return fmt.Errorf("recipient not found: %s", toMemberID)
	}
	if !team.IsConnected(fromMemberID, toMemberID) {
		return fmt.Errorf("no relation from %s to %s", fromMemberID, toMemberID)
	}
	return nil
}
