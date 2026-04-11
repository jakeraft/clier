package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const ManifestFile = "manifest.json"

type Manifest struct {
	Kind             string            `json:"kind"`
	Owner            string            `json:"owner"`
	Name             string            `json:"name"`
	ClonedAt         time.Time         `json:"cloned_at"`
	Upstream         *UpstreamMetadata `json:"upstream,omitempty"`
	RootResource     TrackedResource   `json:"root_resource"`
	TrackedResources []TrackedResource `json:"tracked_resources,omitempty"`
	GeneratedFiles   []string          `json:"generated_files,omitempty"`
	Runtime          *RuntimeMetadata  `json:"runtime,omitempty"`
}

type UpstreamMetadata struct {
	Kind           string     `json:"kind"`
	Owner          string     `json:"owner"`
	Name           string     `json:"name"`
	FetchedVersion *int       `json:"fetched_version,omitempty"`
	FetchedAt      *time.Time `json:"fetched_at,omitempty"`
}

type TrackedResource struct {
	Kind          string `json:"kind"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	LocalPath     string `json:"local_path"`
	RemoteVersion *int   `json:"remote_version,omitempty"`
	BaseHash      string `json:"base_hash,omitempty"`
	Editable      bool   `json:"editable"`
}

type RuntimeMetadata struct {
	Member *MemberRuntimeMetadata `json:"member,omitempty"`
	Team   *TeamRuntimeMetadata   `json:"team,omitempty"`
}

type MemberRuntimeMetadata struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AgentType  string `json:"agent_type"`
	Command    string `json:"command"`
	GitRepoURL string `json:"git_repo_url,omitempty"`
}

type TeamRuntimeMetadata struct {
	ID      int64                       `json:"id"`
	Name    string                      `json:"name"`
	Members []TeamMemberRuntimeMetadata `json:"members"`
}

type TeamMemberRuntimeMetadata struct {
	TeamMemberID int64  `json:"team_member_id"`
	Name         string `json:"name"`
	AgentType    string `json:"agent_type"`
	Command      string `json:"command"`
	GitRepoURL   string `json:"git_repo_url,omitempty"`
}

func ManifestPath(base string) string {
	return filepath.Join(base, ".clier", ManifestFile)
}

func FindManifestPath(fs FileMaterializer, base string) (string, error) {
	path := ManifestPath(base)
	if _, err := fs.Stat(path); err == nil {
		return path, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat working-copy manifest: %w", err)
	}
	return "", os.ErrNotExist
}

func FindManifestAbove(fs FileMaterializer, start string) (string, *Manifest, error) {
	base, err := filepath.Abs(start)
	if err != nil {
		return "", nil, fmt.Errorf("resolve working-copy base: %w", err)
	}
	for dir := base; ; dir = filepath.Dir(dir) {
		if _, err := FindManifestPath(fs, dir); err == nil {
			manifest, loadErr := LoadManifest(fs, dir)
			if loadErr != nil {
				return "", nil, loadErr
			}
			return dir, manifest, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", nil, os.ErrNotExist
}

func SaveManifest(fs FileMaterializer, base string, manifest *Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := fs.EnsureFile(ManifestPath(base), data); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func LoadManifest(fs FileMaterializer, base string) (*Manifest, error) {
	path, err := FindManifestPath(fs, base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("read manifest: %w", err)
		}
		return nil, err
	}
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &manifest, nil
}

func (m *Manifest) FindTrackedResource(localPath string) (*TrackedResource, bool) {
	clean := filepath.ToSlash(filepath.Clean(localPath))
	for i := range m.TrackedResources {
		if filepath.ToSlash(filepath.Clean(m.TrackedResources[i].LocalPath)) == clean {
			return &m.TrackedResources[i], true
		}
	}
	return nil, false
}
