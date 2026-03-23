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
			p, err := NewCliProfile("my-codex", "codex-mini", []string{"--verbose"})
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

	t.Run("MatchesRawID", func(t *testing.T) {
		t.Run("ExactModel_ReturnsTrue", func(t *testing.T) {
			p, _ := NewCliProfile("test", "claude-sonnet", nil)
			if !p.MatchesRawID("claude-sonnet-4-6") {
				t.Error("expected true for exact model match")
			}
		})

		t.Run("WithDateSuffix_ReturnsTrue", func(t *testing.T) {
			p, _ := NewCliProfile("test", "claude-sonnet", nil)
			if !p.MatchesRawID("claude-sonnet-4-6-20250514") {
				t.Error("expected true for model with date suffix")
			}
		})

		t.Run("DifferentModel_ReturnsFalse", func(t *testing.T) {
			p, _ := NewCliProfile("test", "claude-sonnet", nil)
			if p.MatchesRawID("claude-opus-4-6") {
				t.Error("expected false for different model")
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
				if resolved.Binary != preset.Binary {
					t.Errorf("preset %q: Binary = %q, want %q", preset.Key, resolved.Binary, preset.Binary)
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

	t.Run("StripDateSuffix", func(t *testing.T) {
		t.Run("WithDateSuffix_StripsSuffix", func(t *testing.T) {
			got := StripDateSuffix("claude-sonnet-4-6-20250514")
			if got != "claude-sonnet-4-6" {
				t.Errorf("got %q, want %q", got, "claude-sonnet-4-6")
			}
		})

		t.Run("WithoutDateSuffix_ReturnsUnchanged", func(t *testing.T) {
			got := StripDateSuffix("claude-sonnet-4-6")
			if got != "claude-sonnet-4-6" {
				t.Errorf("got %q, want %q", got, "claude-sonnet-4-6")
			}
		})
	})
}

