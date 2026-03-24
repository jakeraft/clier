package settings

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestSettings(t *testing.T) {
	t.Run("CheckAuth_NoAuthDir_ReturnsNotConfigured", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		status, err := s.CheckAuth(domain.BinaryClaude)
		if err != nil {
			t.Fatalf("CheckAuth() error = %v", err)
		}
		if status != AuthNotConfigured {
			t.Errorf("CheckAuth() = %d, want AuthNotConfigured", status)
		}
	})

	t.Run("CheckAuth_UnknownBinary_ReturnsError", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		_, err := s.CheckAuth(domain.CliBinary("unknown"))
		if err == nil {
			t.Error("CheckAuth() should return error for unknown binary")
		}
	})

	t.Run("EnsureDirs_CreatesDirectories", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
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
}

func TestAuth(t *testing.T) {
	t.Run("CopyAuthTo_CopiesFiles", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
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

	t.Run("CopyAuthTo_NoAuth_ReturnsError", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.CopyAuthTo(domain.BinaryClaude, t.TempDir()); err == nil {
			t.Error("CopyAuthTo() should return error when auth not configured")
		}
	})

	t.Run("LoginAuth_UnknownBinary_ReturnsError", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.LoginAuth(domain.CliBinary("unknown")); err == nil {
			t.Error("LoginAuth() should return error for unknown binary")
		}
	})
}

func TestCredential(t *testing.T) {
	t.Run("SetAndGet_ReturnsToken", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.SetCredential("github.com", "tok123"); err != nil {
			t.Fatalf("SetCredential() error = %v", err)
		}
		got, err := s.GetCredential("github.com")
		if err != nil {
			t.Fatalf("GetCredential() error = %v", err)
		}
		if got != "tok123" {
			t.Errorf("GetCredential() = %q, want %q", got, "tok123")
		}
	})

	t.Run("Get_NonexistentHost_ReturnsError", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		_, err := s.GetCredential("missing.example.com")
		if err == nil {
			t.Error("GetCredential() for nonexistent host should return error")
		}
	})

	t.Run("Remove_DeletesCredential", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.SetCredential("gitlab.com", "tok456"); err != nil {
			t.Fatalf("SetCredential() error = %v", err)
		}
		if err := s.RemoveCredential("gitlab.com"); err != nil {
			t.Fatalf("RemoveCredential() error = %v", err)
		}
		_, err := s.GetCredential("gitlab.com")
		if err == nil {
			t.Error("GetCredential() after remove should return error")
		}
	})

	t.Run("Remove_NonexistentHost_ReturnsError", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.SetCredential("exists.com", "tok"); err != nil {
			t.Fatalf("SetCredential() error = %v", err)
		}
		if err := s.RemoveCredential("nonexistent.com"); err == nil {
			t.Error("RemoveCredential() for nonexistent host should return error")
		}
	})

	t.Run("ListHosts_ReturnsAllHosts", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		hosts := []string{"host-a.com", "host-b.com", "host-c.com"}
		for _, h := range hosts {
			if err := s.SetCredential(h, "token"); err != nil {
				t.Fatalf("SetCredential(%q) error = %v", h, err)
			}
		}
		got, err := s.ListCredentialHosts()
		if err != nil {
			t.Fatalf("ListCredentialHosts() error = %v", err)
		}
		if len(got) != len(hosts) {
			t.Fatalf("ListCredentialHosts() returned %d hosts, want %d", len(got), len(hosts))
		}
		sort.Strings(got)
		sort.Strings(hosts)
		for i := range hosts {
			if got[i] != hosts[i] {
				t.Errorf("hosts[%d] = %q, want %q", i, got[i], hosts[i])
			}
		}
	})

	t.Run("FilePermission_Is0600", func(t *testing.T) {
		s := newWithConfigDir(t.TempDir())
		if err := s.SetCredential("example.com", "secret"); err != nil {
			t.Fatalf("SetCredential() error = %v", err)
		}
		info, err := os.Stat(s.credentialsPath())
		if err != nil {
			t.Fatalf("Stat credentials file error = %v", err)
		}
		got := info.Mode().Perm()
		if got != 0600 {
			t.Errorf("credentials file mode = %04o, want 0600", got)
		}
	})
}
