package sprint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildCommand(t *testing.T) {
	t.Run("Claude/IncludesAllArgs", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID:       "m1",
			Binary:         domain.BinaryClaude,
			Model:          "claude-sonnet-4-6",
			SystemArgs:     []string{"--dangerously-skip-permissions"},
			CustomArgs:     []string{"--verbose"},
			ComposedPrompt: "you are a coder",
		}
		cmd, tempFiles, err := BuildCommand(m, "/work")
		if err != nil {
			t.Fatalf("BuildCommand: %v", err)
		}
		if len(tempFiles) != 0 {
			t.Errorf("claude should have no temp files, got %v", tempFiles)
		}
		if !strings.Contains(cmd, "claude") {
			t.Errorf("command should contain binary: %s", cmd)
		}
		if !strings.Contains(cmd, "--model 'claude-sonnet-4-6'") {
			t.Errorf("command should contain model: %s", cmd)
		}
		if !strings.Contains(cmd, "--session-id 'm1'") {
			t.Errorf("command should contain session-id: %s", cmd)
		}
		if !strings.Contains(cmd, "--dangerously-skip-permissions") {
			t.Errorf("command should contain system args: %s", cmd)
		}
		if !strings.Contains(cmd, "--verbose") {
			t.Errorf("command should contain custom args: %s", cmd)
		}
		if !strings.Contains(cmd, "--append-system-prompt") {
			t.Errorf("command should contain prompt: %s", cmd)
		}
		if !strings.HasPrefix(cmd, "cd ") {
			t.Errorf("command should start with cd: %s", cmd)
		}
	})

	t.Run("Codex/WritesInstructionsFile", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID:       "m2",
			Binary:         domain.BinaryCodex,
			Model:          "gpt-5.4",
			SystemArgs:     []string{},
			CustomArgs:     []string{},
			ComposedPrompt: "you are a coder",
		}
		cmd, tempFiles, err := BuildCommand(m, "/work")
		if err != nil {
			t.Fatalf("BuildCommand: %v", err)
		}
		if len(tempFiles) != 1 {
			t.Fatalf("codex should have 1 temp file, got %d", len(tempFiles))
		}
		defer os.Remove(tempFiles[0])

		if !strings.Contains(cmd, "model_instructions_file=") {
			t.Errorf("command should contain instructions file: %s", cmd)
		}

		data, err := os.ReadFile(tempFiles[0])
		if err != nil {
			t.Fatalf("read instructions file: %v", err)
		}
		if string(data) != "you are a coder" {
			t.Errorf("instructions content = %q, want %q", string(data), "you are a coder")
		}
	})
}

func TestBuildEnv(t *testing.T) {
	t.Run("IncludesRequiredVars", func(t *testing.T) {
		m := domain.MemberSnapshot{
			MemberID: "m1",
			Environments: []domain.SnapshotEnvironment{
				{Key: "API_KEY", Value: "secret"},
			},
		}
		env := BuildEnv(m, "sprint-1", "/home/m1")

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			envMap[parts[0]] = parts[1]
		}

		if envMap["HOME"] != "/home/m1" {
			t.Errorf("HOME = %q, want /home/m1", envMap["HOME"])
		}
		if envMap["CLIER_SPRINT_ID"] != "sprint-1" {
			t.Errorf("CLIER_SPRINT_ID = %q, want sprint-1", envMap["CLIER_SPRINT_ID"])
		}
		if envMap["CLIER_MEMBER_ID"] != "m1" {
			t.Errorf("CLIER_MEMBER_ID = %q, want m1", envMap["CLIER_MEMBER_ID"])
		}
		if envMap["API_KEY"] != "secret" {
			t.Errorf("API_KEY = %q, want secret", envMap["API_KEY"])
		}
	})
}

func TestWriteConfigs(t *testing.T) {
	t.Run("Claude/WritesSettingsAndTrust", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary:    domain.BinaryClaude,
			DotConfig: domain.DotConfig{"skipDangerousModePermissionPrompt": true},
		}

		if err := WriteConfigs(m, home, workDir); err != nil {
			t.Fatalf("WriteConfigs: %v", err)
		}

		// Check settings.json
		data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		if err != nil {
			t.Fatalf("read settings.json: %v", err)
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("parse settings.json: %v", err)
		}
		if settings["skipDangerousModePermissionPrompt"] != true {
			t.Errorf("settings missing skipDangerousModePermissionPrompt")
		}

		// Check .claude.json
		data, err = os.ReadFile(filepath.Join(home, ".claude.json"))
		if err != nil {
			t.Fatalf("read .claude.json: %v", err)
		}
		var trust map[string]any
		if err := json.Unmarshal(data, &trust); err != nil {
			t.Fatalf("parse .claude.json: %v", err)
		}
		projects := trust["projects"].(map[string]any)
		if _, ok := projects[workDir]; !ok {
			t.Errorf(".claude.json missing project entry for %s", workDir)
		}
	})

	t.Run("Codex/WritesConfigToml", func(t *testing.T) {
		home := t.TempDir()
		workDir := filepath.Join(home, "project")

		m := domain.MemberSnapshot{
			Binary:    domain.BinaryCodex,
			DotConfig: domain.DotConfig{"sandbox_mode": "danger-full-access"},
		}

		if err := WriteConfigs(m, home, workDir); err != nil {
			t.Fatalf("WriteConfigs: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
		if err != nil {
			t.Fatalf("read config.toml: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "sandbox_mode") {
			t.Errorf("config.toml missing sandbox_mode: %s", content)
		}
		if !strings.Contains(content, "trust_level") {
			t.Errorf("config.toml missing trust_level: %s", content)
		}
	})
}
