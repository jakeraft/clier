package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const CloneMetadataFile = "clone.json"

type CloneMetadata struct {
	Kind          string                  `json:"kind"`
	Owner         string                  `json:"owner"`
	Name          string                  `json:"name"`
	Materializer  string                  `json:"materializer,omitempty"`
	GitRepoURL    string                  `json:"git_repo_url,omitempty"`
	RepoDir       string                  `json:"repo_dir,omitempty"`
	LatestVersion *int                    `json:"latest_version,omitempty"`
	Resources     []CloneResourceMetadata `json:"resources,omitempty"`
	ClonedAt      time.Time               `json:"cloned_at"`
}

type CloneResourceMetadata struct {
	Kind          string `json:"kind"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	GitRepoURL    string `json:"git_repo_url,omitempty"`
	LocalPath     string `json:"local_path"`
	RepoDir       string `json:"repo_dir,omitempty"`
	LatestVersion *int   `json:"latest_version,omitempty"`
}

func SaveCloneMetadata(base string, meta *CloneMetadata) error {
	dir := filepath.Join(base, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create clone metadata dir: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal clone metadata: %w", err)
	}
	path := filepath.Join(dir, CloneMetadataFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write clone metadata: %w", err)
	}
	return nil
}

func LoadCloneMetadata(base string) (*CloneMetadata, error) {
	path := filepath.Join(base, ".clier", CloneMetadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read clone metadata: %w", err)
	}

	var meta CloneMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal clone metadata: %w", err)
	}
	return &meta, nil
}
