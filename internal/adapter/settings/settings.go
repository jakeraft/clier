package settings

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

// Settings is the facade for all user-configurable settings.
type Settings struct {
	Paths *Paths
	Auth  *Auth
}

func New(baseDir string) *Settings {
	return &Settings{
		Paths: &Paths{base: baseDir},
		Auth:  &Auth{},
	}
}

// Paths resolves filesystem paths under the base settings directory.
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
		destPath:        ".claude/.credentials.json",
	},
	domain.BinaryCodex: {
		filePath: ".codex/auth.json",
		destPath: ".codex/auth.json",
	},
}

type credentialSource struct {
	keychainService string // macOS Keychain service name (empty = skip)
	filePath        string // path relative to user HOME
	destPath        string // path relative to dest HOME
}

func (a *Auth) CopyTo(binary domain.CliBinary, destHome string) error {
	src, ok := credentialSources[binary]
	if !ok {
		return fmt.Errorf("unknown binary: %s", binary)
	}

	data, err := a.readCredentials(src)
	if err != nil {
		return fmt.Errorf("%s is not logged in — run: %s login", binary, binary)
	}

	dest := filepath.Join(destHome, src.destPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}
	return os.WriteFile(dest, data, 0600)
}

// readCredentials tries keychain first (macOS), then falls back to file.
func (a *Auth) readCredentials(src credentialSource) ([]byte, error) {
	if src.keychainService != "" {
		if data, err := readKeychain(src.keychainService); err == nil {
			return data, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(home, src.filePath))
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
