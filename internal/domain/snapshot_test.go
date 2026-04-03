package domain

import "testing"

func TestTeamSnapshot_FindMember(t *testing.T) {
	snap := TeamSnapshot{
		TeamName:     "team-1",
		RootMemberID: "m1",
		Members: []MemberSnapshot{
			{MemberID: "m1", MemberName: "alice"},
			{MemberID: "m2", MemberName: "bob"},
		},
	}

	t.Run("Found", func(t *testing.T) {
		m, ok := snap.FindMember("m1")
		if !ok {
			t.Fatal("expected to find m1")
		}
		if m.MemberName != "alice" {
			t.Errorf("MemberName = %q, want alice", m.MemberName)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, ok := snap.FindMember("unknown")
		if ok {
			t.Error("expected not found")
		}
	})
}

func TestTeamSnapshot_MemberName(t *testing.T) {
	snap := TeamSnapshot{
		Members: []MemberSnapshot{
			{MemberID: "m1", MemberName: "alice"},
		},
	}

	t.Run("Found", func(t *testing.T) {
		if got := snap.MemberName("m1"); got != "alice" {
			t.Errorf("got %q, want alice", got)
		}
	})

	t.Run("NotFound_ReturnsEmpty", func(t *testing.T) {
		if got := snap.MemberName("unknown"); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestTeamSnapshot_IsConnected(t *testing.T) {
	snap := TeamSnapshot{
		Members: []MemberSnapshot{
			{
				MemberID: "m1",
				Relations: MemberRelations{Workers: []string{"m2"}},
			},
			{
				MemberID: "m2",
				Relations: MemberRelations{Leaders: []string{"m1"}},
			},
			{MemberID: "m3"},
		},
	}

	t.Run("Connected_ReturnsTrue", func(t *testing.T) {
		if !snap.IsConnected("m1", "m2") {
			t.Error("m1 should be connected to m2")
		}
	})

	t.Run("NotConnected_ReturnsFalse", func(t *testing.T) {
		if snap.IsConnected("m1", "m3") {
			t.Error("m1 should not be connected to m3")
		}
	})

	t.Run("UnknownSender_ReturnsFalse", func(t *testing.T) {
		if snap.IsConnected("unknown", "m1") {
			t.Error("unknown should not be connected")
		}
	})
}
