package sprint

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func testTeam() domain.TeamSnapshot {
	return domain.TeamSnapshot{
		TeamName:     "test",
		RootMemberID: "leader-1",
		Members: []domain.TeamMemberSnapshot{
			{
				MemberID:   "leader-1",
				MemberName: "leader",
				Relations:  domain.MemberRelations{Workers: []string{"worker-1"}},
			},
			{
				MemberID:   "worker-1",
				MemberName: "worker",
				Relations:  domain.MemberRelations{Leaders: []string{"leader-1"}},
			},
		},
	}
}

func TestValidateDelivery(t *testing.T) {
	team := testTeam()

	t.Run("UserToMember_Allowed", func(t *testing.T) {
		if err := validateDelivery(team, domain.UserMemberID, "leader-1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("MemberToUser_Allowed", func(t *testing.T) {
		if err := validateDelivery(team, "leader-1", domain.UserMemberID); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ConnectedMembers_Allowed", func(t *testing.T) {
		if err := validateDelivery(team, "leader-1", "worker-1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UnconnectedMembers_Rejected", func(t *testing.T) {
		team := domain.TeamSnapshot{
			Members: []domain.TeamMemberSnapshot{
				{MemberID: "a", MemberName: "A"},
				{MemberID: "b", MemberName: "B"},
			},
		}
		if err := validateDelivery(team, "a", "b"); err == nil {
			t.Error("expected error for unconnected members")
		}
	})

	t.Run("UnknownSender_Rejected", func(t *testing.T) {
		if err := validateDelivery(team, "unknown", "leader-1"); err == nil {
			t.Error("expected error for unknown sender")
		}
	})

	t.Run("UserToUnknownMember_Rejected", func(t *testing.T) {
		if err := validateDelivery(team, domain.UserMemberID, "unknown"); err == nil {
			t.Error("expected error for unknown recipient")
		}
	})
}
