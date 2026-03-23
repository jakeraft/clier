package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestSystemPrompt(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidInputs_GeneratesUUIDAndSetsFields", func(t *testing.T) {
			s, err := NewSystemPrompt("setup", "echo hello")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(s.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", s.ID)
			}
			if s.Name != "setup" {
				t.Errorf("Name = %q, want %q", s.Name, "setup")
			}
			if s.Prompt != "echo hello" {
				t.Errorf("Prompt = %q, want %q", s.Prompt, "echo hello")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewSystemPrompt("  ", "prompt")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyPrompt_ReturnsError", func(t *testing.T) {
			_, err := NewSystemPrompt("name", "  ")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesNameAndPrompt", func(t *testing.T) {
			s, _ := NewSystemPrompt("old", "old prompt")
			name := "new"
			prompt := "new prompt"
			if err := s.Update(&name, &prompt); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Name != "new" {
				t.Errorf("Name = %q, want %q", s.Name, "new")
			}
			if s.Prompt != "new prompt" {
				t.Errorf("Prompt = %q, want %q", s.Prompt, "new prompt")
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			s, _ := NewSystemPrompt("valid", "prompt")
			name := ""
			if err := s.Update(&name, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("EmptyPrompt_ReturnsError", func(t *testing.T) {
			s, _ := NewSystemPrompt("valid", "prompt")
			prompt := "  "
			if err := s.Update(nil, &prompt); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("PartialFields_LeavesUnchangedFieldsIntact", func(t *testing.T) {
			s, _ := NewSystemPrompt("name", "prompt")
			prompt := "changed"
			if err := s.Update(nil, &prompt); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Name != "name" {
				t.Errorf("Name = %q, want %q", s.Name, "name")
			}
			if s.Prompt != "changed" {
				t.Errorf("Prompt = %q, want %q", s.Prompt, "changed")
			}
		})
	})
}
