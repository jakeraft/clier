package settings

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

const (
	dbFileName  = "data.db"
	authDirName = "auth"
)

type Settings struct {
	configDir string
}

func New(configDir string) *Settings {
	return &Settings{configDir: configDir}
}

func (s *Settings) ConfigDir() string {
	return s.configDir
}

func (s *Settings) DBPath() string {
	return filepath.Join(s.configDir, dbFileName)
}

func (s *Settings) AuthDir(binary domain.CliBinary) string {
	return filepath.Join(s.configDir, authDirName, string(binary))
}

func (s *Settings) EnsureDirs() error {
	dirs := []string{
		s.configDir,
		filepath.Join(s.configDir, authDirName),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return nil
}

// Auth management

var loginCommands = map[domain.CliBinary][]string{
	domain.BinaryClaude: {"claude", "auth", "login"},
	domain.BinaryCodex:  {"codex", "login"},
}

var statusCommands = map[domain.CliBinary][]string{
	domain.BinaryClaude: {"claude", "auth", "status"},
	domain.BinaryCodex:  {"codex", "login", "status"},
}

type AuthStatus int

const (
	AuthOK AuthStatus = iota
	AuthNotConfigured
	AuthInvalid
)

func (s *Settings) CheckAuth(binary domain.CliBinary) (AuthStatus, error) {
	args, ok := statusCommands[binary]
	if !ok {
		return 0, fmt.Errorf("unknown binary: %s", binary)
	}

	authDir := s.AuthDir(binary)
	if _, err := os.Stat(authDir); err != nil {
		return AuthNotConfigured, nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(systemEnv(), "HOME="+authDir)
	if err := cmd.Run(); err != nil {
		return AuthInvalid, nil
	}
	return AuthOK, nil
}

func (s *Settings) LoginAuth(binary domain.CliBinary) error {
	args, ok := loginCommands[binary]
	if !ok {
		return fmt.Errorf("unknown binary: %s", binary)
	}

	authDir := s.AuthDir(binary)
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
	return nil
}

func (s *Settings) CopyAuthTo(binary domain.CliBinary, destHome string) error {
	authDir := s.AuthDir(binary)
	if _, err := os.Stat(authDir); err != nil {
		return fmt.Errorf("auth not configured for %s — run: clier %s login", binary, binary)
	}

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
