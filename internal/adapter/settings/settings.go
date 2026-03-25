package settings

import (
	"bytes"
	"fmt"
	"io/fs"
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
	paths := &Paths{base: baseDir}
	return &Settings{
		Paths: paths,
		Auth:  &Auth{paths: paths},
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

func (p *Paths) Auth(binary domain.CliBinary) string {
	return filepath.Join(p.base, "auth", string(binary))
}

func (p *Paths) Workspaces() string {
	return filepath.Join(p.base, "workspaces")
}

// Auth manages CLI authentication for agent binaries.
type Auth struct {
	paths *Paths
}

var loginCommands = map[domain.CliBinary][]string{
	domain.BinaryClaude: {"claude", "auth", "login"},
	domain.BinaryCodex:  {"codex", "login"},
}

var statusCommands = map[domain.CliBinary][]string{
	domain.BinaryClaude: {"claude", "auth", "status"},
	domain.BinaryCodex:  {"codex", "login", "status"},
}

func (a *Auth) Check(binary domain.CliBinary) error {
	args, ok := statusCommands[binary]
	if !ok {
		return fmt.Errorf("unknown binary: %s", binary)
	}

	authDir := a.paths.Auth(binary)
	if _, err := os.Stat(authDir); err != nil {
		return fmt.Errorf("check auth dir: %w", err)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(systemEnv(), "HOME="+authDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s status: %w", args[0], err)
	}
	return nil
}

func (a *Auth) Login(binary domain.CliBinary) error {
	args, ok := loginCommands[binary]
	if !ok {
		return fmt.Errorf("unknown binary: %s", binary)
	}

	authDir := a.paths.Auth(binary)
	if err := os.MkdirAll(authDir, 0755); err != nil {
		return fmt.Errorf("create auth dir: %w", err)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(systemEnv(), "HOME="+authDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("login %s: %w", binary, err)
	}

	a.syncFromKeychain(binary)
	return nil
}

func (a *Auth) CopyTo(binary domain.CliBinary, destHome string) error {
	authDir := a.paths.Auth(binary)
	if _, err := os.Stat(authDir); err != nil {
		return fmt.Errorf("auth not configured for %s — run: clier %s login", binary, binary)
	}

	a.syncFromKeychain(binary)

	return filepath.WalkDir(authDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(authDir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destHome, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode().Perm())
	})
}

// keychainServices maps binaries to their macOS Keychain service names.
var keychainServices = map[domain.CliBinary]string{
	domain.BinaryClaude: "Claude Code-credentials",
}

// syncFromKeychain updates the stored credentials file with the latest
// token from macOS Keychain, if available and fresher. Best-effort: errors
// are silently ignored so file-based auth still works as fallback.
func (a *Auth) syncFromKeychain(binary domain.CliBinary) {
	service, ok := keychainServices[binary]
	if !ok {
		return
	}

	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return
	}

	keychainData := bytes.TrimSpace(out)
	if len(keychainData) == 0 {
		return
	}

	credPath := filepath.Join(a.paths.Auth(binary), ".claude", ".credentials.json")
	if err := os.MkdirAll(filepath.Dir(credPath), 0755); err != nil {
		return
	}
	_ = os.WriteFile(credPath, keychainData, 0600)
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
