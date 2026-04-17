package workspace

import (
	"errors"
	"fmt"
	"os"

	"github.com/jakeraft/clier/internal/domain"
)

func repoConflict(path, detail string) *domain.Fault {
	return &domain.Fault{
		Kind: domain.KindRepoDirConflict,
		Subject: map[string]string{
			"path":   path,
			"detail": detail,
		},
	}
}

func ensureRepoDir(fs FileMaterializer, git GitRepo, repoURL, repoDir string) error {
	if repoURL == "" {
		return fs.MkdirAll(repoDir)
	}

	info, err := fs.Stat(repoDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return git.Clone(repoURL, repoDir)
		}
		return fmt.Errorf("stat repo dir: %w", err)
	}
	if !info.IsDir() {
		return repoConflict(repoDir, "path exists but is not a directory")
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
			return repoConflict(repoDir, "already tracks "+originURL+", not "+repoURL)
		}
		return nil
	}

	return repoConflict(repoDir, "directory exists but is not a git repo")
}

func IsMaterializedRoot(fs FileMaterializer, git GitRepo, repoURL, root string) (bool, error) {
	info, err := fs.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
