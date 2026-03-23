package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const rootID = "root-member-id"

func createTeamWithMembers(t *testing.T, extraMembers ...string) *Team {
	t.Helper()
	team, err := NewTeam("my-team", rootID)
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	for _, id := range extraMembers {
		if err := team.AddMember(id); err != nil {
			t.Fatalf("AddMember(%q): %v", id, err)
		}
	}
	return team
}

func TestTeam(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndAutoIncludesRoot", func(t *testing.T) {
			team, err := NewTeam("my-team", rootID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(team.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", team.ID)
			}
			if team.Name != "my-team" {
				t.Errorf("Name = %q, want %q", team.Name, "my-team")
			}
			if team.RootMemberID != rootID {
				t.Errorf("RootMemberID = %q, want %q", team.RootMemberID, rootID)
			}
			if len(team.MemberIDs) != 1 || team.MemberIDs[0] != rootID {
				t.Errorf("MemberIDs = %v, want [%s]", team.MemberIDs, rootID)
			}
			if len(team.Relations) != 0 {
				t.Errorf("Relations = %v, want []", team.Relations)
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewTeam("  ", rootID)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidName_ChangesName", func(t *testing.T) {
			team, _ := NewTeam("old", rootID)
			name := "new"
			if err := team.Update(&name, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if team.Name != "new" {
				t.Errorf("Name = %q, want %q", team.Name, "new")
			}
		})

		t.Run("ExistingMember_ChangesRootMemberID", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			memberID := "member-2"
			if err := team.Update(nil, &memberID); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if team.RootMemberID != "member-2" {
				t.Errorf("RootMemberID = %q, want %q", team.RootMemberID, "member-2")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("valid", rootID)
			name := ""
			if err := team.Update(&name, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NonexistentRootMemberID_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("valid", rootID)
			memberID := "nonexistent"
			if err := team.Update(nil, &memberID); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("AddMember", func(t *testing.T) {
		t.Run("NewMember_AddsMemberToTeam", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if err := team.AddMember("member-2"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Contains(team.MemberIDs, "member-2") {
				t.Errorf("MemberIDs %v does not contain member-2", team.MemberIDs)
			}
		})

		t.Run("DuplicateMember_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if err := team.AddMember(rootID); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("RemoveMember", func(t *testing.T) {
		t.Run("ExistingMember_RemovesMemberAndItsRelations", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if err := team.RemoveMember("member-2"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if slices.Contains(team.MemberIDs, "member-2") {
				t.Error("MemberIDs still contains member-2")
			}
			if len(team.Relations) != 0 {
				t.Errorf("Relations = %v, want []", team.Relations)
			}
		})

		t.Run("RootMember_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if err := team.RemoveMember(rootID); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NonexistentMember_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if err := team.RemoveMember("nonexistent"); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("AddRelation", func(t *testing.T) {
		t.Run("ValidLeader_AddsRelation", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			err := team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(team.Relations) != 1 {
				t.Fatalf("Relations length = %d, want 1", len(team.Relations))
			}
			r := team.Relations[0]
			if r.From != rootID || r.To != "member-2" || r.Type != RelationLeader {
				t.Errorf("Relation = %+v, want {%s %s leader}", r, rootID, "member-2")
			}
		})

		t.Run("ValidPeer_AddsRelation", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			err := team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationPeer})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if team.Relations[0].Type != RelationPeer {
				t.Errorf("Type = %q, want %q", team.Relations[0].Type, RelationPeer)
			}
		})

		t.Run("SelfRelation_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			err := team.AddRelation(Relation{From: rootID, To: rootID, Type: RelationPeer})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NonMemberFrom_ReturnsError", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			err := team.AddRelation(Relation{From: "nonexistent", To: "member-2", Type: RelationLeader})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NonMemberTo_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			err := team.AddRelation(Relation{From: rootID, To: "nonexistent", Type: RelationLeader})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("Duplicate_ReturnsError", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if err := team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader}); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("ReversePeerDuplicate_ReturnsError", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationPeer})
			if err := team.AddRelation(Relation{From: "member-2", To: rootID, Type: RelationPeer}); err == nil {
				t.Fatal("expected error for reverse peer duplicate, got nil")
			}
		})

		t.Run("LeaderUniqueness_SecondLeader_ReturnsError", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2", "member-3")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if err := team.AddRelation(Relation{From: "member-3", To: "member-2", Type: RelationLeader}); err == nil {
				t.Fatal("expected error for second leader, got nil")
			}
		})
	})

	t.Run("RemoveRelation", func(t *testing.T) {
		t.Run("ExistingRelation_RemovesIt", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if err := team.RemoveRelation(rootID, "member-2", RelationLeader); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(team.Relations) != 0 {
				t.Errorf("Relations = %v, want []", team.Relations)
			}
		})

		t.Run("Nonexistent_ReturnsError", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if err := team.RemoveRelation(rootID, "x", RelationLeader); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("GetMemberRelations", func(t *testing.T) {
		t.Run("MultipleRelations_ReturnsLeadersWorkersPeers", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2", "member-3", "member-4")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			_ = team.AddRelation(Relation{From: "member-3", To: rootID, Type: RelationLeader})
			_ = team.AddRelation(Relation{From: rootID, To: "member-4", Type: RelationPeer})

			rel := team.GetMemberRelations(rootID)
			if len(rel.Workers) != 1 || rel.Workers[0] != "member-2" {
				t.Errorf("Workers = %v, want [member-2]", rel.Workers)
			}
			if len(rel.Leaders) != 1 || rel.Leaders[0] != "member-3" {
				t.Errorf("Leaders = %v, want [member-3]", rel.Leaders)
			}
			if len(rel.Peers) != 1 || rel.Peers[0] != "member-4" {
				t.Errorf("Peers = %v, want [member-4]", rel.Peers)
			}
		})

		t.Run("NoRelations_ReturnsNilSlices", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			rel := team.GetMemberRelations("member-2")
			if rel.Leaders != nil {
				t.Errorf("Leaders = %v, want nil", rel.Leaders)
			}
			if rel.Workers != nil {
				t.Errorf("Workers = %v, want nil", rel.Workers)
			}
			if rel.Peers != nil {
				t.Errorf("Peers = %v, want nil", rel.Peers)
			}
		})
	})

	t.Run("GetDisconnectedWarnings", func(t *testing.T) {
		t.Run("DisconnectedMember_ReturnsWarning", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2", "member-3")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			warnings := team.GetDisconnectedWarnings()
			if len(warnings) != 1 {
				t.Fatalf("warnings length = %d, want 1", len(warnings))
			}
			if !strings.HasPrefix(warnings[0], "Member member-3") {
				t.Errorf("warning = %q, want prefix 'Member member-3'", warnings[0])
			}
		})

		t.Run("AllConnected_ReturnsEmpty", func(t *testing.T) {
			team := createTeamWithMembers(t, "member-2")
			_ = team.AddRelation(Relation{From: rootID, To: "member-2", Type: RelationLeader})
			if warnings := team.GetDisconnectedWarnings(); len(warnings) != 0 {
				t.Errorf("warnings = %v, want []", warnings)
			}
		})

		t.Run("OnlyRoot_ReturnsEmpty", func(t *testing.T) {
			team, _ := NewTeam("team", rootID)
			if warnings := team.GetDisconnectedWarnings(); len(warnings) != 0 {
				t.Errorf("warnings = %v, want []", warnings)
			}
		})
	})
}
