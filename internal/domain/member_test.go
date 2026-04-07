package domain

import "testing"

func TestNewMember(t *testing.T) {
	m, err := NewMember("coder", "claude", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
		"claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
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
	if m.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want %q", m.Model, "claude-sonnet-4-6")
	}
	if len(m.Args) != 1 || m.Args[0] != "--dangerously-skip-permissions" {
		t.Errorf("args = %v, want [--dangerously-skip-permissions]", m.Args)
	}
	if m.AgentDotMdID != "claude-md-1" {
		t.Errorf("agent_dot_md_id = %q, want %q", m.AgentDotMdID, "claude-md-1")
	}
	if len(m.SkillIDs) != 1 || m.SkillIDs[0] != "skill-1" {
		t.Errorf("skill_ids = %v, want [skill-1]", m.SkillIDs)
	}
	if m.ClaudeSettingsID != "settings-1" {
		t.Errorf("claude_settings_id = %q, want %q", m.ClaudeSettingsID, "settings-1")
	}
	if m.ClaudeJsonID != "claude-json-1" {
		t.Errorf("claude_json_id = %q, want %q", m.ClaudeJsonID, "claude-json-1")
	}
	if m.GitRepoURL != "https://github.com/example/repo.git" {
		t.Errorf("git_repo_url = %q, want %q", m.GitRepoURL, "https://github.com/example/repo.git")
	}
}

func TestNewMember_EmptyName(t *testing.T) {
	_, err := NewMember("", "claude", "model", nil, "", nil, "", "", "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMember_EmptyModel(t *testing.T) {
	_, err := NewMember("name", "claude", "", nil, "", nil, "", "", "")
	if err == nil {
		t.Error("expected error for empty model")
	}
}

func TestNewMember_DefaultAgentType(t *testing.T) {
	m, err := NewMember("coder", "", "claude-sonnet-4-6", nil, "", nil, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.AgentType != "claude" {
		t.Errorf("agent_type = %q, want %q (default)", m.AgentType, "claude")
	}
}

func TestMember_NilSlicesDefault(t *testing.T) {
	m, err := NewMember("coder", "claude", "claude-sonnet-4-6", nil, "", nil, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Args == nil {
		t.Error("Args should be empty slice, not nil")
	}
	if m.SkillIDs == nil {
		t.Error("SkillIDs should be empty slice, not nil")
	}
}

func TestMember_Update(t *testing.T) {
	m, _ := NewMember("old", "claude", "old-model", nil, "", nil, "", "", "")
	newName := "new"
	newAgentType := "codex"
	newModel := "new-model"
	newArgs := []string{"--flag"}
	newMdID := "md-1"
	newSkills := []string{"s-1", "s-2"}
	newSettings := "set-1"
	newCJ := "cj-1"
	newRepo := "https://github.com/example/new.git"
	if err := m.Update(&newName, &newAgentType, &newModel, &newArgs, &newMdID, &newSkills, &newSettings, &newCJ, &newRepo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "new" {
		t.Errorf("name = %q, want %q", m.Name, "new")
	}
	if m.AgentType != "codex" {
		t.Errorf("agent_type = %q, want %q", m.AgentType, "codex")
	}
	if m.Model != "new-model" {
		t.Errorf("model = %q", m.Model)
	}
	if len(m.Args) != 1 {
		t.Errorf("args = %v", m.Args)
	}
	if m.AgentDotMdID != "md-1" {
		t.Errorf("agent_dot_md_id = %q", m.AgentDotMdID)
	}
	if len(m.SkillIDs) != 2 {
		t.Errorf("skill_ids = %v", m.SkillIDs)
	}
	if m.ClaudeSettingsID != "set-1" {
		t.Errorf("claude_settings_id = %q", m.ClaudeSettingsID)
	}
	if m.ClaudeJsonID != "cj-1" {
		t.Errorf("claude_json_id = %q", m.ClaudeJsonID)
	}
	if m.GitRepoURL != "https://github.com/example/new.git" {
		t.Errorf("git_repo_url = %q", m.GitRepoURL)
	}
}
