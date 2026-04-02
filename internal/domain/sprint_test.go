package domain

import (
	"strings"
	"testing"
)

func TestNewSprint(t *testing.T) {
	snapshot := SprintSnapshot{
		TeamName:     "alpha",
		RootMemberID: "root-1",
		Members:      []SprintMemberSnapshot{},
	}

	t.Run("ValidInputs_CreatesSprintWithName", func(t *testing.T) {
		id := "abcdef12-0000-0000-0000-000000000000"
		s, err := NewSprint(id, snapshot)
		if err != nil {
			t.Fatalf("NewSprint: %v", err)
		}
		if s.ID != id {
			t.Errorf("ID = %q, want %q", s.ID, id)
		}
		if !strings.HasPrefix(s.Name, "alpha_abcdef12") {
			t.Errorf("Name = %q, want prefix 'alpha_abcdef12'", s.Name)
		}
	})

	t.Run("EmptyID_ReturnsError", func(t *testing.T) {
		_, err := NewSprint("", snapshot)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyTeamName_ReturnsError", func(t *testing.T) {
		_, err := NewSprint("some-id", SprintSnapshot{TeamName: "", RootMemberID: "root-1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyRootMemberID_ReturnsError", func(t *testing.T) {
		_, err := NewSprint("some-id", SprintSnapshot{TeamName: "alpha", RootMemberID: ""})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
