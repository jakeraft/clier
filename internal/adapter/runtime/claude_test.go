package runtime

import (
	"strings"
	"testing"
)

func TestClaudeRuntime_Binary(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.Binary() != "claude" {
		t.Errorf("Binary() = %q, want %q", rt.Binary(), "claude")
	}
}

func TestClaudeRuntime_ConfigDirEnv(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.ConfigDirEnv("/ws")
	want := "CLAUDE_CONFIG_DIR=/ws/.claude"
	if got != want {
		t.Errorf("ConfigDirEnv() = %q, want %q", got, want)
	}
}

func TestClaudeRuntime_AuthEnvs(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.AuthEnvs("sk-token")
	if len(got) != 1 {
		t.Fatalf("expected 1 env, got %d", len(got))
	}
	if got[0] != "CLAUDE_CODE_OAUTH_TOKEN=sk-token" {
		t.Errorf("got %q", got[0])
	}
}

func TestClaudeRuntime_InstructionFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.InstructionFile() != "CLAUDE.md" {
		t.Errorf("InstructionFile() = %q", rt.InstructionFile())
	}
}

func TestClaudeRuntime_ConfigDir(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.ConfigDir() != ".claude" {
		t.Errorf("ConfigDir() = %q", rt.ConfigDir())
	}
}

func TestClaudeRuntime_SettingsFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.SettingsFile() != "settings.json" {
		t.Errorf("SettingsFile() = %q", rt.SettingsFile())
	}
}

func TestClaudeRuntime_ProjectConfigFile(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.ProjectConfigFile() != ".claude.json" {
		t.Errorf("ProjectConfigFile() = %q", rt.ProjectConfigFile())
	}
}

func TestClaudeRuntime_SkillsDir(t *testing.T) {
	rt := &ClaudeRuntime{}
	if rt.SkillsDir() != ".claude/skills" {
		t.Errorf("SkillsDir() = %q", rt.SkillsDir())
	}
}

func TestClaudeRuntime_SystemConfig(t *testing.T) {
	rt := &ClaudeRuntime{}
	got := rt.SystemConfig("/ws")
	if !strings.Contains(got, "hasCompletedOnboarding") {
		t.Errorf("SystemConfig missing expected content: %q", got)
	}
	if !strings.Contains(got, "/ws/project") {
		t.Errorf("SystemConfig missing workspace path: %q", got)
	}
}
