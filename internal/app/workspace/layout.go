package workspace

import "strings"

const LegacyRepoDirName = "project"

func ResolveRepoDirName(repoURL, fallback string) string {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return sanitizeRepoDirName(fallback)
	}

	repoURL = strings.TrimSuffix(repoURL, "/")
	idx := strings.LastIndexAny(repoURL, "/:")
	name := repoURL
	if idx >= 0 && idx+1 < len(repoURL) {
		name = repoURL[idx+1:]
	}
	name = strings.TrimSuffix(name, ".git")
	name = sanitizeRepoDirName(name)
	if name != "" {
		return name
	}
	return sanitizeRepoDirName(fallback)
}

func sanitizeRepoDirName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-").Replace(name)
	name = strings.Trim(name, ".-")
	if name == "" {
		return "repo"
	}
	return name
}
