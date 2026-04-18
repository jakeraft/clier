package view

import (
	"time"

	appworkspace "github.com/jakeraft/clier/internal/app/workspace"
)

type CloneResult struct {
	Status  string `json:"status"`
	Kind    string `json:"kind"`
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	State   string `json:"state"`
	Version *int   `json:"version"`
}

type PullResult struct {
	Status    string               `json:"status"`
	Resources []PullResourceChange `json:"resources"`
	Kind      string               `json:"kind"`
	Owner     string               `json:"owner"`
	Name      string               `json:"name"`
	State     string               `json:"state"`
}

type PullResourceChange struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	From *int   `json:"from"`
	To   *int   `json:"to"`
}

type FetchResult struct {
	Status    string               `json:"status"`
	Resources []PullResourceChange `json:"resources"`
	Kind      string               `json:"kind"`
	Owner     string               `json:"owner"`
	Name      string               `json:"name"`
	State     string               `json:"state"`
}

type PushResult struct {
	Status string               `json:"status"`
	Pushed []PushResourceChange `json:"pushed"`
}

type RemoveResult struct {
	Status      string   `json:"status"`
	Removed     string   `json:"removed"`
	RemovedRuns []string `json:"removed_runs"`
}

type PushResourceChange struct {
	Kind   string `json:"kind"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`
	From   *int   `json:"from"`
	To     *int   `json:"to"`
	Reason string `json:"reason"`
}

type StatusResult struct {
	WorkingCopy WorkingCopyStatus `json:"working_copy"`
	Local       string            `json:"local"`
	Summary     StatusSummary     `json:"summary"`
	Tracked     []TrackedStatus   `json:"tracked"`
	Runs        RunStatusSummary  `json:"runs"`
}

type WorkingCopyStatus struct {
	Root     string    `json:"root"`
	Kind     string    `json:"kind"`
	Owner    string    `json:"owner"`
	Name     string    `json:"name"`
	ClonedAt time.Time `json:"cloned_at"`
}

type TrackedStatus struct {
	Kind          string `json:"kind"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Local         string `json:"local"`
	PinnedVersion *int   `json:"pinned_version"`
	LatestVersion *int   `json:"latest_version"`
	Remote        string `json:"remote"`
	Hint          string `json:"hint"`
}

type RunStatusSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

type StatusSummary struct {
	Modified    int `json:"modified"`
	Behind      int `json:"behind"`
	PinOutdated int `json:"pin_outdated"`
	Clean       int `json:"clean"`
}

func CloneResultOf(base string, manifest *appworkspace.Manifest) CloneResult {
	var version *int
	if manifest.RootResource.RemoteVersion != nil {
		v := *manifest.RootResource.RemoteVersion
		version = &v
	}
	return CloneResult{
		Status:  "cloned",
		Kind:    manifest.Kind,
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Dir:     base,
		State:   appworkspace.ManifestPath(base),
		Version: version,
	}
}

func PullResultOf(base string, result *appworkspace.PullResult) PullResult {
	resources := make([]PullResourceChange, 0, len(result.Resources))
	for _, resource := range result.Resources {
		resources = append(resources, PullResourceChange{
			Kind: resource.Kind,
			Name: resource.Name,
			From: resource.From,
			To:   resource.To,
		})
	}
	return PullResult{
		Status:    result.Status,
		Resources: resources,
		Kind:      result.Manifest.Kind,
		Owner:     result.Manifest.Owner,
		Name:      result.Manifest.Name,
		State:     appworkspace.ManifestPath(base),
	}
}

func FetchResultOf(base string, result *appworkspace.FetchResult) FetchResult {
	resources := make([]PullResourceChange, 0, len(result.Resources))
	for _, resource := range result.Resources {
		resources = append(resources, PullResourceChange{
			Kind: resource.Kind,
			Name: resource.Name,
			From: resource.From,
			To:   resource.To,
		})
	}
	return FetchResult{
		Status:    result.Status,
		Resources: resources,
		Kind:      result.Manifest.Kind,
		Owner:     result.Manifest.Owner,
		Name:      result.Manifest.Name,
		State:     appworkspace.ManifestPath(base),
	}
}

func PushResultOf(result *appworkspace.PushResult) PushResult {
	pushed := make([]PushResourceChange, 0, len(result.Pushed))
	for _, resource := range result.Pushed {
		pushed = append(pushed, PushResourceChange{
			Kind:   resource.Kind,
			Owner:  resource.Owner,
			Name:   resource.Name,
			From:   resource.From,
			To:     resource.To,
			Reason: resource.Reason,
		})
	}
	return PushResult{
		Status: result.Status,
		Pushed: pushed,
	}
}

func RemoveResultOf(path string, removedRuns []string) RemoveResult {
	if removedRuns == nil {
		removedRuns = []string{}
	}
	return RemoveResult{
		Status:      "removed",
		Removed:     path,
		RemovedRuns: removedRuns,
	}
}

func StatusResultOf(status *appworkspace.Status) StatusResult {
	tracked := make([]TrackedStatus, 0, len(status.Tracked))
	for _, resource := range status.Tracked {
		tracked = append(tracked, TrackedStatus{
			Kind:          resource.Kind,
			Owner:         resource.Owner,
			Name:          resource.Name,
			Path:          resource.Path,
			Local:         resource.Local,
			PinnedVersion: resource.PinnedVersion,
			LatestVersion: resource.LatestVersion,
			Remote:        resource.Remote,
			Hint:          resource.Hint,
		})
	}
	return StatusResult{
		WorkingCopy: WorkingCopyStatus{
			Root:     status.WorkingCopy.Root,
			Kind:     status.WorkingCopy.Kind,
			Owner:    status.WorkingCopy.Owner,
			Name:     status.WorkingCopy.Name,
			ClonedAt: status.WorkingCopy.ClonedAt,
		},
		Local: status.Local,
		Summary: StatusSummary{
			Modified:    status.Summary.Modified,
			Behind:      status.Summary.Behind,
			PinOutdated: status.Summary.PinOutdated,
			Clean:       status.Summary.Clean,
		},
		Tracked: tracked,
		Runs: RunStatusSummary{
			Total:   status.Runs.Total,
			Running: status.Runs.Running,
			Stopped: status.Runs.Stopped,
		},
	}
}
