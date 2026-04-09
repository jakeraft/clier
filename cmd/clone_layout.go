package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	appclone "github.com/jakeraft/clier/internal/app/clone"
)

type cloneTarget struct {
	Kind  string
	Owner string
	Name  string
}

func resolveCloneCreateBase(target cloneTarget) (string, error) {
	if cloneRoot, meta, err := resolveCloneFromCWD(target.Kind); err == nil {
		if shouldReuseCloneRoot(target, cloneRoot, meta) {
			return "", fmt.Errorf("%s workspace already exists at %s", target.Kind, cloneRoot)
		}
	}

	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", err
	}
	cloneBase := defaultCloneBase(base, target)
	if _, err := os.Stat(cloneBase); err == nil {
		return "", fmt.Errorf("%s download destination already exists: %s", target.Kind, cloneBase)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat download destination: %w", err)
	}
	return cloneBase, nil
}

func defaultCloneBase(base string, target cloneTarget) string {
	switch target.Kind {
	case resourceKindMember:
		return filepath.Join(base, target.Owner, target.Name)
	case resourceKindTeam:
		return filepath.Join(base, target.Owner, target.Name)
	default:
		return filepath.Join(base, target.Name)
	}
}

func shouldReuseCloneRoot(target cloneTarget, cloneRoot string, meta *appclone.CloneMetadata) bool {
	if meta == nil || cloneRoot == "" {
		return false
	}
	return meta.Kind == target.Kind && meta.Owner == target.Owner && meta.Name == target.Name
}

func requireCurrentCloneRootKind(expectedKind, action string) (string, *appclone.CloneMetadata, error) {
	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", nil, err
	}

	if _, err := appclone.FindCloneMetadataPath(base); err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("%s must be run from the workspace root that owns %s", action, workspaceMetadataPathLabel())
		}
		return "", nil, err
	}

	meta, err := appclone.LoadCloneMetadata(base)
	if err != nil {
		return "", nil, err
	}
	if meta.Kind != expectedKind {
		return "", nil, fmt.Errorf("current workspace is %s/%s (%s), not a %s workspace",
			meta.Owner, meta.Name, meta.Kind, expectedKind)
	}
	return base, meta, nil
}

func requireCurrentCloneRoot(target cloneTarget, action string) (string, *appclone.CloneMetadata, error) {
	base, meta, err := requireCurrentCloneRootKind(target.Kind, action)
	if err != nil {
		return "", nil, err
	}
	if !shouldReuseCloneRoot(target, base, meta) {
		return "", nil, fmt.Errorf("current workspace is %s/%s (%s), not %s/%s (%s)",
			meta.Owner, meta.Name, meta.Kind, target.Owner, target.Name, target.Kind)
	}
	return base, meta, nil
}
