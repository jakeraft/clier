package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

var testSnapshot = TeamSnapshot{
	TeamName:     "alpha",
	RootMemberID: "root-1",
	Members:      []MemberSnapshot{},
}

func TestSprint(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndStartsRunning", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			if _, err := uuid.Parse(s.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", s.ID)
			}
			if !strings.HasPrefix(s.Name, "alpha_") {
				t.Errorf("Name = %q, want prefix 'alpha_'", s.Name)
			}
			if s.State != SprintRunning {
				t.Errorf("State = %q, want %q", s.State, SprintRunning)
			}
			if s.Error != "" {
				t.Errorf("Error = %q, want empty", s.Error)
			}
		})
	})

	t.Run("Complete", func(t *testing.T) {
		t.Run("Running_TransitionsToCompleted", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			if err := s.Complete(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.State != SprintCompleted {
				t.Errorf("State = %q, want %q", s.State, SprintCompleted)
			}
		})

		t.Run("AlreadyCompleted_ReturnsError", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			_ = s.Complete()
			if err := s.Complete(); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("Errored_ReturnsError", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			_ = s.Fail("boom")
			if err := s.Complete(); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Fail", func(t *testing.T) {
		t.Run("Running_TransitionsToErroredWithMessage", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			if err := s.Fail("something broke"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.State != SprintErrored {
				t.Errorf("State = %q, want %q", s.State, SprintErrored)
			}
			if s.Error != "something broke" {
				t.Errorf("Error = %q, want %q", s.Error, "something broke")
			}
		})

		t.Run("AlreadyCompleted_ReturnsError", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			_ = s.Complete()
			if err := s.Fail("too late"); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("AlreadyErrored_ReturnsError", func(t *testing.T) {
			s := NewSprint(testSnapshot)
			_ = s.Fail("first")
			if err := s.Fail("second"); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
