package settings

import (
	"strings"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestPaths(t *testing.T) {
	p := &Paths{home: "/Users/test"}

	t.Run("Home", func(t *testing.T) {
		if p.Home() != "/Users/test" {
			t.Errorf("Home() = %q, want %q", p.Home(), "/Users/test")
		}
	})

	t.Run("DB", func(t *testing.T) {
		want := "/Users/test/.clier/clier.db"
		if p.DB() != want {
			t.Errorf("DB() = %q, want %q", p.DB(), want)
		}
	})

	t.Run("Workspaces", func(t *testing.T) {
		want := "/Users/test/.clier/workspaces"
		if p.Workspaces() != want {
			t.Errorf("Workspaces() = %q, want %q", p.Workspaces(), want)
		}
	})

	t.Run("Dashboard", func(t *testing.T) {
		want := "/Users/test/.clier/dashboard.html"
		if p.Dashboard() != want {
			t.Errorf("Dashboard() = %q, want %q", p.Dashboard(), want)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("UsesRealHomeDir", func(t *testing.T) {
		s, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if !strings.HasSuffix(s.Paths.DB(), dotDir+"/clier.db") {
			t.Errorf("DB() = %q, should contain %q", s.Paths.DB(), dotDir)
		}
	})

	t.Run("IgnoresHOMEOverride", func(t *testing.T) {
		s1, _ := New()
		t.Setenv("HOME", "/tmp/fake-home")
		s2, _ := New()
		if s1.Paths.Home() != s2.Paths.Home() {
			t.Errorf("New() should be stable regardless of HOME: got %q vs %q", s1.Paths.Home(), s2.Paths.Home())
		}
	})
}

func TestAuth(t *testing.T) {
	auth := &Auth{}

	t.Run("Check", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			err := auth.Check(domain.CliBinary("unknown"))
			if err == nil {
				t.Error("Check() should return error for unknown binary")
			}
		})
	})

	t.Run("ReadToken", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsEmptyString", func(t *testing.T) {
			token, err := auth.ReadToken(domain.CliBinary("unknown"))
			if err != nil {
				t.Errorf("ReadToken() should not return error for unknown binary, got: %v", err)
			}
			if token != "" {
				t.Errorf("ReadToken() = %q, want empty string", token)
			}
		})

		t.Run("Codex_ReturnsEmptyString", func(t *testing.T) {
			token, err := auth.ReadToken(domain.BinaryCodex)
			if err != nil {
				t.Errorf("ReadToken(Codex) should not return error, got: %v", err)
			}
			if token != "" {
				t.Errorf("ReadToken(Codex) = %q, want empty string", token)
			}
		})

		t.Run("Claude_ReturnsToken", func(t *testing.T) {
			// This test relies on either keychain or ~/.claude/.credentials.json existing.
			// If neither is available, the test is skipped.
			token, err := auth.ReadToken(domain.BinaryClaude)
			if err != nil {
				t.Skipf("skipping: no claude credentials available: %v", err)
			}
			if token == "" {
				t.Error("ReadToken(Claude) returned empty token")
			}
		})
	})
}
