package domain

import (
	"strings"
	"testing"
)

func TestNewSprint(t *testing.T) {
	teamSnap := TeamSnapshot{
		TeamName:     "alpha",
		RootMemberID: "root-1",
		Members:      []TeamMemberSnapshot{},
	}
	snap := SprintSnapshot{
		Members: []SprintMemberSnapshot{},
	}

	t.Run("ValidInputs_CreatesSprintWithBothSnapshots", func(t *testing.T) {
		id := "abcdef12-0000-0000-0000-000000000000"
		s, err := NewSprint(id, teamSnap, snap)
		if err != nil {
			t.Fatalf("NewSprint: %v", err)
		}
		if s.ID != id {
			t.Errorf("ID = %q, want %q", s.ID, id)
		}
		if !strings.HasPrefix(s.Name, "alpha_abcdef12") {
			t.Errorf("Name = %q, want prefix 'alpha_abcdef12'", s.Name)
		}
		if s.TeamSnapshot.TeamName != "alpha" {
			t.Errorf("TeamSnapshot.TeamName = %q, want alpha", s.TeamSnapshot.TeamName)
		}
	})

	t.Run("EmptyID_ReturnsError", func(t *testing.T) {
		_, err := NewSprint("", teamSnap, snap)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyTeamName_ReturnsError", func(t *testing.T) {
		_, err := NewSprint("some-id", TeamSnapshot{}, snap)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
