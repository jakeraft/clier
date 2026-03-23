package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestEnv(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			e, err := NewEnv("github-token", "GITHUB_PERSONAL_ACCESS_TOKEN", "ghp_xxx")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(e.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", e.ID)
			}
			if e.Name != "github-token" {
				t.Errorf("Name = %q, want %q", e.Name, "github-token")
			}
			if e.Key != "GITHUB_PERSONAL_ACCESS_TOKEN" {
				t.Errorf("Key = %q, want %q", e.Key, "GITHUB_PERSONAL_ACCESS_TOKEN")
			}
			if e.Value != "ghp_xxx" {
				t.Errorf("Value = %q, want %q", e.Value, "ghp_xxx")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewEnv("  ", "KEY", "val")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyKey_ReturnsError", func(t *testing.T) {
			_, err := NewEnv("name", "  ", "val")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyValue_Allowed", func(t *testing.T) {
			e, err := NewEnv("empty-val", "KEY", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Value != "" {
				t.Errorf("Value = %q, want empty", e.Value)
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesNameKeyValue", func(t *testing.T) {
			e, _ := NewEnv("old", "OLD_KEY", "old_val")
			name := "new"
			key := "NEW_KEY"
			value := "new_val"
			if err := e.Update(&name, &key, &value); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Name != "new" {
				t.Errorf("Name = %q, want %q", e.Name, "new")
			}
			if e.Key != "NEW_KEY" {
				t.Errorf("Key = %q, want %q", e.Key, "NEW_KEY")
			}
			if e.Value != "new_val" {
				t.Errorf("Value = %q, want %q", e.Value, "new_val")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			e, _ := NewEnv("valid", "KEY", "val")
			name := ""
			if err := e.Update(&name, nil, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyKey_ReturnsError", func(t *testing.T) {
			e, _ := NewEnv("valid", "KEY", "val")
			key := "  "
			if err := e.Update(nil, &key, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("PartialFields_LeavesUnchangedFieldsIntact", func(t *testing.T) {
			e, _ := NewEnv("name", "KEY", "val")
			value := "changed"
			if err := e.Update(nil, nil, &value); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Name != "name" {
				t.Errorf("Name = %q, want %q", e.Name, "name")
			}
			if e.Key != "KEY" {
				t.Errorf("Key = %q, want %q", e.Key, "KEY")
			}
			if e.Value != "changed" {
				t.Errorf("Value = %q, want %q", e.Value, "changed")
			}
		})
	})
}
