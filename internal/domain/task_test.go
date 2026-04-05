package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

func TestNewTask(t *testing.T) {
	t.Run("valid task", func(t *testing.T) {
		task, err := domain.NewTask("task-123", "my-team-task-123", "team-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.ID != "task-123" {
			t.Errorf("ID = %q, want %q", task.ID, "task-123")
		}
		if task.Name != "my-team-task-123" {
			t.Errorf("Name = %q, want %q", task.Name, "my-team-task-123")
		}
		if task.TeamID != "team-456" {
			t.Errorf("TeamID = %q, want %q", task.TeamID, "team-456")
		}
		if task.Status != domain.TaskRunning {
			t.Errorf("Status = %q, want %q", task.Status, domain.TaskRunning)
		}
		if task.StoppedAt != nil {
			t.Error("StoppedAt should be nil")
		}
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := domain.NewTask("", "name", "team-456")
		if err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := domain.NewTask("task-123", "", "team-456")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("empty team id", func(t *testing.T) {
		_, err := domain.NewTask("task-123", "name", "")
		if err == nil {
			t.Fatal("expected error for empty team ID")
		}
	})

	t.Run("stop", func(t *testing.T) {
		task, _ := domain.NewTask("task-123", "my-team-task-123", "team-456")
		task.Stop()
		if task.Status != domain.TaskStopped {
			t.Errorf("Status = %q, want %q", task.Status, domain.TaskStopped)
		}
		if task.StoppedAt == nil {
			t.Error("StoppedAt should not be nil after stop")
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			m, err := domain.NewMessage("task-1", "from-1", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(m.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", m.ID)
			}
			if m.TaskID != "task-1" {
				t.Errorf("TaskID = %q, want %q", m.TaskID, "task-1")
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

		t.Run("EmptyTaskID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("", "from-1", "to-1", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyToTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("task-1", "from-1", "  ", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyFromTeamMemberID_Allowed", func(t *testing.T) {
			m, err := domain.NewMessage("task-1", "", "to-1", "hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m.FromTeamMemberID != "" {
				t.Errorf("FromTeamMemberID = %q, want empty", m.FromTeamMemberID)
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewMessage("task-1", "from-1", "to-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}

func TestNote(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			n, err := domain.NewNote("task-1", "member-1", "work started")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(n.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", n.ID)
			}
			if n.TaskID != "task-1" {
				t.Errorf("TaskID = %q, want %q", n.TaskID, "task-1")
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

		t.Run("EmptyTaskID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("", "member-1", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyTeamMemberID_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("task-1", "", "hello")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyContent_ReturnsError", func(t *testing.T) {
			_, err := domain.NewNote("task-1", "member-1", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
