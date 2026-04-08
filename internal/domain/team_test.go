package domain

import (
	"testing"
)

const (
	rootMemberSpecID int64 = 100
	rootMemberName         = "Root Member"
)

// nextID is a simple counter to simulate server-assigned IDs in tests.
var nextID int64 = 1000

func nextTestID() int64 {
	nextID++
	return nextID
}

func createTeamWithMembers(t *testing.T, extras ...struct {
	id   int64
	name string
}) (*Team, []TeamMember) {
	t.Helper()
	team, err := NewTeam("my-team", rootMemberSpecID, rootMemberName)
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	// Simulate server assigning IDs to the root team member.
	team.TeamMembers[0].ID = nextTestID()
	rootID := team.TeamMembers[0].ID
	team.RootTeamMemberID = &rootID

	added := []TeamMember{}
	for _, e := range extras {
		tm, err := team.AddTeamMember(e.id, e.name)
		if err != nil {
			t.Fatalf("AddTeamMember(%d, %q): %v", e.id, e.name, err)
		}
		// Simulate server assigning an ID.
		tm.ID = nextTestID()
		// Update the slice element too.
		team.TeamMembers[len(team.TeamMembers)-1].ID = tm.ID
		added = append(added, *tm)
	}
	return team, added
}

func member(id int64, name string) struct {
	id   int64
	name string
} {
	return struct {
		id   int64
		name string
	}{id, name}
}

func TestNewTeam(t *testing.T) {
	t.Run("ValidInputs_CreatesTeamWithRootTeamMember", func(t *testing.T) {
		team, err := NewTeam("my-team", int64(1), "Agent One")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.ID != 0 {
			t.Errorf("Team.ID = %d, want 0 (server assigns)", team.ID)
		}
		if team.Name != "my-team" {
			t.Errorf("Name = %q, want %q", team.Name, "my-team")
		}
		if len(team.TeamMembers) != 1 {
			t.Fatalf("TeamMembers length = %d, want 1", len(team.TeamMembers))
		}
		root := team.TeamMembers[0]
		if root.MemberID != 1 {
			t.Errorf("Root MemberID = %d, want %d", root.MemberID, 1)
		}
		if root.Name != "Agent One" {
			t.Errorf("Root Name = %q, want %q", root.Name, "Agent One")
		}
		if team.RootTeamMemberID != nil {
			t.Errorf("RootTeamMemberID = %v, want nil (server assigns)", team.RootTeamMemberID)
		}
		if len(team.Relations) != 0 {
			t.Errorf("Relations = %v, want []", team.Relations)
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("  ", int64(1), "Agent One")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ZeroMemberID_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("my-team", int64(0), "Agent One")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyMemberName_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("my-team", int64(1), "  ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAddTeamMember_DuplicateMemberIDAllowed(t *testing.T) {
	team, err := NewTeam("my-team", int64(1), "Agent One")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	// Simulate server assigning root team member ID.
	team.TeamMembers[0].ID = nextTestID()
	rootID := team.TeamMembers[0].ID
	team.RootTeamMemberID = &rootID

	tm1, err := team.AddTeamMember(int64(1), "Agent One Copy")
	if err != nil {
		t.Fatalf("first AddTeamMember: %v", err)
	}
	tm1.ID = nextTestID()
	team.TeamMembers[len(team.TeamMembers)-1].ID = tm1.ID

	tm2, err := team.AddTeamMember(int64(1), "Agent One Third")
	if err != nil {
		t.Fatalf("second AddTeamMember: %v", err)
	}
	tm2.ID = nextTestID()
	team.TeamMembers[len(team.TeamMembers)-1].ID = tm2.ID

	if len(team.TeamMembers) != 3 {
		t.Fatalf("TeamMembers length = %d, want 3", len(team.TeamMembers))
	}

	// All three share the same MemberID but have different TeamMember IDs.
	if tm1.MemberID != 1 || tm2.MemberID != 1 {
		t.Errorf("MemberIDs should all be 1, got %d and %d", tm1.MemberID, tm2.MemberID)
	}
	if tm1.ID == tm2.ID {
		t.Error("TeamMember IDs should be unique")
	}
	if tm1.ID == team.TeamMembers[0].ID {
		t.Error("TeamMember IDs should differ from root")
	}
}

func TestAddTeamMember_Validation(t *testing.T) {
	team, _ := NewTeam("my-team", int64(1), "Agent One")

	t.Run("ZeroMemberID_ReturnsError", func(t *testing.T) {
		_, err := team.AddTeamMember(int64(0), "Name")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		_, err := team.AddTeamMember(int64(2), "  ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRemoveTeamMember(t *testing.T) {
	t.Run("RemovesByTeamMemberID_AndCleansRelations", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		tmID := added[0].ID
		rootID := *team.RootTeamMemberID

		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID})
		if err := team.RemoveTeamMember(tmID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(team.TeamMembers) != 1 {
			t.Errorf("TeamMembers length = %d, want 1", len(team.TeamMembers))
		}
		if len(team.Relations) != 0 {
			t.Errorf("Relations = %v, want []", team.Relations)
		}
	})

	t.Run("NonexistentTeamMember_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		if err := team.RemoveTeamMember(int64(999999)); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRemoveTeamMember_CannotRemoveRoot(t *testing.T) {
	team, _ := createTeamWithMembers(t)
	rootID := *team.RootTeamMemberID
	if err := team.RemoveTeamMember(rootID); err == nil {
		t.Fatal("expected error when removing root team member, got nil")
	}
}

func TestAddRelation_UsesTeamMemberID(t *testing.T) {
	t.Run("ValidLeader_AddsRelation", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		rootID := *team.RootTeamMemberID
		tmID := added[0].ID

		err := team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(team.Relations) != 1 {
			t.Fatalf("Relations length = %d, want 1", len(team.Relations))
		}
		r := team.Relations[0]
		if r.FromTeamMemberID != rootID || r.ToTeamMemberID != tmID {
			t.Errorf("Relation = %+v, want {From:%d To:%d}", r, rootID, tmID)
		}
	})

	t.Run("SelfRelation_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		rootID := *team.RootTeamMemberID
		err := team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: rootID})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonMemberFrom_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		err := team.AddRelation(Relation{FromTeamMemberID: int64(999999), ToTeamMemberID: added[0].ID})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonMemberTo_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		err := team.AddRelation(Relation{FromTeamMemberID: *team.RootTeamMemberID, ToTeamMemberID: int64(999999)})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Duplicate_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		rootID := *team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID})
		if err := team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("LeaderUniqueness_SecondLeader_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t,
			member(2, "Agent Two"),
			member(3, "Agent Three"),
		)
		rootID := *team.RootTeamMemberID
		tm2ID := added[0].ID
		tm3ID := added[1].ID
		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tm2ID})
		if err := team.AddRelation(Relation{FromTeamMemberID: tm3ID, ToTeamMemberID: tm2ID}); err == nil {
			t.Fatal("expected error for second leader, got nil")
		}
	})

	t.Run("MutualLeaderCycle_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		rootID := *team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID})
		if err := team.AddRelation(Relation{FromTeamMemberID: tmID, ToTeamMemberID: rootID}); err == nil {
			t.Fatal("expected error for mutual leader cycle, got nil")
		}
	})
}

func TestRemoveRelation(t *testing.T) {
	t.Run("ExistingRelation_RemovesIt", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		rootID := *team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tmID})
		if err := team.RemoveRelation(rootID, tmID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(team.Relations) != 0 {
			t.Errorf("Relations = %v, want []", team.Relations)
		}
	})

	t.Run("Nonexistent_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		if err := team.RemoveRelation(*team.RootTeamMemberID, int64(999999)); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMemberRelations_UsesTeamMemberID(t *testing.T) {
	t.Run("MultipleRelations_ReturnsLeadersWorkers", func(t *testing.T) {
		team, added := createTeamWithMembers(t,
			member(2, "Agent Two"),
			member(3, "Agent Three"),
		)
		rootID := *team.RootTeamMemberID
		tm2ID := added[0].ID
		tm3ID := added[1].ID

		// root is leader of tm2 (root -> tm2 leader)
		_ = team.AddRelation(Relation{FromTeamMemberID: rootID, ToTeamMemberID: tm2ID})
		// tm3 is leader of root (tm3 -> root leader)
		_ = team.AddRelation(Relation{FromTeamMemberID: tm3ID, ToTeamMemberID: rootID})

		rel := team.MemberRelations(rootID)
		if len(rel.Workers) != 1 || rel.Workers[0] != tm2ID {
			t.Errorf("Workers = %v, want [%d]", rel.Workers, tm2ID)
		}
		if len(rel.Leaders) != 1 || rel.Leaders[0] != tm3ID {
			t.Errorf("Leaders = %v, want [%d]", rel.Leaders, tm3ID)
		}
	})

	t.Run("NoRelations_ReturnsEmptySlices", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		_ = team
		rel := team.MemberRelations(added[0].ID)
		if len(rel.Leaders) != 0 {
			t.Errorf("Leaders = %v, want []", rel.Leaders)
		}
		if len(rel.Workers) != 0 {
			t.Errorf("Workers = %v, want []", rel.Workers)
		}
	})
}

func TestReplaceComposition(t *testing.T) {
	t.Run("ValidReplacement_ReplacesAtomically", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)

		newRootID := nextTestID()
		newMemberID := nextTestID()
		newRoot := TeamMember{ID: newRootID, MemberID: 10, Name: "New Root"}
		newMember := TeamMember{ID: newMemberID, MemberID: 20, Name: "New Member"}

		err := team.ReplaceComposition(
			"new-team",
			newRootID,
			[]TeamMember{newRoot, newMember},
			[]Relation{{FromTeamMemberID: newRootID, ToTeamMemberID: newMemberID}},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.Name != "new-team" {
			t.Errorf("Name = %q, want %q", team.Name, "new-team")
		}
		if team.RootTeamMemberID == nil || *team.RootTeamMemberID != newRootID {
			t.Errorf("RootTeamMemberID = %v, want %d", team.RootTeamMemberID, newRootID)
		}
		if len(team.TeamMembers) != 2 {
			t.Fatalf("TeamMembers length = %d, want 2", len(team.TeamMembers))
		}
		if len(team.Relations) != 1 {
			t.Fatalf("Relations length = %d, want 1", len(team.Relations))
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		err := team.ReplaceComposition("", *team.RootTeamMemberID, team.TeamMembers, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("RootNotInMembers_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		nonexistentID := nextTestID()
		memberTMID := nextTestID()
		err := team.ReplaceComposition("team", nonexistentID,
			[]TeamMember{{ID: memberTMID, MemberID: 10, Name: "X"}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("InvalidRelation_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		rootTMID := nextTestID()
		root := TeamMember{ID: rootTMID, MemberID: 10, Name: "A"}
		err := team.ReplaceComposition("team", rootTMID,
			[]TeamMember{root},
			[]Relation{{FromTeamMemberID: rootTMID, ToTeamMemberID: int64(999999)}},
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ZeroTeamMemberID_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		err := team.ReplaceComposition("team", int64(1),
			[]TeamMember{{ID: 0, MemberID: 10, Name: "A"}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error for zero team member ID, got nil")
		}
	})

	t.Run("EmptyTeamMemberName_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		tmID := nextTestID()
		err := team.ReplaceComposition("team", int64(1),
			[]TeamMember{{ID: tmID, MemberID: 10, Name: ""}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error for empty team member name, got nil")
		}
	})
}

func TestFindTeamMember(t *testing.T) {
	t.Run("ExistingTeamMember_ReturnsIt", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		tmID := added[0].ID

		found, ok := team.FindTeamMember(tmID)
		if !ok {
			t.Fatal("expected to find team member, got not found")
		}
		if found.ID != tmID {
			t.Errorf("ID = %d, want %d", found.ID, tmID)
		}
		if found.MemberID != 2 {
			t.Errorf("MemberID = %d, want %d", found.MemberID, 2)
		}
		if found.Name != "Agent Two" {
			t.Errorf("Name = %q, want %q", found.Name, "Agent Two")
		}
	})

	t.Run("FindRoot_ReturnsIt", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		found, ok := team.FindTeamMember(*team.RootTeamMemberID)
		if !ok {
			t.Fatal("expected to find root team member")
		}
		if found.MemberID != rootMemberSpecID {
			t.Errorf("MemberID = %d, want %d", found.MemberID, rootMemberSpecID)
		}
	})

	t.Run("Nonexistent_ReturnsFalse", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		_, ok := team.FindTeamMember(int64(999999))
		if ok {
			t.Fatal("expected not found, got found")
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("ValidName_ChangesName", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		name := "new"
		if err := team.Update(&name, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.Name != "new" {
			t.Errorf("Name = %q, want %q", team.Name, "new")
		}
	})

	t.Run("ExistingTeamMember_ChangesRootTeamMemberID", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member(2, "Agent Two"))
		tmID := added[0].ID
		if err := team.Update(nil, &tmID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.RootTeamMemberID == nil || *team.RootTeamMemberID != tmID {
			t.Errorf("RootTeamMemberID = %v, want %d", team.RootTeamMemberID, tmID)
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		name := ""
		if err := team.Update(&name, nil); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonexistentRootTeamMemberID_ReturnsError", func(t *testing.T) {
		team, _ := createTeamWithMembers(t)
		id := int64(999999)
		if err := team.Update(nil, &id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestIsConnectedTo(t *testing.T) {
	rel := MemberRelations{
		Leaders: []int64{1},
		Workers: []int64{2},
	}

	t.Run("Connected_ReturnsTrue", func(t *testing.T) {
		for _, id := range []int64{1, 2} {
			if !rel.IsConnectedTo(id) {
				t.Errorf("IsConnectedTo(%d) = false, want true", id)
			}
		}
	})

	t.Run("NotConnected_ReturnsFalse", func(t *testing.T) {
		if rel.IsConnectedTo(int64(999)) {
			t.Error("IsConnectedTo(999) = true, want false")
		}
	})

	t.Run("Empty_ReturnsFalse", func(t *testing.T) {
		empty := MemberRelations{}
		if empty.IsConnectedTo(int64(1)) {
			t.Error("IsConnectedTo(1) = true, want false")
		}
	})
}
