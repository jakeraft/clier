package runplan

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

func TestBuildCodexFiles(t *testing.T) {
	const ms = "{{CLIER_MEMBERSPACE}}"

	t.Run("WithDotConfig_ReturnsConfigToml", func(t *testing.T) {
		dotConfig := domain.DotConfig{"sandbox_mode": "danger-full-access"}
		workDir := ms + "/project"

		files, err := buildCodexFiles(dotConfig, workDir, ms)
		if err != nil {
			t.Fatalf("buildCodexFiles: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}

		wantPath := ms + "/.codex/config.toml"
		if files[0].Path != wantPath {
			t.Errorf("Path = %q, want %q", files[0].Path, wantPath)
		}
		if !strings.Contains(files[0].Content, "sandbox_mode") {
			t.Error("config.toml should contain sandbox_mode")
		}
		if !strings.Contains(files[0].Content, "trust_level") {
			t.Error("config.toml should contain trust_level")
		}
	})

	t.Run("NoAuthFile_OnlyConfigToml", func(t *testing.T) {
		files, err := buildCodexFiles(domain.DotConfig{}, ms+"/project", ms)
		if err != nil {
			t.Fatalf("buildCodexFiles: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
	})
}
