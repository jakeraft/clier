package domain

import "testing"

func TestNewMember(t *testing.T) {
	m, err := NewMember("coder", "claude-sonnet-4-6", []string{"--dangerously-skip-permissions"},
		"claude-md-1", []string{"skill-1"}, "settings-1", "claude-json-1",
		[]string{"env-1"}, "repo-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "coder" {
		t.Errorf("name = %q, want %q", m.Name, "coder")
	}
	if m.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want %q", m.Model, "claude-sonnet-4-6")
	}
	if len(m.Args) != 1 || m.Args[0] != "--dangerously-skip-permissions" {
		t.Errorf("args = %v, want [--dangerously-skip-permissions]", m.Args)
	}
	if m.ClaudeMdID != "claude-md-1" {
		t.Errorf("claude_md_id = %q, want %q", m.ClaudeMdID, "claude-md-1")
	}
	if len(m.SkillIDs) != 1 || m.SkillIDs[0] != "skill-1" {
		t.Errorf("skill_ids = %v, want [skill-1]", m.SkillIDs)
	}
	if m.SettingsID != "settings-1" {
		t.Errorf("settings_id = %q, want %q", m.SettingsID, "settings-1")
	}
	if m.ClaudeJsonID != "claude-json-1" {
		t.Errorf("claude_json_id = %q, want %q", m.ClaudeJsonID, "claude-json-1")
	}
	if m.GitRepoID != "repo-1" {
		t.Errorf("git_repo_id = %q, want %q", m.GitRepoID, "repo-1")
	}
}

func TestNewMember_EmptyName(t *testing.T) {
	_, err := NewMember("", "model", nil, "", nil, "", "", nil, "")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewMember_EmptyModel(t *testing.T) {
	_, err := NewMember("name", "", nil, "", nil, "", "", nil, "")
	if err == nil {
		t.Error("expected error for empty model")
	}
}

func TestMember_NilSlicesDefault(t *testing.T) {
	m, err := NewMember("coder", "claude-sonnet-4-6", nil, "", nil, "", "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Args == nil {
		t.Error("Args should be empty slice, not nil")
	}
	if m.SkillIDs == nil {
		t.Error("SkillIDs should be empty slice, not nil")
	}
	if m.EnvIDs == nil {
		t.Error("EnvIDs should be empty slice, not nil")
	}
}

func TestMember_Update(t *testing.T) {
	m, _ := NewMember("old", "old-model", nil, "", nil, "", "", nil, "")
	newName := "new"
	newModel := "new-model"
	newArgs := []string{"--flag"}
	newMdID := "md-1"
	newSkills := []string{"s-1", "s-2"}
	newSettings := "set-1"
	newCJ := "cj-1"
	newEnvs := []string{"e-1"}
	newRepo := "r-1"
	if err := m.Update(&newName, &newModel, &newArgs, &newMdID, &newSkills, &newSettings, &newCJ, &newEnvs, &newRepo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "new" {
		t.Errorf("name = %q, want %q", m.Name, "new")
	}
	if m.Model != "new-model" {
		t.Errorf("model = %q", m.Model)
	}
	if len(m.Args) != 1 {
		t.Errorf("args = %v", m.Args)
	}
	if m.ClaudeMdID != "md-1" {
		t.Errorf("claude_md_id = %q", m.ClaudeMdID)
	}
	if len(m.SkillIDs) != 2 {
		t.Errorf("skill_ids = %v", m.SkillIDs)
	}
	if m.SettingsID != "set-1" {
		t.Errorf("settings_id = %q", m.SettingsID)
	}
	if m.ClaudeJsonID != "cj-1" {
		t.Errorf("claude_json_id = %q", m.ClaudeJsonID)
	}
	if m.GitRepoID != "r-1" {
		t.Errorf("git_repo_id = %q", m.GitRepoID)
	}
}
