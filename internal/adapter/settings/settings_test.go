package settings

import (
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	p := &Paths{base: "/Users/test/.clier"}

	t.Run("Base", func(t *testing.T) {
		if p.Base() != "/Users/test/.clier" {
			t.Errorf("Base() = %q, want %q", p.Base(), "/Users/test/.clier")
		}
	})

	t.Run("Workspaces", func(t *testing.T) {
		want := "/Users/test/.clier/workspaces"
		if p.Workspaces() != want {
			t.Errorf("Workspaces() = %q, want %q", p.Workspaces(), want)
		}
	})

	t.Run("ExpandTilde", func(t *testing.T) {
		got := p.ExpandTilde("~/Documents/file.txt")
		want := "/Users/test/Documents/file.txt"
		if got != want {
			t.Errorf("ExpandTilde() = %q, want %q", got, want)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("UsesRealHomeDir", func(t *testing.T) {
		s, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if !strings.HasSuffix(s.Paths.Base(), dotDir) {
			t.Errorf("Base() = %q, should end with %q", s.Paths.Base(), dotDir)
		}
	})

	t.Run("IgnoresHOMEOverride", func(t *testing.T) {
		s1, _ := New()
		t.Setenv("HOME", "/tmp/fake-home")
		s2, _ := New()
		if s1.Paths.Base() != s2.Paths.Base() {
			t.Errorf("New() should be stable regardless of HOME: got %q vs %q", s1.Paths.Base(), s2.Paths.Base())
		}
	})
}
