package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestMessage(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			m, err := NewMessage("sprint-1", "from-1", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(m.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", m.ID)
			}
			if m.SprintID != "sprint-1" {
				t.Errorf("SprintID = %q, want %q", m.SprintID, "sprint-1")
			}
			if m.FromMemberID != "from-1" {
				t.Errorf("FromMemberID = %q, want %q", m.FromMemberID, "from-1")
			}
			if m.ToMemberID != "to-1" {
				t.Errorf("ToMemberID = %q, want %q", m.ToMemberID, "to-1")
			}
			if m.Content != "hello" {
				t.Errorf("Content = %q, want %q", m.Content, "hello")
			}
			if m.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := NewMessage("sprint-1", "from-1", "to-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
