package cmd

import (
	"os"
	"path/filepath"
	"testing"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestValidateDownloadedWorkspace_Member(t *testing.T) {
	base := t.TempDir()
	required := []string{
		filepath.Join(base, "CLAUDE.md"),
		filepath.Join(base, ".clier", "work-log-protocol.md"),
		filepath.Join(base, ".claude", "settings.local.json"),
	}
	for _, path := range required {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}

	meta := &appworkspace.Manifest{
		Kind: resourceKindMember,
		Workspace: &appworkspace.WorkspaceMetadata{
			Member: &appworkspace.MemberWorkspaceMetadata{
				ID:      1,
				Name:    "reviewer",
				Command: "codex",
			},
		},
	}
	if err := validateDownloadedWorkspace(base, meta); err != nil {
		t.Fatalf("validateDownloadedWorkspace: %v", err)
	}
}

func TestValidateDownloadedWorkspace_MissingFileFails(t *testing.T) {
	base := t.TempDir()
	meta := &appworkspace.Manifest{
		Kind: resourceKindMember,
		Workspace: &appworkspace.WorkspaceMetadata{
			Member: &appworkspace.MemberWorkspaceMetadata{
				ID:      1,
				Name:    "reviewer",
				Command: "codex",
			},
		},
	}
	if err := validateDownloadedWorkspace(base, meta); err == nil {
		t.Fatalf("expected validation error for incomplete workspace")
	}
}
