package domain_test

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestNewRun(t *testing.T) {
	teamID := int64(456)
	memberID := int64(789)

	t.Run("valid run with team", func(t *testing.T) {
		run, err := domain.NewRun("run-123", "my-team-run-123", &teamID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.ID != "run-123" {
			t.Errorf("ID = %q, want %q", run.ID, "run-123")
		}
		if run.Name != "my-team-run-123" {
			t.Errorf("Name = %q, want %q", run.Name, "my-team-run-123")
		}
		if run.TeamID == nil || *run.TeamID != 456 {
			t.Errorf("TeamID = %v, want 456", run.TeamID)
		}
		if run.MemberID != nil {
			t.Errorf("MemberID = %v, want nil", run.MemberID)
		}
		if run.Status != domain.RunRunning {
			t.Errorf("Status = %q, want %q", run.Status, domain.RunRunning)
		}
		if run.StoppedAt != nil {
			t.Error("StoppedAt should be nil")
		}
	})

	t.Run("valid run with member", func(t *testing.T) {
		run, err := domain.NewRun("run-456", "member-run-456", nil, &memberID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.TeamID != nil {
			t.Errorf("TeamID = %v, want nil", run.TeamID)
		}
		if run.MemberID == nil || *run.MemberID != 789 {
			t.Errorf("MemberID = %v, want 789", run.MemberID)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := domain.NewRun("", "name", &teamID, nil)
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := domain.NewRun("run-123", "", &teamID, nil)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("nil team and member allowed", func(t *testing.T) {
		run, err := domain.NewRun("run-123", "name", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.TeamID != nil {
			t.Errorf("TeamID = %v, want nil", run.TeamID)
		}
		if run.MemberID != nil {
			t.Errorf("MemberID = %v, want nil", run.MemberID)
		}
	})

	t.Run("stop", func(t *testing.T) {
		run, _ := domain.NewRun("run-123", "my-team-run-123", &teamID, nil)
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
		t.Run("ValidInputs_SetsFields", func(t *testing.T) {
			m, err := domain.NewMessage("run-1", 10, 20, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// ID is empty — server assigns it.
			if m.RunID != "run-1" {
				t.Errorf("RunID = %q, want %q", m.RunID, "run-1")
			}
			if m.FromTeamMemberID != 10 {
				t.Errorf("FromTeamMemberID = %d, want %d", m.FromTeamMemberID, 10)
			}
			if m.ToTeamMemberID != 20 {
				t.Errorf("ToTeamMemberID = %d, want %d", m.ToTeamMemberID, 20)
			}
			if m.Content != "hello" {
				t.Errorf("Content = %q, want %q", m.Content, "hello")
			}
			if m.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("EmptyRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("", 10, 20, "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("ZeroToTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("run-1", 10, 0, "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("ZeroFromTeamMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage("run-1", 0, 20, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.FromTeamMemberID != 0 {
				t.Errorf("FromTeamMemberID = %d, want 0", m.FromTeamMemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("run-1", 10, 20, "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestNote(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_SetsFields", func(t *testing.T) {
			n, err := domain.NewNote("run-1", 42, "work started")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// ID is empty — server assigns it.
			if n.RunID != "run-1" {
				t.Errorf("RunID = %q, want %q", n.RunID, "run-1")
			}
			if n.TeamMemberID != 42 {
				t.Errorf("TeamMemberID = %d, want %d", n.TeamMemberID, 42)
			}
			if n.Content != "work started" {
				t.Errorf("Content = %q, want %q", n.Content, "work started")
			}
			if n.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("EmptyRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("", 42, "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("ZeroTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("run-1", 0, "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("run-1", 42, "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
