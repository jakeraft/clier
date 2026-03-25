package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestPaths(t *testing.T) {
	t.Run("Base_ReturnsBaseDir", func(t *testing.T) {
		s := New("/tmp/clier")
		if s.Paths.Base() != "/tmp/clier" {
			t.Errorf("Base() = %q, want %q", s.Paths.Base(), "/tmp/clier")
		}
	})

	t.Run("DB_ReturnsDBPath", func(t *testing.T) {
		s := New("/tmp/clier")
		want := "/tmp/clier/clier.db"
		if s.Paths.DB() != want {
			t.Errorf("DB() = %q, want %q", s.Paths.DB(), want)
		}
	})

	t.Run("Workspaces_ReturnsWorkspacesPath", func(t *testing.T) {
		s := New("/tmp/clier")
		want := "/tmp/clier/workspaces"
		if s.Paths.Workspaces() != want {
			t.Errorf("Workspaces() = %q, want %q", s.Paths.Workspaces(), want)
		}
	})
}

func TestAuth(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			err := s.Auth.Check(domain.CliBinary("unknown"))
			if err == nil {
				t.Error("Check() should return error for unknown binary")
			}
		})
	})

	t.Run("CopyTo", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			if err := s.Auth.CopyTo(domain.CliBinary("unknown"), t.TempDir()); err == nil {
				t.Error("CopyTo() should return error for unknown binary")
			}
		})

		t.Run("Claude_CopiesToDest", func(t *testing.T) {
			s := New(t.TempDir())
			destHome := t.TempDir()

			// This test relies on either keychain or ~/.claude/.credentials.json existing.
			// If neither is available, the test is skipped.
			err := s.Auth.CopyTo(domain.BinaryClaude, destHome)
			if err != nil {
				t.Skipf("skipping: no claude credentials available: %v", err)
			}

			credPath := filepath.Join(destHome, ".claude", ".credentials.json")
			data, err := os.ReadFile(credPath)
			if err != nil {
				t.Fatalf("read copied credentials: %v", err)
			}
			if len(data) == 0 {
				t.Error("copied credentials file is empty")
			}
		})
	})
}
