package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// Settings is the facade for all user-configurable settings.
type Settings struct {
	Paths *Paths
	Auth  *Auth
}

const dotDir = ".clier"

func New() (*Settings, error) {
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	return &Settings{
		Paths: &Paths{base: filepath.Join(u.HomeDir, dotDir)},
		Auth:  &Auth{},
	}, nil
}

// Paths resolves filesystem paths under ~/.clier.
type Paths struct {
	base string
}

func (p *Paths) Base() string {
	return p.base
}

func (p *Paths) DB() string {
	return filepath.Join(p.base, "clier.db")
}

func (p *Paths) Workspaces() string {
	return filepath.Join(p.base, "workspaces")
}

func (p *Paths) Dashboard() string {
	return filepath.Join(p.base, "dashboard.html")
}

func (p *Paths) HomeDir() string {
	return filepath.Dir(p.base)
}

// ExpandTilde replaces ~/ prefixes with the parent of the base directory (OS home).
func (p *Paths) ExpandTilde(s string) string {
	return strings.ReplaceAll(s, "~/", filepath.Dir(p.base)+"/")
}

// Auth manages Claude CLI authentication.
// It does not store credentials itself — it reads from the user's
// actual CLI auth (Keychain on macOS, credential files on Linux).
type Auth struct{}

// Check verifies that Claude CLI is authenticated.
func (a *Auth) Check() error {
	cmd := exec.Command("claude", "auth", "status")
	cmd.Env = systemEnv()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude auth check failed — run: claude login: %w", err)
	}
	return nil
}

// ReadToken extracts the OAuth access token from keychain (macOS) or credential file.
func (a *Auth) ReadToken() (string, error) {
	data, err := a.readCredentials()
	if err != nil {
		return "", errors.New("claude is not logged in — run: claude login")
	}
	return extractAccessToken(data)
}

// extractAccessToken extracts the accessToken from Claude credential JSON.
// Credential format: {"claudeAiOauth":{"accessToken":"sk-ant-...", ...}}
func extractAccessToken(data []byte) (string, error) {
	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", fmt.Errorf("parse claude credentials: %w", err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", errors.New("claude credentials missing accessToken")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

// readCredentials tries keychain first (macOS), then falls back to file.
func (a *Auth) readCredentials() ([]byte, error) {
	if data, err := readKeychain("Claude Code-credentials"); err == nil {
		return data, nil
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(u.HomeDir, ".claude/.credentials.json"))
}

func readKeychain(service string) ([]byte, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return nil, err
	}
	data := bytes.TrimSpace(out)
	if len(data) == 0 {
		return nil, errors.New("empty keychain entry")
	}
	return data, nil
}

func systemEnv() []string {
	return os.Environ()
}
