package settings

import (
	"errors"
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

	t.Run("Auth_ReturnsBinaryAuthPath", func(t *testing.T) {
		s := New("/tmp/clier")
		want := "/tmp/clier/auth/claude"
		if s.Paths.Auth(domain.BinaryClaude) != want {
			t.Errorf("Auth() = %q, want %q", s.Paths.Auth(domain.BinaryClaude), want)
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
		t.Run("NoAuthDir_ReturnsNotExist", func(t *testing.T) {
			s := New(t.TempDir())
			err := s.Auth.Check(domain.BinaryClaude)
			if err == nil {
				t.Fatal("Check() should return error when auth dir missing")
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Check() error should wrap os.ErrNotExist, got: %v", err)
			}
		})

		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			err := s.Auth.Check(domain.CliBinary("unknown"))
			if err == nil {
				t.Error("Check() should return error for unknown binary")
			}
		})
	})

	t.Run("CopyTo", func(t *testing.T) {
		t.Run("ExistingAuth_CopiesFiles", func(t *testing.T) {
			s := New(t.TempDir())
			authDir := s.Paths.Auth(domain.BinaryClaude)
			if err := os.MkdirAll(filepath.Join(authDir, ".claude"), 0755); err != nil {
				t.Fatalf("create auth dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(authDir, ".claude", "credentials.json"), []byte(`{"token":"abc"}`), 0644); err != nil {
				t.Fatalf("write auth file: %v", err)
			}

			destHome := t.TempDir()
			if err := s.Auth.CopyTo(domain.BinaryClaude, destHome); err != nil {
				t.Fatalf("CopyTo() error = %v", err)
			}

			data, err := os.ReadFile(filepath.Join(destHome, ".claude", "credentials.json"))
			if err != nil {
				t.Fatalf("read copied file: %v", err)
			}
			if string(data) != `{"token":"abc"}` {
				t.Errorf("copied content = %q, want %q", string(data), `{"token":"abc"}`)
			}
		})

		t.Run("NoAuth_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			if err := s.Auth.CopyTo(domain.BinaryClaude, t.TempDir()); err == nil {
				t.Error("CopyTo() should return error when auth not configured")
			}
		})
	})

	t.Run("Login", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			if err := s.Auth.Login(domain.CliBinary("unknown")); err == nil {
				t.Error("Login() should return error for unknown binary")
			}
		})
	})
}
