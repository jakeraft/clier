package domain

import (
	"testing"

	"github.com/google/uuid"
)

const (
	rootMemberSpecID = "root-member-spec"
	rootMemberName   = "Root Member"
)

func createTeamWithMembers(t *testing.T, extras ...struct{ id, name string }) (*Team, []TeamMember) {
	t.Helper()
	team, err := NewTeam("my-team", rootMemberSpecID, rootMemberName)
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	added := []TeamMember{}
	for _, e := range extras {
		tm, err := team.AddTeamMember(e.id, e.name)
		if err != nil {
			t.Fatalf("AddTeamMember(%q, %q): %v", e.id, e.name, err)
		}
		added = append(added, *tm)
	}
	return team, added
}

func member(id, name string) struct{ id, name string } {
	return struct{ id, name string }{id, name}
}

func TestNewTeam(t *testing.T) {
	t.Run("ValidInputs_CreatesTeamWithRootTeamMember", func(t *testing.T) {
		team, err := NewTeam("my-team", "spec-1", "Agent One")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := uuid.Parse(team.ID); err != nil {
			t.Errorf("Team.ID %q is not a valid UUID", team.ID)
		}
		if team.Name != "my-team" {
			t.Errorf("Name = %q, want %q", team.Name, "my-team")
		}
		if len(team.TeamMembers) != 1 {
			t.Fatalf("TeamMembers length = %d, want 1", len(team.TeamMembers))
		}
		root := team.TeamMembers[0]
		if _, err := uuid.Parse(root.ID); err != nil {
			t.Errorf("Root TeamMember.ID %q is not a valid UUID", root.ID)
		}
		if root.MemberID != "spec-1" {
			t.Errorf("Root MemberID = %q, want %q", root.MemberID, "spec-1")
		}
		if root.Name != "Agent One" {
			t.Errorf("Root Name = %q, want %q", root.Name, "Agent One")
		}
		if team.RootTeamMemberID != root.ID {
			t.Errorf("RootTeamMemberID = %q, want %q", team.RootTeamMemberID, root.ID)
		}
		if len(team.Relations) != 0 {
			t.Errorf("Relations = %v, want []", team.Relations)
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("  ", "spec-1", "Agent One")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyMemberID_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("my-team", "  ", "Agent One")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyMemberName_ReturnsError", func(t *testing.T) {
		_, err := NewTeam("my-team", "spec-1", "  ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAddTeamMember_DuplicateMemberIDAllowed(t *testing.T) {
	team, err := NewTeam("my-team", "spec-1", "Agent One")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}

	tm1, err := team.AddTeamMember("spec-1", "Agent One Copy")
	if err != nil {
		t.Fatalf("first AddTeamMember: %v", err)
	}
	tm2, err := team.AddTeamMember("spec-1", "Agent One Third")
	if err != nil {
		t.Fatalf("second AddTeamMember: %v", err)
	}

	if len(team.TeamMembers) != 3 {
		t.Fatalf("TeamMembers length = %d, want 3", len(team.TeamMembers))
	}

	// All three share the same MemberID but have different TeamMember IDs.
	if tm1.MemberID != "spec-1" || tm2.MemberID != "spec-1" {
		t.Errorf("MemberIDs should all be spec-1, got %q and %q", tm1.MemberID, tm2.MemberID)
	}
	if tm1.ID == tm2.ID {
		t.Error("TeamMember IDs should be unique")
	}
	if tm1.ID == team.TeamMembers[0].ID {
		t.Error("TeamMember IDs should differ from root")
	}
}

func TestAddTeamMember_Validation(t *testing.T) {
	team, _ := NewTeam("my-team", "spec-1", "Agent One")

	t.Run("EmptyMemberID_ReturnsError", func(t *testing.T) {
		_, err := team.AddTeamMember("", "Name")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		_, err := team.AddTeamMember("spec-2", "  ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRemoveTeamMember(t *testing.T) {
	t.Run("RemovesByTeamMemberID_AndCleansRelations", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		tmID := added[0].ID
		rootID := team.RootTeamMemberID

		_ = team.AddRelation(Relation{From: rootID, To: tmID})
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
		team, _ := NewTeam("team", "spec-1", "Agent One")
		if err := team.RemoveTeamMember("nonexistent-uuid"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRemoveTeamMember_CannotRemoveRoot(t *testing.T) {
	team, _ := NewTeam("team", "spec-1", "Agent One")
	rootID := team.RootTeamMemberID
	if err := team.RemoveTeamMember(rootID); err == nil {
		t.Fatal("expected error when removing root team member, got nil")
	}
}

func TestAddRelation_UsesTeamMemberID(t *testing.T) {
	t.Run("ValidLeader_AddsRelation", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		rootID := team.RootTeamMemberID
		tmID := added[0].ID

		err := team.AddRelation(Relation{From: rootID, To: tmID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(team.Relations) != 1 {
			t.Fatalf("Relations length = %d, want 1", len(team.Relations))
		}
		r := team.Relations[0]
		if r.From != rootID || r.To != tmID {
			t.Errorf("Relation = %+v, want {From:%s To:%s}", r, rootID, tmID)
		}
	})

	t.Run("SelfRelation_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		rootID := team.RootTeamMemberID
		err := team.AddRelation(Relation{From: rootID, To: rootID})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonMemberFrom_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		err := team.AddRelation(Relation{From: "nonexistent", To: added[0].ID})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonMemberTo_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		err := team.AddRelation(Relation{From: team.RootTeamMemberID, To: "nonexistent"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Duplicate_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		rootID := team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{From: rootID, To: tmID})
		if err := team.AddRelation(Relation{From: rootID, To: tmID}); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("LeaderUniqueness_SecondLeader_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t,
			member("spec-2", "Agent Two"),
			member("spec-3", "Agent Three"),
		)
		rootID := team.RootTeamMemberID
		tm2ID := added[0].ID
		tm3ID := added[1].ID
		_ = team.AddRelation(Relation{From: rootID, To: tm2ID})
		if err := team.AddRelation(Relation{From: tm3ID, To: tm2ID}); err == nil {
			t.Fatal("expected error for second leader, got nil")
		}
	})

	t.Run("MutualLeaderCycle_ReturnsError", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		rootID := team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{From: rootID, To: tmID})
		if err := team.AddRelation(Relation{From: tmID, To: rootID}); err == nil {
			t.Fatal("expected error for mutual leader cycle, got nil")
		}
	})
}

func TestRemoveRelation(t *testing.T) {
	t.Run("ExistingRelation_RemovesIt", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		rootID := team.RootTeamMemberID
		tmID := added[0].ID
		_ = team.AddRelation(Relation{From: rootID, To: tmID})
		if err := team.RemoveRelation(rootID, tmID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(team.Relations) != 0 {
			t.Errorf("Relations = %v, want []", team.Relations)
		}
	})

	t.Run("Nonexistent_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		if err := team.RemoveRelation(team.RootTeamMemberID, "x"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMemberRelations_UsesTeamMemberID(t *testing.T) {
	t.Run("MultipleRelations_ReturnsLeadersWorkers", func(t *testing.T) {
		team, added := createTeamWithMembers(t,
			member("spec-2", "Agent Two"),
			member("spec-3", "Agent Three"),
		)
		rootID := team.RootTeamMemberID
		tm2ID := added[0].ID
		tm3ID := added[1].ID

		// root is leader of tm2 (root -> tm2 leader)
		_ = team.AddRelation(Relation{From: rootID, To: tm2ID})
		// tm3 is leader of root (tm3 -> root leader)
		_ = team.AddRelation(Relation{From: tm3ID, To: rootID})

		rel := team.MemberRelations(rootID)
		if len(rel.Workers) != 1 || rel.Workers[0] != tm2ID {
			t.Errorf("Workers = %v, want [%s]", rel.Workers, tm2ID)
		}
		if len(rel.Leaders) != 1 || rel.Leaders[0] != tm3ID {
			t.Errorf("Leaders = %v, want [%s]", rel.Leaders, tm3ID)
		}
	})

	t.Run("NoRelations_ReturnsEmptySlices", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
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
		team, _ := NewTeam("old-team", "spec-1", "Agent One")

		newRoot := TeamMember{ID: uuid.NewString(), MemberID: "spec-a", Name: "New Root"}
		newMember := TeamMember{ID: uuid.NewString(), MemberID: "spec-b", Name: "New Member"}

		err := team.ReplaceComposition(
			"new-team",
			newRoot.ID,
			[]TeamMember{newRoot, newMember},
			[]Relation{{From: newRoot.ID, To: newMember.ID}},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.Name != "new-team" {
			t.Errorf("Name = %q, want %q", team.Name, "new-team")
		}
		if team.RootTeamMemberID != newRoot.ID {
			t.Errorf("RootTeamMemberID = %q, want %q", team.RootTeamMemberID, newRoot.ID)
		}
		if len(team.TeamMembers) != 2 {
			t.Fatalf("TeamMembers length = %d, want 2", len(team.TeamMembers))
		}
		if len(team.Relations) != 1 {
			t.Fatalf("Relations length = %d, want 1", len(team.Relations))
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		err := team.ReplaceComposition("", team.RootTeamMemberID, team.TeamMembers, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("RootNotInMembers_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		err := team.ReplaceComposition("team", "nonexistent-id",
			[]TeamMember{{ID: uuid.NewString(), MemberID: "spec-x", Name: "X"}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("InvalidRelation_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		root := TeamMember{ID: uuid.NewString(), MemberID: "spec-a", Name: "A"}
		err := team.ReplaceComposition("team", root.ID,
			[]TeamMember{root},
			[]Relation{{From: root.ID, To: "nonexistent"}},
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyTeamMemberID_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		err := team.ReplaceComposition("team", "some-id",
			[]TeamMember{{ID: "", MemberID: "spec-a", Name: "A"}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error for empty team member ID, got nil")
		}
	})

	t.Run("EmptyTeamMemberName_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		err := team.ReplaceComposition("team", "some-id",
			[]TeamMember{{ID: uuid.NewString(), MemberID: "spec-a", Name: ""}},
			nil,
		)
		if err == nil {
			t.Fatal("expected error for empty team member name, got nil")
		}
	})
}

func TestFindTeamMember(t *testing.T) {
	t.Run("ExistingTeamMember_ReturnsIt", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		tmID := added[0].ID

		found, ok := team.FindTeamMember(tmID)
		if !ok {
			t.Fatal("expected to find team member, got not found")
		}
		if found.ID != tmID {
			t.Errorf("ID = %q, want %q", found.ID, tmID)
		}
		if found.MemberID != "spec-2" {
			t.Errorf("MemberID = %q, want %q", found.MemberID, "spec-2")
		}
		if found.Name != "Agent Two" {
			t.Errorf("Name = %q, want %q", found.Name, "Agent Two")
		}
	})

	t.Run("FindRoot_ReturnsIt", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		found, ok := team.FindTeamMember(team.RootTeamMemberID)
		if !ok {
			t.Fatal("expected to find root team member")
		}
		if found.MemberID != "spec-1" {
			t.Errorf("MemberID = %q, want %q", found.MemberID, "spec-1")
		}
	})

	t.Run("Nonexistent_ReturnsFalse", func(t *testing.T) {
		team, _ := NewTeam("team", "spec-1", "Agent One")
		_, ok := team.FindTeamMember("nonexistent-uuid")
		if ok {
			t.Fatal("expected not found, got found")
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("ValidName_ChangesName", func(t *testing.T) {
		team, _ := NewTeam("old", "spec-1", "Agent One")
		name := "new"
		if err := team.Update(&name, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.Name != "new" {
			t.Errorf("Name = %q, want %q", team.Name, "new")
		}
	})

	t.Run("ExistingTeamMember_ChangesRootTeamMemberID", func(t *testing.T) {
		team, added := createTeamWithMembers(t, member("spec-2", "Agent Two"))
		tmID := added[0].ID
		if err := team.Update(nil, &tmID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if team.RootTeamMemberID != tmID {
			t.Errorf("RootTeamMemberID = %q, want %q", team.RootTeamMemberID, tmID)
		}
	})

	t.Run("EmptyName_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("valid", "spec-1", "Agent One")
		name := ""
		if err := team.Update(&name, nil); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("NonexistentRootTeamMemberID_ReturnsError", func(t *testing.T) {
		team, _ := NewTeam("valid", "spec-1", "Agent One")
		id := "nonexistent"
		if err := team.Update(nil, &id); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestIsConnectedTo(t *testing.T) {
	rel := MemberRelations{
		Leaders: []string{"leader-1"},
		Workers: []string{"worker-1"},
	}

	t.Run("Connected_ReturnsTrue", func(t *testing.T) {
		for _, id := range []string{"leader-1", "worker-1"} {
			if !rel.IsConnectedTo(id) {
				t.Errorf("IsConnectedTo(%q) = false, want true", id)
			}
		}
	})

	t.Run("NotConnected_ReturnsFalse", func(t *testing.T) {
		if rel.IsConnectedTo("stranger") {
			t.Error("IsConnectedTo(stranger) = true, want false")
		}
	})

	t.Run("Empty_ReturnsFalse", func(t *testing.T) {
		empty := MemberRelations{}
		if empty.IsConnectedTo("anyone") {
			t.Error("IsConnectedTo(anyone) = true, want false")
		}
	})
}
