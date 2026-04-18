package cmd

import (
	"path/filepath"
	"strings"

	apprun "github.com/jakeraft/clier/internal/app/run"
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
	"github.com/jakeraft/clier/internal/domain"
)

// workspaceDir returns the resolved workspace root from config.
func workspaceDir() (string, error) {
	cfg, err := currentConfig()
	if err != nil {
		return "", err
	}
	return cfg.WorkspaceDir, nil
}

// runsDir returns the central directory holding all run plans.
func runsDir() (string, error) {
	root, err := workspaceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, apprun.RunsDirName), nil
}

// workingCopyPath returns the canonical absolute path for a team's working copy.
func workingCopyPath(owner, name string) (string, error) {
	root, err := workspaceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appworkspace.ResourceDirName(owner, name)), nil
}

// validateOwner rejects owner names that would collide with internal
// workspace subdirectories (anything starting with '.', e.g., ".runs").
func validateOwner(owner string) error {
	if strings.HasPrefix(owner, ".") {
		return &domain.Fault{
			Kind:    domain.KindInvalidArgument,
			Subject: map[string]string{"detail": "owner name cannot start with '.': " + owner},
		}
	}
	return nil
}
