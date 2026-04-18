package workspace

import (
	"path"
	"path/filepath"
	"strconv"
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

func SplitVersionedResourceID(id string) (owner, name string, version *int, err error) {
	raw := strings.TrimSpace(id)
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		owner, name, err = SplitResourceID(raw)
		return owner, name, nil, err
	}
	if at == 0 || at == len(raw)-1 {
		return "", "", nil, &domain.Fault{
			Kind: domain.KindInvalidArgument,
			Subject: map[string]string{
				"detail": "expected <owner/name>@<version>, got " + quoteOrEmpty(id),
			},
		}
	}

	owner, name, err = SplitResourceID(strings.TrimSpace(raw[:at]))
	if err != nil {
		return "", "", nil, err
	}
	parsed, convErr := strconv.Atoi(strings.TrimSpace(raw[at+1:]))
	if convErr != nil || parsed <= 0 {
		return "", "", nil, &domain.Fault{
			Kind: domain.KindInvalidArgument,
			Subject: map[string]string{
				"detail": "expected positive version in " + quoteOrEmpty(id),
			},
		}
	}
	return owner, name, intPtr(parsed), nil
}

func quoteOrEmpty(s string) string {
	if s == "" {
		return `""`
	}
	return `"` + s + `"`
}

// ResourceDirName returns the flat single-segment local key for a
// resource identified by (owner, name). Owner and name are joined with
// "." so that namespaced resources project into one directory level.
// This is the single source of truth for any local-only unique key
// derived from a resource ID — directories, manifest entries, generated
// filenames — so on-disk layout stays consistent.
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

// SkillLocalPath returns the slash-joined path for storage in
// TrackedResource.LocalPath. Use SkillFilePath for native filesystem
// operations.
func SkillLocalPath(localBase, owner, name string) string {
	return path.Join(localBase, ResourceDirName(owner, name), "SKILL.md")
}

// SkillFilePath returns the native OS path for materializing a skill on disk.
func SkillFilePath(skillsDir, owner, name string) string {
	return filepath.Join(skillsDir, ResourceDirName(owner, name), "SKILL.md")
}
