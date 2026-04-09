package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

type workspaceTarget struct {
	Kind  string
	Owner string
	Name  string
}

func resolveWorkspaceCreateBase(target workspaceTarget) (string, error) {
	if workspaceRoot, meta, err := resolveWorkspaceFromCWD(target.Kind); err == nil {
		if shouldReuseWorkspaceRoot(target, workspaceRoot, meta) {
			return "", fmt.Errorf("%s workspace already exists at %s", target.Kind, workspaceRoot)
		}
	}

	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", err
	}
	workspaceBase := defaultWorkspaceBase(base, target)
	if _, err := os.Stat(workspaceBase); err == nil {
		return "", fmt.Errorf("%s download destination already exists: %s", target.Kind, workspaceBase)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat download destination: %w", err)
	}
	return workspaceBase, nil
}

func defaultWorkspaceBase(base string, target workspaceTarget) string {
	switch target.Kind {
	case resourceKindMember:
		return filepath.Join(base, target.Owner, target.Name)
	case resourceKindTeam:
		return filepath.Join(base, target.Owner, target.Name)
	default:
		return filepath.Join(base, target.Name)
	}
}

func shouldReuseWorkspaceRoot(target workspaceTarget, workspaceRoot string, meta *appworkspace.Manifest) bool {
	if meta == nil || workspaceRoot == "" {
		return false
	}
	return meta.Kind == target.Kind && meta.Owner == target.Owner && meta.Name == target.Name
}

func requireCurrentWorkspaceRootKind(expectedKind, action string) (string, *appworkspace.Manifest, error) {
	base, err := resolveWorkspaceBase()
	if err != nil {
		return "", nil, err
	}

	if _, err := appworkspace.FindManifestPath(base); err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("%s must be run from the workspace root that owns %s", action, workspaceMetadataPathLabel())
		}
		return "", nil, err
	}

	meta, err := appworkspace.LoadManifest(base)
	if err != nil {
		return "", nil, err
	}
	if meta.Kind != expectedKind {
		return "", nil, fmt.Errorf("current workspace is %s/%s (%s), not a %s workspace",
			meta.Owner, meta.Name, meta.Kind, expectedKind)
	}
	return base, meta, nil
}

func requireCurrentWorkspaceRoot(target workspaceTarget, action string) (string, *appworkspace.Manifest, error) {
	base, meta, err := requireCurrentWorkspaceRootKind(target.Kind, action)
	if err != nil {
		return "", nil, err
	}
	if !shouldReuseWorkspaceRoot(target, base, meta) {
		return "", nil, fmt.Errorf("current workspace is %s/%s (%s), not %s/%s (%s)",
			meta.Owner, meta.Name, meta.Kind, target.Owner, target.Name, target.Kind)
	}
	return base, meta, nil
}
