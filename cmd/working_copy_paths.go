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
