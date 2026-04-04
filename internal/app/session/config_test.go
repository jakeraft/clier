package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildClaudeFiles(t *testing.T) {
	const ms = "{{CLIER_MEMBERSPACE}}"

	t.Run("WithDotConfig_ReturnsSettingsAndTrust", func(t *testing.T) {
		dotConfig := domain.DotConfig{"skipDangerousModePermissionPrompt": true}
		workDir := ms + "/project"

		files, err := buildClaudeFiles(dotConfig, workDir, ms)
		if err != nil {
			t.Fatalf("buildClaudeFiles: %v", err)
		}

		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}

		// settings.json — path uses memberspace placeholder
		wantSettingsPath := ms + "/.claude/settings.json"
		if files[0].Path != wantSettingsPath {
			t.Errorf("Path = %q, want %q", files[0].Path, wantSettingsPath)
		}
		var settings map[string]any
		if err := json.Unmarshal([]byte(files[0].Content), &settings); err != nil {
			t.Fatalf("parse settings: %v", err)
		}
		if settings["skipDangerousModePermissionPrompt"] != true {
			t.Error("missing skipDangerousModePermissionPrompt")
		}

		// trust — path uses memberspace placeholder
		wantTrustPath := ms + "/.claude/.claude.json"
		if files[1].Path != wantTrustPath {
			t.Errorf("Path = %q, want %q", files[1].Path, wantTrustPath)
		}
		if !strings.Contains(files[1].Content, workDir) {
			t.Error("trust config should contain workDir")
		}
	})

	t.Run("TildePaths_PreservedInPlan", func(t *testing.T) {
		dotConfig := domain.DotConfig{"claudeMdExcludes": []string{"~/.claude/**"}}

		files, err := buildClaudeFiles(dotConfig, ms+"/project", ms)
		if err != nil {
			t.Fatalf("buildClaudeFiles: %v", err)
		}

		if !strings.Contains(files[0].Content, "~/.claude/**") {
			t.Error("tilde paths should be preserved in plan")
		}
	})
}
