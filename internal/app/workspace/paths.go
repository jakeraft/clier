package workspace

import (
	"path/filepath"
	"strings"

	"github.com/jakeraft/clier/internal/domain"
)

func ResourceID(owner, name string) string {
	return owner + "/" + name
}

func SplitResourceID(id string) (owner, name string, err error) {
	parts := strings.SplitN(strings.TrimSpace(id), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", &domain.Fault{
			Kind: domain.KindInvalidArgument,
			Subject: map[string]string{
				"detail": "expected <owner/name>, got " + quoteOrEmpty(id),
			},
		}
	}
	return parts[0], parts[1], nil
}

func quoteOrEmpty(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}

// ResourceDirName returns the flat single-segment directory name for a
// resource identified by (owner, name). Owner and name are joined with
// "." so that namespaced resources project into one directory level.
// This keeps materialized layouts compatible with tools (e.g., Claude
// Code's skill scanner) that only look one level deep.
func ResourceDirName(owner, name string) string {
	owner = strings.TrimSpace(owner)
	name = sanitizeRepoDirName(name)
	if owner == "" {
		return name
	}
	return sanitizeRepoDirName(owner) + "." + name
}

func AgentWorkspacePath(base, owner, name string) string {
	return filepath.Join(base, filepath.FromSlash(AgentWorkspaceLocalPath(owner, name)))
}

func AgentWorkspaceLocalPath(owner, name string) string {
	return filepath.ToSlash(ResourceDirName(owner, name))
}

func SkillLocalPath(localBase, owner, name string) string {
	return filepath.ToSlash(filepath.Join(localBase, ResourceDirName(owner, name), "SKILL.md"))
}
