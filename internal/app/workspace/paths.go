package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ResourceID(owner, name string) string {
	return owner + "/" + name
}

func SplitResourceID(id string) (owner, name string, err error) {
	parts := strings.SplitN(strings.TrimSpace(id), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid resource %q: want <owner/name>", id)
	}
	return parts[0], parts[1], nil
}

func AgentWorkspacePath(base, owner, name string) string {
	return filepath.Join(base, filepath.FromSlash(AgentWorkspaceLocalPath(owner, name)))
}

func AgentWorkspaceLocalPath(owner, name string) string {
	parts := []string{}
	owner = strings.TrimSpace(owner)
	if owner != "" {
		parts = append(parts, sanitizeRepoDirName(owner))
	}
	parts = append(parts, sanitizeRepoDirName(name))
	return filepath.ToSlash(filepath.Join(parts...))
}

func SkillLocalPath(localBase, owner, name string) string {
	return filepath.ToSlash(filepath.Join(localBase, owner, name, "SKILL.md"))
}
