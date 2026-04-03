package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestWorkspace(t *testing.T) {
	t.Run("Prepare", func(t *testing.T) {
		t.Run("ValidSnapshot_CreatesAllDirsAndFiles", func(t *testing.T) {
			baseDir := t.TempDir()
			ws := New(baseDir)

			sprintDir := filepath.Join(baseDir, "sprint-1")
			snapshot := domain.SprintSnapshot{
				Members: []domain.SprintMemberSnapshot{
					{
						MemberID:   "m1",
						MemberName: "alice",
						Home:       filepath.Join(sprintDir, "m1"),
						WorkDir:    filepath.Join(sprintDir, "m1", "project"),
						Files: []domain.FileEntry{
							{Path: ".claude/settings.json", Content: `{"key": "value"}`},
						},
					},
				},
			}

			err := ws.Prepare(context.Background(), "sprint-1", snapshot)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}

			// WorkDir created
			if _, err := os.Stat(snapshot.Members[0].WorkDir); err != nil {
				t.Errorf("workdir not created: %v", err)
			}

			// File written
			data, err := os.ReadFile(filepath.Join(sprintDir, "m1", ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			if string(data) != `{"key": "value"}` {
				t.Errorf("content = %q, want {\"key\": \"value\"}", string(data))
			}
		})

		t.Run("EmptyMembers_NoError", func(t *testing.T) {
			ws := New(t.TempDir())
			err := ws.Prepare(context.Background(), "sprint-1", domain.SprintSnapshot{})
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Run("RemovesSprintDir", func(t *testing.T) {
			baseDir := t.TempDir()
			ws := New(baseDir)

			sprintDir := filepath.Join(baseDir, "sprint-1", "member-1")
			if err := os.MkdirAll(sprintDir, 0755); err != nil {
				t.Fatalf("create dir: %v", err)
			}

			if err := ws.Cleanup("sprint-1"); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			if _, err := os.Stat(filepath.Join(baseDir, "sprint-1")); !os.IsNotExist(err) {
				t.Error("sprint dir should be removed")
			}
		})
	})
}
