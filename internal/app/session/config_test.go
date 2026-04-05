package session

import (
	"strings"
	"testing"
)

func TestBuildClaudeFiles(t *testing.T) {
	const ms = "{{CLIER_MEMBERSPACE}}"

	t.Run("ReturnsSettingsAndClaudeJSON", func(t *testing.T) {
		settingsJSON := `{"skipDangerousModePermissionPrompt":true}`
		claudeJSON := `{"hasCompletedOnboarding":true,"projects":{"` + ms + `/project":{"hasTrustDialogAccepted":true}}}`

		files := buildClaudeFiles(settingsJSON, claudeJSON, ms)

		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}

		// settings.json
		if files[0].Path != ms+"/.claude/settings.json" {
			t.Errorf("Path = %q, want %q", files[0].Path, ms+"/.claude/settings.json")
		}
		if files[0].Content != settingsJSON {
			t.Errorf("Content = %q, want %q", files[0].Content, settingsJSON)
		}

		// .claude.json
		if files[1].Path != ms+"/.claude/.claude.json" {
			t.Errorf("Path = %q, want %q", files[1].Path, ms+"/.claude/.claude.json")
		}
		if files[1].Content != claudeJSON {
			t.Errorf("Content = %q, want %q", files[1].Content, claudeJSON)
		}
	})

	t.Run("TildePaths_PreservedInSettings", func(t *testing.T) {
		settingsJSON := `{"claudeMdExcludes":["~/.claude/**"]}`

		files := buildClaudeFiles(settingsJSON, `{}`, ms)

		if !strings.Contains(files[0].Content, "~/.claude/**") {
			t.Error("tilde paths should be preserved in settings")
		}
	})
}
