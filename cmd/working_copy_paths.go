package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	apprun "github.com/jakeraft/clier/internal/app/run"
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
		return fmt.Errorf("owner name cannot start with '.': %q", owner)
	}
	return nil
}
