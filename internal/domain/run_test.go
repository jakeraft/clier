package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

func TestNewRun(t *testing.T) {
	t.Run("valid run", func(t *testing.T) {
		run, err := domain.NewRun("run-123", "my-team-run-123", "team-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.ID != "run-123" {
			t.Errorf("ID = %q, want %q", run.ID, "run-123")
		}
		if run.Name != "my-team-run-123" {
			t.Errorf("Name = %q, want %q", run.Name, "my-team-run-123")
		}
		if run.TeamID != "team-456" {
			t.Errorf("TeamID = %q, want %q", run.TeamID, "team-456")
		}
		if run.Status != domain.RunRunning {
			t.Errorf("Status = %q, want %q", run.Status, domain.RunRunning)
		}
		if run.StoppedAt != nil {
			t.Error("StoppedAt should be nil")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := domain.NewRun("", "name", "team-456")
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := domain.NewRun("run-123", "", "team-456")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("empty team id", func(t *testing.T) {
		_, err := domain.NewRun("run-123", "name", "")
		if err == nil {
			t.Fatal("expected error for empty team ID")
		}
	})

	t.Run("stop", func(t *testing.T) {
		run, _ := domain.NewRun("run-123", "my-team-run-123", "team-456")
		run.Stop()
		if run.Status != domain.RunStopped {
			t.Errorf("Status = %q, want %q", run.Status, domain.RunStopped)
		}
		if run.StoppedAt == nil {
			t.Error("StoppedAt should not be nil after stop")
		}
	})
}

func TestRunName(t *testing.T) {
	tests := []struct {
		teamName string
		runID    string
		want     string
	}{
		{"my-team", "abcdefgh-1234", "my-team-abcdefgh"},
		{"short", "abc", "short-abc"},
		{"dots.and:colons here", "12345678-9abc", "dots-and-colons-here-12345678"},
		{"a-very-long-team-name-that-exceeds", "id-12345", "a-very-long-team-nam-id-12345"},
	}
	for _, tt := range tests {
		got := domain.RunName(tt.teamName, tt.runID)
		if got != tt.want {
			t.Errorf("RunName(%q, %q) = %q, want %q", tt.teamName, tt.runID, got, tt.want)
		}
	}
}

func TestMessage(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			m, err := domain.NewMessage("run-1", "from-1", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(m.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", m.ID)
			}
			if m.RunID != "run-1" {
				t.Errorf("RunID = %q, want %q", m.RunID, "run-1")
			}
			if m.FromTeamMemberID != "from-1" {
				t.Errorf("FromTeamMemberID = %q, want %q", m.FromTeamMemberID, "from-1")
			}
			if m.ToTeamMemberID != "to-1" {
				t.Errorf("ToTeamMemberID = %q, want %q", m.ToTeamMemberID, "to-1")
			}
			if m.Content != "hello" {
				t.Errorf("Content = %q, want %q", m.Content, "hello")
			}
			if m.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("EmptyRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("", "from-1", "to-1", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyToTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("run-1", "from-1", "  ", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyFromTeamMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage("run-1", "", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.FromTeamMemberID != "" {
				t.Errorf("FromTeamMemberID = %q, want empty", m.FromTeamMemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("run-1", "from-1", "to-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestNote(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			n, err := domain.NewNote("run-1", "member-1", "work started")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(n.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", n.ID)
			}
			if n.RunID != "run-1" {
				t.Errorf("RunID = %q, want %q", n.RunID, "run-1")
			}
			if n.TeamMemberID != "member-1" {
				t.Errorf("TeamMemberID = %q, want %q", n.TeamMemberID, "member-1")
			}
			if n.Content != "work started" {
				t.Errorf("Content = %q, want %q", n.Content, "work started")
			}
			if n.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("EmptyRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("", "member-1", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("run-1", "", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("run-1", "member-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
