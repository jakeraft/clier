package terminal

import (
	"testing"
)

func TestParseRef(t *testing.T) {
	t.Run("WorkspaceRef_ExtractsRef", func(t *testing.T) {
		got, err := parseRef("Created workspace:42", "workspace:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "workspace:42" {
			t.Errorf("got %q, want %q", got, "workspace:42")
		}
	})

	t.Run("SurfaceRef_ExtractsRef", func(t *testing.T) {
		got, err := parseRef("surface:10 ready", "surface:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "surface:10" {
			t.Errorf("got %q, want %q", got, "surface:10")
		}
	})

	t.Run("MultipleRefs_ReturnsFirst", func(t *testing.T) {
		got, err := parseRef("surface:1 surface:2 surface:3", "surface:")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "surface:1" {
			t.Errorf("got %q, want %q", got, "surface:1")
		}
	})

	t.Run("PrefixNotFound_ReturnsError", func(t *testing.T) {
		_, err := parseRef("no ref here", "workspace:")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("EmptyOutput_ReturnsError", func(t *testing.T) {
		_, err := parseRef("", "surface:")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
