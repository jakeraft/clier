package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestCliProfile(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("ValidPreset_ResolvesPresetAndSetsFields", func(t *testing.T) {
			p, err := NewCliProfile("my-claude", "claude-sonnet", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := uuid.Parse(p.ID); err != nil {
				t.Errorf("ID %q is not a valid UUID", p.ID)
			}
			if p.Name != "my-claude" {
				t.Errorf("Name = %q, want %q", p.Name, "my-claude")
			}
			if p.Binary != BinaryClaude {
				t.Errorf("Binary = %q, want %q", p.Binary, BinaryClaude)
			}
			if p.Model != "claude-sonnet-4-6" {
				t.Errorf("Model = %q, want %q", p.Model, "claude-sonnet-4-6")
			}
			if len(p.SystemArgs) == 0 || p.SystemArgs[0] != "--dangerously-skip-permissions" {
				t.Errorf("SystemArgs = %v, want [--dangerously-skip-permissions]", p.SystemArgs)
			}
			if len(p.CustomArgs) != 0 {
				t.Errorf("CustomArgs = %v, want []", p.CustomArgs)
			}
		})

		t.Run("WithCustomArgs_SetsCustomArgs", func(t *testing.T) {
			p, err := NewCliProfile("my-claude", "claude-haiku", []string{"--verbose"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(p.CustomArgs) != 1 || p.CustomArgs[0] != "--verbose" {
				t.Errorf("CustomArgs = %v, want [--verbose]", p.CustomArgs)
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			_, err := NewCliProfile("  ", "claude-sonnet", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("UnknownPreset_ReturnsError", func(t *testing.T) {
			_, err := NewCliProfile("name", "unknown-preset", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("ValidFields_ChangesNameAndCustomArgs", func(t *testing.T) {
			p, _ := NewCliProfile("old", "claude-haiku", nil)
			name := "new"
			args := []string{"--debug"}
			if err := p.Update(&name, &args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name != "new" {
				t.Errorf("Name = %q, want %q", p.Name, "new")
			}
			if len(p.CustomArgs) != 1 || p.CustomArgs[0] != "--debug" {
				t.Errorf("CustomArgs = %v, want [--debug]", p.CustomArgs)
			}
		})

		t.Run("EmptyName_ReturnsError", func(t *testing.T) {
			p, _ := NewCliProfile("valid", "claude-haiku", nil)
			name := ""
			if err := p.Update(&name, nil); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run("NilFields_LeavesUnchanged", func(t *testing.T) {
			p, _ := NewCliProfile("keep", "claude-haiku", []string{"--flag"})
			if err := p.Update(nil, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name != "keep" {
				t.Errorf("Name = %q, want %q", p.Name, "keep")
			}
			if len(p.CustomArgs) != 1 || p.CustomArgs[0] != "--flag" {
				t.Errorf("CustomArgs = %v, want [--flag]", p.CustomArgs)
			}
		})
	})

	t.Run("NewCliProfileRaw", func(t *testing.T) {
		t.Run("ValidInputs", func(t *testing.T) {
			p, err := NewCliProfileRaw("my-profile", "claude-sonnet-4-6", BinaryClaude,
				[]string{"--dangerously-skip-permissions"}, []string{"--verbose"},
				DotConfig{"key": "val"})
			if err != nil {
				t.Fatalf("NewCliProfileRaw: %v", err)
			}
			if p.Name != "my-profile" {
				t.Errorf("Name = %q, want %q", p.Name, "my-profile")
			}
			if p.Model != "claude-sonnet-4-6" {
				t.Errorf("Model = %q, want %q", p.Model, "claude-sonnet-4-6")
			}
			if p.Binary != BinaryClaude {
				t.Errorf("Binary = %q, want %q", p.Binary, BinaryClaude)
			}
			if p.ID == "" {
				t.Error("ID should not be empty")
			}
		})

		t.Run("EmptyName", func(t *testing.T) {
			_, err := NewCliProfileRaw("", "model", BinaryClaude, nil, nil, nil)
			if err == nil {
				t.Error("expected error for empty name")
			}
		})

		t.Run("EmptyModel", func(t *testing.T) {
			_, err := NewCliProfileRaw("name", "", BinaryClaude, nil, nil, nil)
			if err == nil {
				t.Error("expected error for empty model")
			}
		})

		t.Run("InvalidBinary", func(t *testing.T) {
			_, err := NewCliProfileRaw("name", "model", CliBinary("invalid"), nil, nil, nil)
			if err == nil {
				t.Error("expected error for invalid binary")
			}
		})

		t.Run("NilSlicesDefaultToEmpty", func(t *testing.T) {
			p, err := NewCliProfileRaw("name", "model", BinaryClaude, nil, nil, nil)
			if err != nil {
				t.Fatalf("NewCliProfileRaw: %v", err)
			}
			if p.SystemArgs == nil {
				t.Error("SystemArgs should be empty slice, not nil")
			}
			if p.CustomArgs == nil {
				t.Error("CustomArgs should be empty slice, not nil")
			}
		})
	})

	t.Run("ResolvePreset", func(t *testing.T) {
		t.Run("AllKnownPresets_ResolvesCorrectly", func(t *testing.T) {
			for _, preset := range CliProfilePresets {
				resolved, err := ResolvePreset(preset.Key)
				if err != nil {
					t.Errorf("ResolvePreset(%q) error: %v", preset.Key, err)
					continue
				}
				if resolved.Model != preset.Model {
					t.Errorf("preset %q: Model = %q, want %q", preset.Key, resolved.Model, preset.Model)
				}
			}
		})

		t.Run("UnknownKey_ReturnsError", func(t *testing.T) {
			_, err := ResolvePreset("nonexistent")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

}
