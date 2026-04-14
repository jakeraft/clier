package domain_test

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestNewRun(t *testing.T) {
	teamID := int64(456)
	memberID := int64(789)

	t.Run("valid run with team", func(t *testing.T) {
		run, err := domain.NewRun(123, "my-team-run-123", &teamID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.ID != 123 {
			t.Errorf("ID = %d, want %d", run.ID, 123)
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
		run, err := domain.NewRun(456, "member-run-456", nil, &memberID)
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

	t.Run("zero id", func(t *testing.T) {
		_, err := domain.NewRun(0, "name", &teamID, nil)
		if err == nil {
			t.Fatal("expected error for zero ID")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := domain.NewRun(123, "", &teamID, nil)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("nil team and member allowed", func(t *testing.T) {
		run, err := domain.NewRun(123, "name", nil, nil)
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
		run, _ := domain.NewRun(123, "my-team-run-123", &teamID, nil)
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

func int64Ptr(v int64) *int64 { return &v }

func TestMessage(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_SetsFields", func(t *testing.T) {
			from := int64Ptr(10)
			to := int64Ptr(20)
			m, err := domain.NewMessage(1, from, to, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// ID is zero — server assigns it.
			if m.RunID != 1 {
				t.Errorf("RunID = %d, want %d", m.RunID, 1)
			}
			if m.FromMemberID == nil || *m.FromMemberID != 10 {
				t.Errorf("FromMemberID = %v, want 10", m.FromMemberID)
			}
			if m.ToMemberID == nil || *m.ToMemberID != 20 {
				t.Errorf("ToMemberID = %v, want 20", m.ToMemberID)
			}
			if m.Content != "hello" {
				t.Errorf("Content = %q, want %q", m.Content, "hello")
			}
			if m.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("ZeroRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage(0, int64Ptr(10), int64Ptr(20), "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NilToMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage(1, int64Ptr(10), nil, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.ToMemberID != nil {
				t.Errorf("ToMemberID = %v, want nil", m.ToMemberID)
			}
		})

		t.Run("NilFromMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage(1, nil, int64Ptr(20), "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.FromMemberID != nil {
				t.Errorf("FromMemberID = %v, want nil", m.FromMemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage(1, int64Ptr(10), int64Ptr(20), "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestNote(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_SetsFields", func(t *testing.T) {
			tmID := int64Ptr(42)
			n, err := domain.NewNote(1, tmID, "work started")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// ID is zero — server assigns it.
			if n.RunID != 1 {
				t.Errorf("RunID = %d, want %d", n.RunID, 1)
			}
			if n.MemberID == nil || *n.MemberID != 42 {
				t.Errorf("MemberID = %v, want 42", n.MemberID)
			}
			if n.Content != "work started" {
				t.Errorf("Content = %q, want %q", n.Content, "work started")
			}
			if n.CreatedAt.IsZero() {
				t.Error("CreatedAt is zero")
			}
		})

		t.Run("ZeroRunID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote(0, int64Ptr(42), "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NilMemberID_Allowed", func(t *testing.T) {
			n, err := domain.NewNote(1, nil, "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n.MemberID != nil {
				t.Errorf("MemberID = %v, want nil", n.MemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote(1, int64Ptr(42), "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
