package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecGit implements workspace.GitRepo using the git CLI.
type ExecGit struct{}

func New() *ExecGit {
	return &ExecGit{}
}

func (g *ExecGit) Clone(repoURL, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("create repo parent dir: %w", err)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, targetDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %w: %s", repoURL, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g *ExecGit) IsRepo(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(strings.ToLower(string(out)), "not a git repository") {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse --show-toplevel: %w: %s", err, strings.TrimSpace(string(out)))
	}
	topLevel := strings.TrimSpace(string(out))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("abs dir: %w", err)
	}
	topLevel, err = filepath.EvalSymlinks(topLevel)
	if err != nil {
		return false, fmt.Errorf("eval git top-level: %w", err)
	}
	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return false, fmt.Errorf("eval dir: %w", err)
	}
	return filepath.Clean(topLevel) == filepath.Clean(absDir), nil
}

func (g *ExecGit) Origin(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "config", "--get", "remote.origin.url")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git config remote.origin.url: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *ExecGit) Diff(pathA, pathB string) (string, bool, error) {
	cmd := exec.Command("git", "diff", "--no-index", "--no-color", "--", pathA, pathB)
	output, err := cmd.CombinedOutput()
	switch {
	case err == nil:
		return string(output), false, nil
	case diffExitCode(err) == 1:
		return string(output), true, nil
	default:
		return "", false, fmt.Errorf("git diff: %w", err)
	}
}

func diffExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
