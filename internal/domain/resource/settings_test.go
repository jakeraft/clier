package resource

import "testing"

func TestNewSettings(t *testing.T) {
	s, err := NewSettings("skip-permissions", `{"skipDangerousModePermissionPrompt":true}`)
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

func TestNewSettings_EmptyName(t *testing.T) {
	_, err := NewSettings("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewSettings_EmptyContent(t *testing.T) {
	_, err := NewSettings("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestSettings_Update(t *testing.T) {
	s, _ := NewSettings("old", `{"old":true}`)
	newName := "new"
	newContent := `{"new":true}`
	if err := s.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "new" || s.Content != `{"new":true}` {
		t.Error("update did not apply")
	}
}
