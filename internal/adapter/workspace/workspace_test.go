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
		t.Run("ValidMembers_CreatesAllDirsAndFiles", func(t *testing.T) {
			baseDir := t.TempDir()
			ws := New(baseDir)

			memberspace := filepath.Join(baseDir, "plan-1", "m1")
			members := []domain.MemberPlan{
				{
					TeamMemberID: "m1",
					MemberName:   "alice",
					Workspace: domain.WorkspacePlan{
						Memberspace: memberspace,
						Files: []domain.FileEntry{
							{
								Path:    filepath.Join(memberspace, ".claude", "settings.json"),
								Content: `{"key": "value"}`,
							},
						},
					},
				},
			}

			err := ws.Prepare(context.Background(), members)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}

			// WorkDir (memberspace/project) created
			workDir := filepath.Join(memberspace, "project")
			if _, err := os.Stat(workDir); err != nil {
				t.Errorf("workdir not created: %v", err)
			}

			// File written (absolute path)
			data, err := os.ReadFile(filepath.Join(memberspace, ".claude", "settings.json"))
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			if string(data) != `{"key": "value"}` {
				t.Errorf("content = %q, want {\"key\": \"value\"}", string(data))
			}
		})

		t.Run("EmptyMembers_NoError", func(t *testing.T) {
			ws := New(t.TempDir())
			err := ws.Prepare(context.Background(), nil)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
		})
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Run("RemovesPlanDir", func(t *testing.T) {
			baseDir := t.TempDir()
			ws := New(baseDir)

			planDir := filepath.Join(baseDir, "plan-1", "member-1")
			if err := os.MkdirAll(planDir, 0755); err != nil {
				t.Fatalf("create dir: %v", err)
			}

			if err := ws.Cleanup("plan-1"); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			if _, err := os.Stat(filepath.Join(baseDir, "plan-1")); !os.IsNotExist(err) {
				t.Error("plan dir should be removed")
			}
		})
	})
}
