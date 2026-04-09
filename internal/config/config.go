package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const dotDir = ".clier"
const DefaultServerURL = "http://localhost:8080"

// File stores user-configurable CLI settings.
type File struct {
	ServerURL       string `json:"server_url,omitempty"`
	CredentialsPath string `json:"credentials_path,omitempty"`
	RefsPath        string `json:"refs_path,omitempty"`
	WorkspacesPath  string `json:"workspaces_path,omitempty"`
}

func defaultBaseDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("get current user: %w", err)
	}
	return filepath.Join(u.HomeDir, dotDir), nil
}

// DefaultPath returns the default ~/.clier/config.json location.
func DefaultPath() (string, error) {
	base, err := defaultBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "config.json"), nil
}

// Resolve fills missing config values with defaults and normalizes paths.
func Resolve(cfg *File) (*File, error) {
	base, err := defaultBaseDir()
	if err != nil {
		return nil, err
	}

	out := &File{
		ServerURL:       DefaultServerURL,
		CredentialsPath: filepath.Join(base, "credentials.json"),
		RefsPath:        filepath.Join(base, "refs"),
		WorkspacesPath:  filepath.Join(base, "workspaces"),
	}
	if cfg == nil {
		return out, nil
	}

	if cfg.ServerURL != "" {
		out.ServerURL = normalizeServerURL(cfg.ServerURL)
	}
	if cfg.CredentialsPath != "" {
		out.CredentialsPath = expandTilde(base, cfg.CredentialsPath)
	}
	if cfg.RefsPath != "" {
		out.RefsPath = expandTilde(base, cfg.RefsPath)
	}
	if cfg.WorkspacesPath != "" {
		out.WorkspacesPath = expandTilde(base, cfg.WorkspacesPath)
	}

	return out, nil
}

// Load reads config.json from the given path.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("corrupted config file: %w", err)
	}

	return &cfg, nil
}

// Save writes config.json to the given path with 0600 permission.
func Save(path string, cfg *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func expandTilde(baseDir, s string) string {
	return strings.ReplaceAll(s, "~/", filepath.Dir(baseDir)+"/")
}

func normalizeServerURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}
