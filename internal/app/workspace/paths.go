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
