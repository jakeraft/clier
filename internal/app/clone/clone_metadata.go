package clone

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	WorkspaceMetadataFile   = "workspace.json"
	LegacyCloneMetadataFile = "clone.json"
	CloneMetadataFile       = WorkspaceMetadataFile
)

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
	Workspace     *WorkspaceMetadata      `json:"workspace,omitempty"`
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

type WorkspaceMetadata struct {
	Member *MemberWorkspaceMetadata `json:"member,omitempty"`
	Team   *TeamWorkspaceMetadata   `json:"team,omitempty"`
}

type MemberWorkspaceMetadata struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	GitRepoURL string `json:"git_repo_url,omitempty"`
}

type TeamWorkspaceMetadata struct {
	ID      int64                         `json:"id"`
	Name    string                        `json:"name"`
	Members []TeamMemberWorkspaceMetadata `json:"members"`
}

type TeamMemberWorkspaceMetadata struct {
	TeamMemberID int64  `json:"team_member_id"`
	Name         string `json:"name"`
	Command      string `json:"command"`
	GitRepoURL   string `json:"git_repo_url,omitempty"`
}

func MetadataPath(base string) string {
	return filepath.Join(base, ".clier", WorkspaceMetadataFile)
}

func legacyMetadataPath(base string) string {
	return filepath.Join(base, ".clier", LegacyCloneMetadataFile)
}

func FindCloneMetadataPath(base string) (string, error) {
	candidates := []string{MetadataPath(base), legacyMetadataPath(base)}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat workspace metadata: %w", err)
		}
	}
	return "", os.ErrNotExist
}

func SaveCloneMetadata(base string, meta *CloneMetadata) error {
	dir := filepath.Join(base, ".clier")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create workspace metadata dir: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspace metadata: %w", err)
	}
	path := MetadataPath(base)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write workspace metadata: %w", err)
	}
	return nil
}

func LoadCloneMetadata(base string) (*CloneMetadata, error) {
	path, err := FindCloneMetadataPath(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("read workspace metadata: %w", err)
		}
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workspace metadata: %w", err)
	}

	var meta CloneMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal workspace metadata: %w", err)
	}
	return &meta, nil
}
