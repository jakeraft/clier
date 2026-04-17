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
	if !strings.HasSuffix(got.WorkspaceDir, filepath.Join(dotDir, "workspace")) {
		t.Fatalf("WorkspaceDir = %q, should end with %q", got.WorkspaceDir, filepath.Join(dotDir, "workspace"))
	}
}

func TestResolveOverrides(t *testing.T) {
	got, err := Resolve(&File{
		ServerURL:       "https://api.example.com/",
		CredentialsPath: "/tmp/creds.json",
		WorkspaceDir:    "/tmp/clier-workspaces",
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ServerURL != "https://api.example.com" {
		t.Fatalf("ServerURL = %q, want %q", got.ServerURL, "https://api.example.com")
	}
	if got.CredentialsPath != "/tmp/creds.json" {
		t.Fatalf("CredentialsPath = %q, want %q", got.CredentialsPath, "/tmp/creds.json")
	}
	if got.WorkspaceDir != "/tmp/clier-workspaces" {
		t.Fatalf("WorkspaceDir = %q, want %q", got.WorkspaceDir, "/tmp/clier-workspaces")
	}
}
