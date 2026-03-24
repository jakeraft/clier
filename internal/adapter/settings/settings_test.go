package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestSettings(t *testing.T) {
	t.Run("CheckAuth", func(t *testing.T) {
		t.Run("NoAuthDir_ReturnsNotExist", func(t *testing.T) {
			s := New(t.TempDir())
			err := s.CheckAuth(domain.BinaryClaude)
			if err == nil {
				t.Fatal("CheckAuth() should return error when auth dir missing")
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Errorf("CheckAuth() error should wrap os.ErrNotExist, got: %v", err)
			}
		})

		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			err := s.CheckAuth(domain.CliBinary("unknown"))
			if err == nil {
				t.Error("CheckAuth() should return error for unknown binary")
			}
		})
	})

	t.Run("EnsureDirs", func(t *testing.T) {
		t.Run("ValidConfig_CreatesDirectories", func(t *testing.T) {
			s := New(t.TempDir())
			if err := s.EnsureDirs(); err != nil {
				t.Fatalf("EnsureDirs() error = %v", err)
			}
			for _, dir := range []string{s.ConfigDir(), filepath.Join(s.ConfigDir(), authDirName)} {
				info, err := os.Stat(dir)
				if err != nil {
					t.Errorf("dir %q not created: %v", dir, err)
					continue
				}
				if !info.IsDir() {
					t.Errorf("path %q is not a directory", dir)
				}
			}
		})
	})

	t.Run("CopyAuthTo", func(t *testing.T) {
		t.Run("ExistingAuth_CopiesFiles", func(t *testing.T) {
			s := New(t.TempDir())
			authDir := s.AuthDir(domain.BinaryClaude)
			if err := os.MkdirAll(filepath.Join(authDir, ".claude"), 0755); err != nil {
				t.Fatalf("create auth dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(authDir, ".claude", "credentials.json"), []byte(`{"token":"abc"}`), 0644); err != nil {
				t.Fatalf("write auth file: %v", err)
			}

			destHome := t.TempDir()
			if err := s.CopyAuthTo(domain.BinaryClaude, destHome); err != nil {
				t.Fatalf("CopyAuthTo() error = %v", err)
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
			if err := s.CopyAuthTo(domain.BinaryClaude, t.TempDir()); err == nil {
				t.Error("CopyAuthTo() should return error when auth not configured")
			}
		})
	})

	t.Run("LoginAuth", func(t *testing.T) {
		t.Run("UnknownBinary_ReturnsError", func(t *testing.T) {
			s := New(t.TempDir())
			if err := s.LoginAuth(domain.CliBinary("unknown")); err == nil {
				t.Error("LoginAuth() should return error for unknown binary")
			}
		})
	})
}
