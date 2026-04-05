package resource

import (
	"testing"

	"github.com/google/uuid"
)

func TestGitRepo(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			r, err := NewGitRepo("my-repo", "https://github.com/org/repo")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(r.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", r.ID)
			}
			if r.Name != "my-repo" {
				t.Errorf("Name = %q, want %q", r.Name, "my-repo")
			}
			if r.URL != "https://github.com/org/repo" {
				t.Errorf("URL = %q, want %q", r.URL, "https://github.com/org/repo")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewGitRepo("", "https://example.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyURL_ReturnsError", func(t *testing.T) {
			_, err := NewGitRepo("name", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesNameAndURL", func(t *testing.T) {
			r, _ := NewGitRepo("old", "https://old.com")
			name, url := "new", "https://new.com"
			if err := r.Update(&name, &url); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.Name != "new" {
				t.Errorf("Name = %q, want %q", r.Name, "new")
			}
			if r.URL != "https://new.com" {
				t.Errorf("URL = %q, want %q", r.URL, "https://new.com")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			r, _ := NewGitRepo("valid", "https://example.com")
			name := ""
			if err := r.Update(&name, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyURL_ReturnsError", func(t *testing.T) {
			r, _ := NewGitRepo("valid", "https://example.com")
			url := "  "
			if err := r.Update(nil, &url); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
