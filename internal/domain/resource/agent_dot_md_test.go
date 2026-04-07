package resource

import "testing"

func TestNewAgentDotMd(t *testing.T) {
	md, err := NewAgentDotMd("my-rules", "# Project Rules\n\nAlways use TDD.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "my-rules" {
		t.Errorf("name = %q, want %q", md.Name, "my-rules")
	}
	if md.Content != "# Project Rules\n\nAlways use TDD." {
		t.Errorf("content mismatch")
	}
	if md.ID == "" {
		t.Error("ID should be set")
	}
}

func TestNewAgentDotMd_EmptyName(t *testing.T) {
	_, err := NewAgentDotMd("", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewAgentDotMd_EmptyContent(t *testing.T) {
	_, err := NewAgentDotMd("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestAgentDotMd_Update(t *testing.T) {
	md, _ := NewAgentDotMd("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := md.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "new" || md.Content != "new content" {
		t.Error("update did not apply")
	}
}
