package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestEnvironment(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			e, err := NewEnvironment("prod", "API_KEY", "secret")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(e.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", e.ID)
			}
			if e.Name != "prod" {
				t.Errorf("Name = %q, want %q", e.Name, "prod")
			}
			if e.Key != "API_KEY" {
				t.Errorf("Key = %q, want %q", e.Key, "API_KEY")
			}
			if e.Value != "secret" {
				t.Errorf("Value = %q, want %q", e.Value, "secret")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewEnvironment("", "KEY", "val")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyKey_ReturnsError", func(t *testing.T) {
			_, err := NewEnvironment("name", "  ", "val")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyValue_ReturnsError", func(t *testing.T) {
			_, err := NewEnvironment("name", "KEY", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesAllFields", func(t *testing.T) {
			e, _ := NewEnvironment("old", "OLD_KEY", "old_val")
			name, key, value := "new", "NEW_KEY", "new_val"
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
			e, _ := NewEnvironment("valid", "KEY", "val")
			name := ""
			if err := e.Update(&name, nil, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyKey_ReturnsError", func(t *testing.T) {
			e, _ := NewEnvironment("valid", "KEY", "val")
			key := "  "
			if err := e.Update(nil, &key, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyValue_ReturnsError", func(t *testing.T) {
			e, _ := NewEnvironment("valid", "KEY", "val")
			value := "  "
			if err := e.Update(nil, nil, &value); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})
}
