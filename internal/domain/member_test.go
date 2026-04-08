package domain

import "testing"

func int64Ptr(v int64) *int64 { return &v }

func TestNewMember(t *testing.T) {
	claudeMdID := int64Ptr(1)
	settingsID := int64Ptr(2)
	m, err := NewMember("coder", "claude", "claude --dangerously-skip-permissions",
		claudeMdID, []int64{10}, settingsID,
		"https://github.com/example/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "coder" {
		t.Errorf("name = %q, want %q", m.Name, "coder")
	}
	if m.AgentType != "claude" {
		t.Errorf("agent_type = %q, want %q", m.AgentType, "claude")
	}
	if m.Command != "claude --dangerously-skip-permissions" {
		t.Errorf("command = %q, want %q", m.Command, "claude --dangerously-skip-permissions")
	}
	if m.ClaudeMdID == nil || *m.ClaudeMdID != 1 {
		t.Errorf("claude_md_id = %v, want 1", m.ClaudeMdID)
	}
	if len(m.SkillIDs) != 1 || m.SkillIDs[0] != 10 {
		t.Errorf("skill_ids = %v, want [10]", m.SkillIDs)
	}
	if m.ClaudeSettingsID == nil || *m.ClaudeSettingsID != 2 {
		t.Errorf("claude_settings_id = %v, want 2", m.ClaudeSettingsID)
	}
	if m.GitRepoURL != "https://github.com/example/repo.git" {
		t.Errorf("git_repo_url = %q, want %q", m.GitRepoURL, "https://github.com/example/repo.git")
	}
}

func TestNewMember_EmptyName(t *testing.T) {
	_, err := NewMember("", "", "claude", nil, nil, nil, "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMember_EmptyCommand(t *testing.T) {
	_, err := NewMember("name", "", "", nil, nil, nil, "")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestMember_NilSlicesDefault(t *testing.T) {
	m, err := NewMember("coder", "", "claude", nil, nil, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SkillIDs == nil {
		t.Error("SkillIDs should be empty slice, not nil")
	}
}

func TestMember_Update(t *testing.T) {
	m, _ := NewMember("old", "", "claude", nil, nil, nil, "")
	newName := "new"
	newAgentType := "codex"
	newCommand := "codex --flag"
	newMdID := int64Ptr(1)
	newSkills := []int64{10, 20}
	newSettings := int64Ptr(3)
	newRepo := "https://github.com/example/new.git"
	if err := m.Update(&newName, &newAgentType, &newCommand, &newMdID, &newSkills, &newSettings, &newRepo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "new" {
		t.Errorf("name = %q, want %q", m.Name, "new")
	}
	if m.AgentType != "codex" {
		t.Errorf("agent_type = %q, want %q", m.AgentType, "codex")
	}
	if m.Command != "codex --flag" {
		t.Errorf("command = %q, want %q", m.Command, "codex --flag")
	}
	if m.ClaudeMdID == nil || *m.ClaudeMdID != 1 {
		t.Errorf("claude_md_id = %v, want 1", m.ClaudeMdID)
	}
	if len(m.SkillIDs) != 2 {
		t.Errorf("skill_ids = %v", m.SkillIDs)
	}
	if m.ClaudeSettingsID == nil || *m.ClaudeSettingsID != 3 {
		t.Errorf("claude_settings_id = %v, want 3", m.ClaudeSettingsID)
	}
	if m.GitRepoURL != "https://github.com/example/new.git" {
		t.Errorf("git_repo_url = %q", m.GitRepoURL)
	}
}
