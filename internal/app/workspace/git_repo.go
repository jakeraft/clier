package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ensureRepoDir(repoURL, repoDir string) error {
	if repoURL == "" {
		return os.MkdirAll(repoDir, 0755)
	}

	info, err := os.Stat(repoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return gitClone(repoURL, repoDir)
		}
		return fmt.Errorf("stat repo dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo path %s exists and is not a directory", repoDir)
	}

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return fmt.Errorf("read repo dir: %w", err)
	}
	if len(entries) == 0 {
		return gitClone(repoURL, repoDir)
	}

	isRepo, err := isGitRepo(repoDir)
	if err != nil {
		return fmt.Errorf("check git repo: %w", err)
	}
	if isRepo {
		originURL, err := gitOrigin(repoDir)
		if err != nil {
			return fmt.Errorf("read git origin: %w", err)
		}
		if originURL != repoURL {
			return fmt.Errorf("repo dir %s already tracks %s, not %s", repoDir, originURL, repoURL)
		}
		return nil
	}

	return fmt.Errorf("repo dir %s already exists and is not a git repo", repoDir)
}

func IsMaterializedRoot(repoURL, root string) (bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return false, nil
	}

	if repoURL != "" {
		return isGitRepo(root)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return false, fmt.Errorf("read root: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != ".clier" {
			return true, nil
		}
	}
	return false, nil
}

func gitClone(repoURL, repoDir string) error {
	if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
		return fmt.Errorf("create repo parent dir: %w", err)
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone %s: %w: %s", repoURL, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitOrigin(repoDir string) (string, error) {
	cmd := exec.Command("git", "-C", repoDir, "config", "--get", "remote.origin.url")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git config remote.origin.url: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func isGitRepo(repoDir string) (bool, error) {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(strings.ToLower(string(out)), "not a git repository") {
			return false, nil
		}
		return false, fmt.Errorf("git rev-parse --show-toplevel: %w: %s", err, strings.TrimSpace(string(out)))
	}
	topLevel := strings.TrimSpace(string(out))
	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return false, fmt.Errorf("abs repo dir: %w", err)
	}
	topLevel, err = filepath.EvalSymlinks(topLevel)
	if err != nil {
		return false, fmt.Errorf("eval git top-level: %w", err)
	}
	absRepoDir, err = filepath.EvalSymlinks(absRepoDir)
	if err != nil {
		return false, fmt.Errorf("eval repo dir: %w", err)
	}
	return filepath.Clean(topLevel) == filepath.Clean(absRepoDir), nil
}
