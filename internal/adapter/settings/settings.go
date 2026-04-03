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

	"github.com/jakeraft/clier/internal/domain"
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

// Auth manages CLI authentication for agent binaries.
// It does not store credentials itself — it reads from the user's
// actual CLI auth (Keychain on macOS, credential files on Linux).
type Auth struct{}

var statusCommands = map[domain.CliBinary][]string{
	domain.BinaryClaude: {"claude", "auth", "status"},
	domain.BinaryCodex:  {"codex", "login", "status"},
}

func (a *Auth) Check(binary domain.CliBinary) error {
	args, ok := statusCommands[binary]
	if !ok {
		return fmt.Errorf("unknown binary: %s", binary)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = systemEnv()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s is not logged in — run: %s login", binary, binary)
	}
	return nil
}

// credentialSources defines where to find credentials for each binary.
// Tried in order: keychain first (macOS), then file fallback.
var credentialSources = map[domain.CliBinary]credentialSource{
	domain.BinaryClaude: {
		keychainService: "Claude Code-credentials",
		filePath:        ".claude/.credentials.json",
	},
}

// authFilePaths maps binaries that use file-based auth to their credential file
// path relative to $HOME. Binaries using token-based auth (e.g. Claude) are absent.
var authFilePaths = map[domain.CliBinary]string{
	domain.BinaryCodex: ".codex/auth.json",
}

type credentialSource struct {
	keychainService string // macOS Keychain service name (empty = skip)
	filePath        string // path relative to user HOME
}

// ReadToken reads the env-based auth token for the given binary.
// Claude: extracts the OAuth access token from keychain (macOS) or credential file.
// Codex: returns empty string (uses file-based auth via ReadAuthFile).
func (a *Auth) ReadToken(binary domain.CliBinary) (string, error) {
	src, ok := credentialSources[binary]
	if !ok {
		return "", nil // binary uses file-based auth, not token (e.g. Codex)
	}

	data, err := a.readCredentials(src)
	if err != nil {
		return "", fmt.Errorf("%s is not logged in — run: %s login", binary, binary)
	}

	return extractClaudeAccessToken(data)
}

// ReadAuthFile reads the raw auth file for binaries that use file-based auth.
// Codex: returns the content of ~/.codex/auth.json.
// Claude: returns nil (uses token-based auth via ReadToken).
func (a *Auth) ReadAuthFile(binary domain.CliBinary) ([]byte, error) {
	relPath, ok := authFilePaths[binary]
	if !ok {
		return nil, nil
	}
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(u.HomeDir, relPath))
}

// extractClaudeAccessToken extracts the accessToken from Claude credential JSON.
// Credential format: {"claudeAiOauth":{"accessToken":"sk-ant-...", ...}}
func extractClaudeAccessToken(data []byte) (string, error) {
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
func (a *Auth) readCredentials(src credentialSource) ([]byte, error) {
	if src.keychainService != "" {
		if data, err := readKeychain(src.keychainService); err == nil {
			return data, nil
		}
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(u.HomeDir, src.filePath))
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
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") {
			env = append(env, "PATH="+systemPath())
			continue
		}
		env = append(env, e)
	}
	return env
}

func systemPath() string {
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if !strings.Contains(d, "cmux") {
			dirs = append(dirs, d)
		}
	}
	return strings.Join(dirs, string(filepath.ListSeparator))
}
