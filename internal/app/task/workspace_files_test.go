package task

import (
	"encoding/json"
	"testing"

	agentrt "github.com/jakeraft/clier/internal/adapter/runtime"
	"github.com/jakeraft/clier/internal/domain/resource"
)

func TestMergeJSON(t *testing.T) {
	t.Run("BothEmpty", func(t *testing.T) {
		got := mergeJSON("", "")
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("SystemOnly", func(t *testing.T) {
		got := mergeJSON(`{"a":1}`, "")
		if got != `{"a":1}` {
			t.Errorf("got %q", got)
		}
	})

	t.Run("UserOnly", func(t *testing.T) {
		got := mergeJSON("", `{"b":2}`)
		if got != `{"b":2}` {
			t.Errorf("got %q", got)
		}
	})

	t.Run("DisjointKeys", func(t *testing.T) {
		got := mergeJSON(`{"a":1}`, `{"b":2}`)
		var m map[string]int
		json.Unmarshal([]byte(got), &m)
		if m["a"] != 1 || m["b"] != 2 {
			t.Errorf("got %q, want both keys", got)
		}
	})

	t.Run("UserOverridesSystem", func(t *testing.T) {
		got := mergeJSON(`{"a":1}`, `{"a":99}`)
		var m map[string]int
		json.Unmarshal([]byte(got), &m)
		if m["a"] != 99 {
			t.Errorf("got %q, want a=99", got)
		}
	})

	t.Run("ProjectsDeepMerge", func(t *testing.T) {
		system := `{"hasCompletedOnboarding":true,"projects":{"/ws/project":{"hasTrustDialogAccepted":true}}}`
		user := `{"projects":{"/extra/path":{"custom":true}}}`
		got := mergeJSON(system, user)

		var m map[string]json.RawMessage
		json.Unmarshal([]byte(got), &m)

		var projects map[string]json.RawMessage
		json.Unmarshal(m["projects"], &projects)

		if _, ok := projects["/ws/project"]; !ok {
			t.Error("system projects entry should be preserved")
		}
		if _, ok := projects["/extra/path"]; !ok {
			t.Error("user projects entry should be added")
		}
	})

	t.Run("MalformedUser", func(t *testing.T) {
		got := mergeJSON(`{"a":1}`, "not json")
		if got != `{"a":1}` {
			t.Errorf("got %q, want system json preserved", got)
		}
	})
}

func TestBuildWorkspaceFiles(t *testing.T) {
	rt := &agentrt.ClaudeRuntime{}

	t.Run("AllEmpty", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "", "", "", "", "", nil)
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("OnlySystemAgentDotMd", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "# Protocol", "", "", "", "", nil)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0].Path != "/ws/project/CLAUDE.md" {
			t.Errorf("path = %q", files[0].Path)
		}
		if files[0].Content != "# Protocol" {
			t.Errorf("content = %q", files[0].Content)
		}
	})

	t.Run("SystemAndUserAgentDotMd", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "# Protocol", "# User Rules", "", "", "", nil)
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		want := "# Protocol\n\n---\n\n# User Rules"
		if files[0].Content != want {
			t.Errorf("content = %q, want %q", files[0].Content, want)
		}
	})

	t.Run("SettingsFile", func(t *testing.T) {
		files := buildWorkspaceFiles(rt, "/ws", "", "", `{"key":"val"}`, "", "", nil)
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
		files := buildWorkspaceFiles(rt, "/ws", "", "", "", "", "", skills)
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
