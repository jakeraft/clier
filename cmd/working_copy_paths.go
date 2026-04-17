package cmd

import (
	"path/filepath"
	"strings"

	apprun "github.com/jakeraft/clier/internal/app/run"
	"github.com/jakeraft/clier/internal/domain"
)

// workspaceDir returns the resolved workspace root from config.
func workspaceDir() string {
	return currentConfig().WorkspaceDir
}

// runsDir returns the central directory holding all run plans.
func runsDir() string {
	return filepath.Join(workspaceDir(), apprun.RunsDirName)
}

// workingCopyPath returns the canonical absolute path for a team's working copy.
func workingCopyPath(owner, name string) string {
	if owner == "" {
		return filepath.Join(workspaceDir(), name)
	}
	return filepath.Join(workspaceDir(), owner, name)
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
