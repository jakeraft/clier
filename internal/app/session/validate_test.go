package session

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func testTeam() domain.Team {
	return domain.Team{
		ID:               "team-1",
		Name:             "test-team",
		RootTeamMemberID: "leader-1",
		TeamMembers: []domain.TeamMember{
			{ID: "leader-1", MemberID: "m-leader", Name: "Leader"},
			{ID: "worker-1", MemberID: "m-worker", Name: "Worker"},
		},
		Relations: []domain.Relation{
			{From: "leader-1", To: "worker-1", Type: domain.RelationLeader},
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
		disconnected := domain.Team{
			TeamMembers: []domain.TeamMember{
				{ID: "a", MemberID: "m-a", Name: "A"},
				{ID: "b", MemberID: "m-b", Name: "B"},
			},
			Relations: []domain.Relation{},
		}
		if err := validateDelivery(disconnected, "a", "b"); err == nil {
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
