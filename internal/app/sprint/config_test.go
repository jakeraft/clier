package sprint

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildClaudeFiles(t *testing.T) {
	t.Run("WithDotConfig_ReturnsSettingsAndTrust", func(t *testing.T) {
		dotConfig := domain.DotConfig{"skipDangerousModePermissionPrompt": true}
		workDir := "/base/sprint-1/m1/project"

		files := buildClaudeFiles(dotConfig, workDir)

		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}

		// settings.json
		if files[0].Path != ".claude/settings.json" {
			t.Errorf("Path = %q, want .claude/settings.json", files[0].Path)
		}
		var settings map[string]any
		if err := json.Unmarshal([]byte(files[0].Content), &settings); err != nil {
			t.Fatalf("parse settings: %v", err)
		}
		if settings["skipDangerousModePermissionPrompt"] != true {
			t.Error("missing skipDangerousModePermissionPrompt")
		}

		// .claude.json (trust)
		if files[1].Path != ".claude/.claude.json" {
			t.Errorf("Path = %q, want .claude/.claude.json", files[1].Path)
		}
		if !strings.Contains(files[1].Content, workDir) {
			t.Error("trust config should contain workDir")
		}
	})

	t.Run("TildePaths_Expanded", func(t *testing.T) {
		dotConfig := domain.DotConfig{"claudeMdExcludes": []string{"~/.claude/**"}}
		files := buildClaudeFiles(dotConfig, "/work")

		if strings.Contains(files[0].Content, "~/") {
			t.Error("tilde should be expanded")
		}
	})
}

func TestBuildCodexFiles(t *testing.T) {
	t.Run("WithDotConfig_ReturnsConfigToml", func(t *testing.T) {
		dotConfig := domain.DotConfig{"sandbox_mode": "danger-full-access"}
		workDir := "/base/sprint-1/m2/project"

		files := buildCodexFiles(dotConfig, workDir)

		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0].Path != ".codex/config.toml" {
			t.Errorf("Path = %q, want .codex/config.toml", files[0].Path)
		}
		if !strings.Contains(files[0].Content, "sandbox_mode") {
			t.Error("config.toml should contain sandbox_mode")
		}
		if !strings.Contains(files[0].Content, "trust_level") {
			t.Error("config.toml should contain trust_level")
		}
	})
}
