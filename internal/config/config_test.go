package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := &File{
		ServerURL:       "https://api.example.com",
		CredentialsPath: "/tmp/creds.json",
		RefsPath:        "/tmp/refs",
		WorkspacesPath:  "/tmp/workspaces",
	}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.ServerURL != want.ServerURL {
		t.Fatalf("ServerURL = %q, want %q", got.ServerURL, want.ServerURL)
	}
	if got.CredentialsPath != want.CredentialsPath {
		t.Fatalf("CredentialsPath = %q, want %q", got.CredentialsPath, want.CredentialsPath)
	}
	if got.RefsPath != want.RefsPath {
		t.Fatalf("RefsPath = %q, want %q", got.RefsPath, want.RefsPath)
	}
	if got.WorkspacesPath != want.WorkspacesPath {
		t.Fatalf("WorkspacesPath = %q, want %q", got.WorkspacesPath, want.WorkspacesPath)
	}
}

func TestLoadCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join(dotDir, "config.json")) {
		t.Errorf("DefaultPath() = %q, should end with %q", p, filepath.Join(dotDir, "config.json"))
	}
}

func TestResolveDefaults(t *testing.T) {
	got, err := Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve(nil) error: %v", err)
	}
	if got.ServerURL != DefaultServerURL {
		t.Fatalf("ServerURL = %q, want %q", got.ServerURL, DefaultServerURL)
	}
	if !strings.HasSuffix(got.CredentialsPath, filepath.Join(dotDir, "credentials.json")) {
		t.Fatalf("CredentialsPath = %q", got.CredentialsPath)
	}
	if !strings.HasSuffix(got.RefsPath, filepath.Join(dotDir, "refs")) {
		t.Fatalf("RefsPath = %q", got.RefsPath)
	}
	if !strings.HasSuffix(got.WorkspacesPath, filepath.Join(dotDir, "workspaces")) {
		t.Fatalf("WorkspacesPath = %q", got.WorkspacesPath)
	}
}

func TestResolveOverrides(t *testing.T) {
	got, err := Resolve(&File{
		ServerURL:       "https://api.example.com/",
		CredentialsPath: "~/creds.json",
		RefsPath:        "/tmp/refs",
		WorkspacesPath:  "~/work",
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ServerURL != "https://api.example.com" {
		t.Fatalf("ServerURL = %q, want %q", got.ServerURL, "https://api.example.com")
	}
	if !strings.Contains(got.CredentialsPath, "creds.json") {
		t.Fatalf("CredentialsPath = %q", got.CredentialsPath)
	}
	if got.RefsPath != "/tmp/refs" {
		t.Fatalf("RefsPath = %q, want %q", got.RefsPath, "/tmp/refs")
	}
	if !strings.Contains(got.WorkspacesPath, "work") {
		t.Fatalf("WorkspacesPath = %q", got.WorkspacesPath)
	}
}
