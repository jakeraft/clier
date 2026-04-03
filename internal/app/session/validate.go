package session

import (
	"fmt"
	"slices"

	"github.com/jakeraft/clier/internal/domain"
)

func validateDelivery(team domain.Team, fromMemberID, toMemberID string) error {
	isUserSender := fromMemberID == domain.UserMemberID
	isUserRecipient := toMemberID == domain.UserMemberID

	if isUserSender || isUserRecipient {
		if !isUserRecipient {
			if !slices.Contains(team.MemberIDs, toMemberID) {
				return fmt.Errorf("recipient not found: %s", toMemberID)
			}
		}
		return nil
	}

	if !slices.Contains(team.MemberIDs, fromMemberID) {
		return fmt.Errorf("sender not found: %s", fromMemberID)
	}
	if !slices.Contains(team.MemberIDs, toMemberID) {
		return fmt.Errorf("recipient not found: %s", toMemberID)
	}
	if !team.MemberRelations(fromMemberID).IsConnectedTo(toMemberID) {
		return fmt.Errorf("no relation from %s to %s", fromMemberID, toMemberID)
	}
	return nil
}
