package session

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
)

func validateDelivery(team domain.Team, fromTeamMemberID, toTeamMemberID string) error {
	isUserSender := fromTeamMemberID == domain.UserMemberID
	isUserRecipient := toTeamMemberID == domain.UserMemberID

	if isUserSender || isUserRecipient {
		if !isUserRecipient {
			if _, ok := team.FindTeamMember(toTeamMemberID); !ok {
				return fmt.Errorf("recipient not found: %s", toTeamMemberID)
			}
		}
		return nil
	}

	if _, ok := team.FindTeamMember(fromTeamMemberID); !ok {
		return fmt.Errorf("sender not found: %s", fromTeamMemberID)
	}
	if _, ok := team.FindTeamMember(toTeamMemberID); !ok {
		return fmt.Errorf("recipient not found: %s", toTeamMemberID)
	}
	if !team.MemberRelations(fromTeamMemberID).IsConnectedTo(toTeamMemberID) {
		return fmt.Errorf("no relation from %s to %s", fromTeamMemberID, toTeamMemberID)
	}
	return nil
}
