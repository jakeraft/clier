package domain

import "testing"

func TestProfileFor_DefaultIsClaude(t *testing.T) {
	for _, agentType := range []string{"claude", ""} {
		profile, err := ProfileFor(agentType)
		if err != nil {
			t.Fatalf("ProfileFor(%q) unexpected error: %v", agentType, err)
		}
		if profile.InstructionFile != "CLAUDE.md" {
			t.Errorf("ProfileFor(%q).InstructionFile = %q, want %q", agentType, profile.InstructionFile, "CLAUDE.md")
		}
		if profile.SettingsDir != ".claude" {
			t.Errorf("ProfileFor(%q).SettingsDir = %q, want %q", agentType, profile.SettingsDir, ".claude")
		}
		if profile.ReadyMarker != "Claude" {
			t.Errorf("ProfileFor(%q).ReadyMarker = %q, want %q", agentType, profile.ReadyMarker, "Claude")
		}
		if profile.ExitCommand != "/exit" {
			t.Errorf("ProfileFor(%q).ExitCommand = %q, want %q", agentType, profile.ExitCommand, "/exit")
		}
		if profile.InstructionKind != "claude-md" {
			t.Errorf("ProfileFor(%q).InstructionKind = %q, want %q", agentType, profile.InstructionKind, "claude-md")
		}
		if profile.SettingsKind != "claude-setting" {
			t.Errorf("ProfileFor(%q).SettingsKind = %q, want %q", agentType, profile.SettingsKind, "claude-setting")
		}
	}
}

func TestProfileFor_Codex(t *testing.T) {
	profile, err := ProfileFor("codex")
	if err != nil {
		t.Fatalf("ProfileFor(\"codex\") unexpected error: %v", err)
	}
	if profile.InstructionFile != "AGENTS.md" {
		t.Errorf("InstructionFile = %q, want %q", profile.InstructionFile, "AGENTS.md")
	}
	if profile.SettingsDir != ".codex" {
		t.Errorf("SettingsDir = %q, want %q", profile.SettingsDir, ".codex")
	}
	if profile.SettingsFile != "config.toml" {
		t.Errorf("SettingsFile = %q, want %q", profile.SettingsFile, "config.toml")
	}
	if profile.LocalSettingsFile != "" {
		t.Errorf("LocalSettingsFile = %q, want empty", profile.LocalSettingsFile)
	}
	if profile.SkillsDir != "skills" {
		t.Errorf("SkillsDir = %q, want %q", profile.SkillsDir, "skills")
	}
	if profile.ReadyMarker != "" {
		t.Errorf("ReadyMarker = %q, want empty", profile.ReadyMarker)
	}
	if profile.ExitCommand != "/exit" {
		t.Errorf("ExitCommand = %q, want %q", profile.ExitCommand, "/exit")
	}
	if profile.HomeExcludeKey != "" {
		t.Errorf("HomeExcludeKey = %q, want empty", profile.HomeExcludeKey)
	}
	if profile.InstructionKind != "codex-md" {
		t.Errorf("InstructionKind = %q, want %q", profile.InstructionKind, "codex-md")
	}
	if profile.SettingsKind != "codex-setting" {
		t.Errorf("SettingsKind = %q, want %q", profile.SettingsKind, "codex-setting")
	}
}

func TestProfileFor_UnknownType(t *testing.T) {
	_, err := ProfileFor("unknown-agent")
	if err == nil {
		t.Error("ProfileFor(\"unknown-agent\") expected error, got nil")
	}
}
