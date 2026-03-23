package terminal

import (
	"testing"
)

func TestShellQuote(t *testing.T) {
	t.Run("Empty_ReturnsSingleQuotes", func(t *testing.T) {
		got := ShellQuote("")
		if got != "''" {
			t.Errorf("got %q, want %q", got, "''")
		}
	})

	t.Run("Simple_WrapsInSingleQuotes", func(t *testing.T) {
		got := ShellQuote("hello")
		if got != "'hello'" {
			t.Errorf("got %q, want %q", got, "'hello'")
		}
	})

	t.Run("WithSpaces_PreservesSpaces", func(t *testing.T) {
		got := ShellQuote("hello world")
		if got != "'hello world'" {
			t.Errorf("got %q, want %q", got, "'hello world'")
		}
	})

	t.Run("WithSingleQuote_EscapesQuote", func(t *testing.T) {
		got := ShellQuote("it's")
		want := `'it'\''s'`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("WithMultipleSingleQuotes_EscapesAll", func(t *testing.T) {
		got := ShellQuote("a'b'c")
		want := `'a'\''b'\''c'`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("WithSpecialChars_NoEscaping", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			in   string
			want string
		}{
			{"DoubleQuote", `say "hi"`, `'say "hi"'`},
			{"Dollar", "$HOME", "'$HOME'"},
			{"Backtick", "`cmd`", "'`cmd`'"},
			{"Backslash", `a\b`, `'a\b'`},
			{"Newline", "a\nb", "'a\nb'"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got := ShellQuote(tc.in)
				if got != tc.want {
					t.Errorf("ShellQuote(%q) = %q, want %q", tc.in, got, tc.want)
				}
			})
		}
	})
}

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
