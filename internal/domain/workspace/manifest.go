package workspace

import (
	"path/filepath"
	"strings"
	"time"
)

type ResourceRef struct {
	Owner   string
	Name    string
	Version int
}

type TeamProjection struct {
	Name           string
	AgentType      string
	Command        string
	GitRepoURL     string
	InstructionRef *ResourceRef
	SettingsRef    *ResourceRef
	Skills         []ResourceRef
	Children       []Child
}

type Child struct {
	Owner        string
	Name         string
	ChildVersion int
}

type TeamState struct {
	Owner      string
	Name       string
	Version    int
	LocalDir   string
	Projection TeamProjection
}

type TrackedResource struct {
	Kind          string
	AgentType     string
	Owner         string
	Name          string
	LocalPath     string
	RemoteVersion *int
	BaseHash      string
	Editable      bool
}

type Manifest struct {
	Kind             string
	Owner            string
	Name             string
	ClonedAt         time.Time
	RootResource     TrackedResource
	Teams            []TeamState
	TrackedResources []TrackedResource
	GeneratedFiles   []string
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

func (m *Manifest) FindTeam(owner, name string) (*TeamState, bool) {
	for i := range m.Teams {
		if m.Teams[i].Owner == owner && m.Teams[i].Name == name {
			return &m.Teams[i], true
		}
	}
	return nil, false
}

func (m *Manifest) FindTeamByLocalDir(localDir string) (*TeamState, bool) {
	clean := filepath.ToSlash(filepath.Clean(localDir))
	for i := range m.Teams {
		if filepath.ToSlash(filepath.Clean(m.Teams[i].LocalDir)) == clean {
			return &m.Teams[i], true
		}
	}
	return nil, false
}

func (m *Manifest) AgentForLocalPath(localPath string) (*TeamState, bool) {
	clean := filepath.ToSlash(filepath.Clean(localPath))
	for i := range m.Teams {
		if m.Teams[i].LocalDir == "" {
			continue
		}
		prefix := filepath.ToSlash(filepath.Clean(m.Teams[i].LocalDir))
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return &m.Teams[i], true
		}
	}
	return nil, false
}
