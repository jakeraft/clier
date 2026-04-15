package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/adapter/api"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func TestValidateWorkingCopy_Member(t *testing.T) {
	memberName := "reviewer"
	base := t.TempDir()
	memberBase := filepath.Join(base, memberName)
	required := []string{
		filepath.Join(memberBase, "CLAUDE.md"),
		filepath.Join(memberBase, ".clier", "work-log-protocol.md"),
		filepath.Join(memberBase, ".claude", "settings.local.json"),
		filepath.Join(memberBase, ".clier", appworkspace.TeamProtocolFileName(memberName)),
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
		Kind: string(api.KindMember),
		Runtime: &appworkspace.RuntimeMetadata{
			Team: &appworkspace.TeamRuntimeMetadata{
				ID:   0,
				Name: memberName,
				Members: []appworkspace.TeamMemberRuntimeMetadata{{
					MemberID: 1,
					Name:     memberName,
					Command:  "codex",
				}},
			},
		},
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy: %v", err)
	}
}

func TestValidateWorkingCopy_CodexMember(t *testing.T) {
	memberName := "coder"
	base := t.TempDir()
	memberBase := filepath.Join(base, memberName)
	required := []string{
		filepath.Join(memberBase, "AGENTS.md"),
		filepath.Join(memberBase, ".clier", "work-log-protocol.md"),
		filepath.Join(memberBase, ".clier", appworkspace.TeamProtocolFileName(memberName)),
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
		Kind: string(api.KindMember),
		Runtime: &appworkspace.RuntimeMetadata{
			Team: &appworkspace.TeamRuntimeMetadata{
				ID:   0,
				Name: memberName,
				Members: []appworkspace.TeamMemberRuntimeMetadata{{
					MemberID:  1,
					Name:      memberName,
					AgentType: "codex",
					Command:   "codex",
				}},
			},
		},
	}
	if err := validateWorkingCopy(base, meta); err != nil {
		t.Fatalf("validateWorkingCopy (codex): %v", err)
	}
}

func TestValidateWorkingCopy_MissingFileFails(t *testing.T) {
	base := t.TempDir()
	meta := &appworkspace.Manifest{
		Kind: string(api.KindMember),
		Runtime: &appworkspace.RuntimeMetadata{
			Team: &appworkspace.TeamRuntimeMetadata{
				ID:   0,
				Name: "reviewer",
				Members: []appworkspace.TeamMemberRuntimeMetadata{{
					MemberID: 1,
					Name:     "reviewer",
					Command:  "codex",
				}},
			},
		},
	}
	if err := validateWorkingCopy(base, meta); err == nil {
		t.Fatalf("expected validation error for incomplete local clone")
	}
}
