package resource

import "testing"

func TestNewClaudeJson(t *testing.T) {
	cj, err := NewClaudeJson("onboarding-done", `{"hasCompletedOnboarding":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cj.Name != "onboarding-done" {
		t.Errorf("name = %q, want %q", cj.Name, "onboarding-done")
	}
	if cj.Content != `{"hasCompletedOnboarding":true}` {
		t.Errorf("content mismatch")
	}
}

func TestNewClaudeJson_EmptyName(t *testing.T) {
	_, err := NewClaudeJson("", "{}")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeJson_EmptyContent(t *testing.T) {
	_, err := NewClaudeJson("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestClaudeJson_Update(t *testing.T) {
	cj, _ := NewClaudeJson("old", `{"old":true}`)
	newName := "new"
	newContent := `{"new":true}`
	if err := cj.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cj.Name != "new" || cj.Content != `{"new":true}` {
		t.Error("update did not apply")
	}
}
