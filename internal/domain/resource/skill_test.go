package resource

import "testing"

func TestNewSkill(t *testing.T) {
	s, err := NewSkill("code-review", "Review code for quality issues")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "code-review" {
		t.Errorf("name = %q, want %q", s.Name, "code-review")
	}
	if s.Content != "Review code for quality issues" {
		t.Errorf("content mismatch")
	}
	if s.ID == "" {
		t.Error("ID should be set")
	}
}

func TestNewSkill_EmptyName(t *testing.T) {
	_, err := NewSkill("", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestNewSkill_EmptyContent(t *testing.T) {
	_, err := NewSkill("name", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestNewSkill_InvalidName(t *testing.T) {
	for _, bad := range []string{"Has Spaces", "UPPERCASE", "special!char", "under_score", ".dotfile"} {
		_, err := NewSkill(bad, "content")
		if err == nil {
			t.Errorf("expected error for invalid name %q", bad)
		}
	}
}

func TestNewSkill_ValidNames(t *testing.T) {
	for _, good := range []string{"code-review", "tdd", "my-skill-123"} {
		_, err := NewSkill(good, "content")
		if err != nil {
			t.Errorf("unexpected error for valid name %q: %v", good, err)
		}
	}
}

func TestSkill_Update(t *testing.T) {
	s, _ := NewSkill("old", "old content")
	newName := "new"
	newContent := "new content"
	if err := s.Update(&newName, &newContent); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "new" || s.Content != "new content" {
		t.Error("update did not apply")
	}
}
