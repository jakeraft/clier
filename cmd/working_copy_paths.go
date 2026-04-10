package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

type resourceTarget struct {
	Kind  string
	Owner string
	Name  string
}

func resolveCloneBase(target resourceTarget) (string, error) {
	base, err := resolveCurrentDir()
	if err != nil {
		return "", err
	}
	if copyRoot, _, err := appworkspace.FindManifestAbove(newFileMaterializer(), base); err == nil {
		return "", fmt.Errorf("clone must be run outside an existing local clone; found %s", copyRoot)
	}

	cloneDir := defaultCloneDir(base, target)
	if _, err := os.Stat(cloneDir); err == nil {
		return "", fmt.Errorf("%s clone destination already exists: %s", target.Kind, cloneDir)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat clone destination: %w", err)
	}
	return cloneDir, nil
}

func defaultCloneDir(base string, target resourceTarget) string {
	if target.Owner == "" {
		return filepath.Join(base, target.Name)
	}
	return filepath.Join(base, target.Owner, target.Name)
}

func matchesWorkingCopyTarget(target resourceTarget, copyRoot string, manifest *appworkspace.Manifest) bool {
	if manifest == nil || copyRoot == "" {
		return false
	}
	return manifest.Kind == target.Kind && manifest.Owner == target.Owner && manifest.Name == target.Name
}

func requireCurrentCopyRootKind(expectedKind, action string) (string, *appworkspace.Manifest, error) {
	base, err := resolveCurrentDir()
	if err != nil {
		return "", nil, err
	}

	copyRoot, manifest, err := appworkspace.FindManifestAbove(newFileMaterializer(), base)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("%s must be run from a local clone that owns %s", action, manifestPathLabel())
		}
		return "", nil, err
	}
	if copyRoot != base {
		return "", nil, fmt.Errorf("%s must be run from the working-copy root that owns %s", action, manifestPathLabel())
	}
	if manifest.Kind != expectedKind {
		return "", nil, fmt.Errorf("current local clone is %s/%s (%s), not a %s clone",
			manifest.Owner, manifest.Name, manifest.Kind, expectedKind)
	}
	return base, manifest, nil
}

func requireCurrentCopyRoot(target resourceTarget, action string) (string, *appworkspace.Manifest, error) {
	base, manifest, err := requireCurrentCopyRootKind(target.Kind, action)
	if err != nil {
		return "", nil, err
	}
	if !matchesWorkingCopyTarget(target, base, manifest) {
		return "", nil, fmt.Errorf("current local clone is %s/%s (%s), not %s/%s (%s)",
			manifest.Owner, manifest.Name, manifest.Kind, target.Owner, target.Name, target.Kind)
	}
	return base, manifest, nil
}
