package sprint

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func testMembers() []domain.MemberSnapshot {
	return []domain.MemberSnapshot{
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
	}
}

func TestResolveSender(t *testing.T) {
	members := testMembers()

	t.Run("KnownMember_ReturnsMemberName", func(t *testing.T) {
		got := resolveSender(members, "leader-1")
		if got != "leader" {
			t.Errorf("got %q, want %q", got, "leader")
		}
	})

	t.Run("UserMemberID_ReturnsUserMemberID", func(t *testing.T) {
		got := resolveSender(members, domain.UserMemberID)
		if got != domain.UserMemberID {
			t.Errorf("got %q, want %q", got, domain.UserMemberID)
		}
	})

	t.Run("UnknownID_ReturnsUserMemberID", func(t *testing.T) {
		got := resolveSender(members, "unknown")
		if got != domain.UserMemberID {
			t.Errorf("got %q, want %q", got, domain.UserMemberID)
		}
	})
}

func TestValidateDelivery(t *testing.T) {
	members := testMembers()

	t.Run("UserToMember_Allowed", func(t *testing.T) {
		if err := validateDelivery(members, domain.UserMemberID, "leader-1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UserToUser_Allowed", func(t *testing.T) {
		if err := validateDelivery(members, domain.UserMemberID, domain.UserMemberID); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("MemberToUser_Allowed", func(t *testing.T) {
		if err := validateDelivery(members, "leader-1", domain.UserMemberID); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ConnectedMembers_Allowed", func(t *testing.T) {
		if err := validateDelivery(members, "leader-1", "worker-1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UnconnectedMembers_Rejected", func(t *testing.T) {
		// worker→leader is not connected (worker has leader as leader, not as worker)
		members := []domain.MemberSnapshot{
			{MemberID: "a", MemberName: "A"},
			{MemberID: "b", MemberName: "B"},
		}
		err := validateDelivery(members, "a", "b")
		if err == nil {
			t.Error("expected error for unconnected members")
		}
	})

	t.Run("UserToUnknownMember_Rejected", func(t *testing.T) {
		err := validateDelivery(members, domain.UserMemberID, "unknown")
		if err == nil {
			t.Error("expected error for unknown recipient")
		}
	})

	t.Run("UnknownSender_Rejected", func(t *testing.T) {
		err := validateDelivery(members, "unknown", "leader-1")
		if err == nil {
			t.Error("expected error for unknown sender")
		}
	})
}
