package cmd

import (
	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

func cloneResultPayload(base string, manifest *appworkspace.Manifest) map[string]any {
	result := map[string]any{
		"status": "cloned",
		"kind":   manifest.Kind,
		"owner":  manifest.Owner,
		"name":   manifest.Name,
		"dir":    base,
		"state":  appworkspace.ManifestPath(base),
	}
	if manifest.RootResource.RemoteVersion != nil {
		result["version"] = *manifest.RootResource.RemoteVersion
	}
	return result
}

func pullResultPayload(base string, result *appworkspace.PullResult) map[string]any {
	return map[string]any{
		"status":    result.Status,
		"resources": result.Resources,
		"kind":      result.Manifest.Kind,
		"owner":     result.Manifest.Owner,
		"name":      result.Manifest.Name,
		"state":     appworkspace.ManifestPath(base),
	}
}

func fetchResultPayload(base string, result *appworkspace.FetchResult) map[string]any {
	return map[string]any{
		"status":    result.Status,
		"resources": result.Resources,
		"kind":      result.Manifest.Kind,
		"owner":     result.Manifest.Owner,
		"name":      result.Manifest.Name,
		"state":     appworkspace.ManifestPath(base),
	}
}

func pushResultPayload(result *appworkspace.PushResult) map[string]any {
	return map[string]any{
		"status": result.Status,
		"pushed": result.Pushed,
	}
}
