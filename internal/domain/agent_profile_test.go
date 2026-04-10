package domain

import "testing"

func TestProfileFor_DefaultIsClaude(t *testing.T) {
	for _, agentType := range []string{"claude", ""} {
		profile := ProfileFor(agentType)
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
	}
}
