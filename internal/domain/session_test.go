package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

func TestNewSession(t *testing.T) {
	t.Run("valid session", func(t *testing.T) {
		session, err := domain.NewSession("session-123", "team-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.ID != "session-123" {
			t.Errorf("ID = %q, want %q", session.ID, "session-123")
		}
		if session.TeamID != "team-456" {
			t.Errorf("TeamID = %q, want %q", session.TeamID, "team-456")
		}
		if session.Status != domain.SessionRunning {
			t.Errorf("Status = %q, want %q", session.Status, domain.SessionRunning)
		}
		if session.StoppedAt != nil {
			t.Error("StoppedAt should be nil")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := domain.NewSession("", "team-456")
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty team id", func(t *testing.T) {
		_, err := domain.NewSession("session-123", "")
		if err == nil {
			t.Fatal("expected error for empty team ID")
		}
	})

	t.Run("stop", func(t *testing.T) {
		session, _ := domain.NewSession("session-123", "team-456")
		session.Stop()
		if session.Status != domain.SessionStopped {
			t.Errorf("Status = %q, want %q", session.Status, domain.SessionStopped)
		}
		if session.StoppedAt == nil {
			t.Error("StoppedAt should not be nil after stop")
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			m, err := domain.NewMessage("session-1", "from-1", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(m.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", m.ID)
			}
			if m.SessionID != "session-1" {
				t.Errorf("SessionID = %q, want %q", m.SessionID, "session-1")
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

		t.Run("EmptySessionID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("", "from-1", "to-1", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyToMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("session-1", "from-1", "  ", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyFromMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage("session-1", "", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.FromMemberID != "" {
				t.Errorf("FromMemberID = %q, want empty", m.FromMemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("session-1", "from-1", "to-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
