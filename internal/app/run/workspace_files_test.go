package run

import (
	"testing"

	agentrt "github.com/jakeraft/clier/internal/adapter/runtime"
	"github.com/jakeraft/clier/internal/domain/resource"
)

func TestBuildWorkspaceFiles(t *testing.T) {
	rt := &agentrt.ClaudeRuntime{}

	t.Run("AllEmpty", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "", "", "", "", nil)
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("OnlySystemClaudeMd", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "# Protocol", "", "", "", nil)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0].Path != "/ws/CLAUDE.md" {
			t.Errorf("path = %q", files[0].Path)
		}
		if files[0].Content != "# Protocol" {
			t.Errorf("content = %q", files[0].Content)
		}
	})

	t.Run("SystemAndUserClaudeMd_SeparateFiles", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "# Protocol", "# User Rules", "", "", nil)
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		// System protocol in parent directory
		if files[0].Path != "/ws/CLAUDE.md" {
			t.Errorf("system path = %q, want /ws/CLAUDE.md", files[0].Path)
		}
		if files[0].Content != "# Protocol" {
			t.Errorf("system content = %q", files[0].Content)
		}
		// User instructions in project directory
		if files[1].Path != "/ws/project/CLAUDE.md" {
			t.Errorf("user path = %q, want /ws/project/CLAUDE.md", files[1].Path)
		}
		if files[1].Content != "# User Rules" {
			t.Errorf("user content = %q", files[1].Content)
		}
	})

	t.Run("SettingsFile", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "", "", `{"key":"val"}`, "", nil)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0].Path != "/ws/.claude/settings.json" {
			t.Errorf("path = %q", files[0].Path)
		}
	})

	t.Run("SkillFiles", func(t *testing.T) {
		skills := []resource.Skill{
			{Name: "code-review", Content: "Review code"},
			{Name: "tdd", Content: "Test first"},
		}
		files := buildWorkspaceFiles(rt, "/ws", "", "", "", "", skills)
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		if files[0].Path != "/ws/.claude/skills/code-review/SKILL.md" {
			t.Errorf("path = %q", files[0].Path)
		}
		if files[1].Path != "/ws/.claude/skills/tdd/SKILL.md" {
			t.Errorf("path = %q", files[1].Path)
		}
	})
}
