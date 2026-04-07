package resource

import "testing"

func TestNewClaudeSettings(t *testing.T) {
	s, err := NewClaudeSettings("skip-permissions", `{"skipDangerousModePermissionPrompt":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "skip-permissions" {
		t.Errorf("name = %q, want %q", s.Name, "skip-permissions")
	}
	if s.Content != `{"skipDangerousModePermissionPrompt":true}` {
		t.Errorf("content mismatch")
	}
}

func TestNewClaudeSettings_EmptyName(t *testing.T) {
	_, err := NewClaudeSettings("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeSettings_EmptyContent(t *testing.T) {
	_, err := NewClaudeSettings("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestNewClaudeSettings_InvalidJSON(t *testing.T) {
	_, err := NewClaudeSettings("name", "not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestClaudeSettings_Update(t *testing.T) {
	s, _ := NewClaudeSettings("old", `{"old":true}`)
	newName := "new"
	newContent := `{"new":true}`
	if err := s.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "new" || s.Content != `{"new":true}` {
		t.Error("update did not apply")
	}
}
