package workspace

import (
	"fmt"
	"os"
)

func ensureRepoDir(fs FileMaterializer, git GitRepo, repoURL, repoDir string) error {
	if repoURL == "" {
		return fs.MkdirAll(repoDir)
	}

	info, err := fs.Stat(repoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return git.Clone(repoURL, repoDir)
		}
		return fmt.Errorf("stat repo dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo path %s exists and is not a directory", repoDir)
	}

	entries, err := fs.ReadDir(repoDir)
	if err != nil {
		return fmt.Errorf("read repo dir: %w", err)
	}
	if len(entries) == 0 {
		return git.Clone(repoURL, repoDir)
	}

	isRepo, err := git.IsRepo(repoDir)
	if err != nil {
		return fmt.Errorf("check git repo: %w", err)
	}
	if isRepo {
		originURL, err := git.Origin(repoDir)
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

func IsMaterializedRoot(fs FileMaterializer, git GitRepo, repoURL, root string) (bool, error) {
	info, err := fs.Stat(root)
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
		return git.IsRepo(root)
	}

	entries, err := fs.ReadDir(root)
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
