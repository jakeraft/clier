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
		Paths: &Paths{home: u.HomeDir},
		Auth:  &Auth{},
	}, nil
}

// Paths resolves filesystem paths under ~/.clier.
type Paths struct {
	home string
}

func (p *Paths) Home() string {
	return p.home
}

func (p *Paths) DB() string {
	return filepath.Join(p.home, dotDir, "clier.db")
}

func (p *Paths) Workspaces() string {
	return filepath.Join(p.home, dotDir, "workspaces")
}

func (p *Paths) Dashboard() string {
	return filepath.Join(p.home, dotDir, "dashboard.html")
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

type credentialSource struct {
	keychainService string // macOS Keychain service name (empty = skip)
	filePath        string // path relative to user HOME
}

// ReadToken reads the auth token for the given binary.
// Claude: extracts the OAuth access token from keychain (macOS) or credential file.
// Codex: returns empty string (auth is independent of CODEX_HOME).
func (a *Auth) ReadToken(binary domain.CliBinary) (string, error) {
	src, ok := credentialSources[binary]
	if !ok {
		return "", nil // binary doesn't need a token (e.g. Codex)
	}

	data, err := a.readCredentials(src)
	if err != nil {
		return "", fmt.Errorf("%s is not logged in — run: %s login", binary, binary)
	}

	return extractClaudeAccessToken(data)
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
