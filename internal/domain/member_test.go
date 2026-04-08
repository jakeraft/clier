package domain

import "testing"

func TestNewMember(t *testing.T) {
	m, err := NewMember("coder", "claude --dangerously-skip-permissions",
		"claude-md-1", []string{"skill-1"}, "settings-1",
		"https://github.com/example/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "coder" {
		t.Errorf("name = %q, want %q", m.Name, "coder")
	}
	if m.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("command = %q, want %q", m.Command, "claude --dangerously-skip-permissions")
	}
	if m.ClaudeMdID != "claude-md-1" {
		t.Errorf("claude_md_id = %q, want %q", m.ClaudeMdID, "claude-md-1")
	}
	if len(m.SkillIDs) != 1 || m.SkillIDs[0] != "skill-1" {
		t.Errorf("skill_ids = %v, want [skill-1]", m.SkillIDs)
	}
	if m.ClaudeSettingsID != "settings-1" {
		t.Errorf("claude_settings_id = %q, want %q", m.ClaudeSettingsID, "settings-1")
	}
	if m.GitRepoURL != "https://github.com/example/repo.git" {
		t.Errorf("git_repo_url = %q, want %q", m.GitRepoURL, "https://github.com/example/repo.git")
	}
}

func TestNewMember_EmptyName(t *testing.T) {
	_, err := NewMember("", "claude", "", nil, "", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMember_EmptyCommand(t *testing.T) {
	_, err := NewMember("name", "", "", nil, "", "")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestMember_NilSlicesDefault(t *testing.T) {
	m, err := NewMember("coder", "claude", "", nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SkillIDs == nil {
		t.Error("SkillIDs should be empty slice, not nil")
	}
}

func TestMember_Update(t *testing.T) {
	m, _ := NewMember("old", "claude", "", nil, "", "")
	newName := "new"
	newCommand := "codex --flag"
	newMdID := "md-1"
	newSkills := []string{"s-1", "s-2"}
	newSettings := "set-1"
	newRepo := "https://github.com/example/new.git"
	if err := m.Update(&newName, &newCommand, &newMdID, &newSkills, &newSettings, &newRepo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "new" {
		t.Errorf("name = %q, want %q", m.Name, "new")
	}
	if m.Command != "codex --flag" {
		t.Errorf("command = %q, want %q", m.Command, "codex --flag")
	}
	if m.ClaudeMdID != "md-1" {
		t.Errorf("claude_md_id = %q", m.ClaudeMdID)
	}
	if len(m.SkillIDs) != 2 {
		t.Errorf("skill_ids = %v", m.SkillIDs)
	}
	if m.ClaudeSettingsID != "set-1" {
		t.Errorf("claude_settings_id = %q", m.ClaudeSettingsID)
	}
	if m.GitRepoURL != "https://github.com/example/new.git" {
		t.Errorf("git_repo_url = %q", m.GitRepoURL)
	}
}
