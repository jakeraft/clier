package resource

import "testing"

func TestNewClaudeMd(t *testing.T) {
	md, err := NewClaudeMd("my-rules", "# Project Rules\n\nAlways use TDD.")
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

func TestNewClaudeMd_EmptyName(t *testing.T) {
	_, err := NewClaudeMd("", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewClaudeMd_EmptyContent(t *testing.T) {
	_, err := NewClaudeMd("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestClaudeMd_Update(t *testing.T) {
	md, _ := NewClaudeMd("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := md.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md.Name != "new" || md.Content != "new content" {
		t.Error("update did not apply")
	}
}
